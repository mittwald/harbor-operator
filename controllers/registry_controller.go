/*


Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"fmt"
	"net/url"
	"reflect"
	"time"

	h "github.com/mittwald/goharbor-client/v3/apiv2"
	legacymodel "github.com/mittwald/goharbor-client/v3/apiv2/model/legacy"
	registryapi "github.com/mittwald/goharbor-client/v3/apiv2/registry"
	"github.com/mittwald/harbor-operator/controllers/helper"
	"github.com/mittwald/harbor-operator/controllers/internal"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	registriesv1alpha1 "github.com/mittwald/harbor-operator/api/v1alpha1"
)

// RegistryReconciler reconciles a Registry object
type RegistryReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

func (r *RegistryReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// Create a new controller
	c, err := controller.New("registry-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource Registry
	err = c.Watch(&source.Kind{Type: &registriesv1alpha1.Registry{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	return nil
}

// +kubebuilder:rbac:groups=registries.mittwald.de,resources=registries,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=registries.mittwald.de,resources=registries/status,verbs=get;update;patch
func (r *RegistryReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	reqLogger := r.Log.WithValues("registry", req.NamespacedName)
	reqLogger.Info("Reconciling Registry")

	now := metav1.Now()
	ctx := context.Background()

	// Fetch the Registry instance
	registry := &registriesv1alpha1.Registry{}

	err := r.Client.Get(ctx, req.NamespacedName, registry)
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return ctrl.Result{}, err
	}

	originalRegistry := registry.DeepCopy()

	if registry.ObjectMeta.DeletionTimestamp != nil &&
		registry.Status.Phase != registriesv1alpha1.RegistryStatusPhaseTerminating {
		registry.Status = registriesv1alpha1.RegistryStatus{Phase: registriesv1alpha1.RegistryStatusPhaseTerminating}

		return r.updateRegistryCR(ctx, nil, originalRegistry, registry)
	}

	// Fetch the Instance
	harbor, err := internal.FetchReadyHarborInstance(ctx, registry.Namespace, registry.Spec.ParentInstance.Name, r.Client)
	if err != nil {
		if _, ok := err.(internal.ErrInstanceNotFound); ok {
			helper.PullFinalizer(registry, internal.FinalizerName)
		} else if _, ok := err.(internal.ErrInstanceNotReady); ok {
			return ctrl.Result{RequeueAfter: 30 * time.Second}, err
		} else {
			registry.Status = registriesv1alpha1.RegistryStatus{LastTransition: &now}
		}

		return r.updateRegistryCR(ctx, nil, originalRegistry, registry)
	}

	// Build a client to connect to the harbor API
	harborClient, err := internal.BuildClient(ctx, r, harbor)
	if err != nil {
		return ctrl.Result{Requeue: true}, err
	}

	switch registry.Status.Phase {
	default:
		return ctrl.Result{}, nil

	case registriesv1alpha1.RegistryStatusPhaseUnknown:
		registry.Status = registriesv1alpha1.RegistryStatus{Phase: registriesv1alpha1.RegistryStatusPhaseCreating}

	case registriesv1alpha1.RegistryStatusPhaseCreating:
		if err := r.assertExistingRegistry(ctx, harborClient, registry); err != nil {
			return ctrl.Result{}, err
		}
		helper.PushFinalizer(registry, internal.FinalizerName)

		registry.Status = registriesv1alpha1.RegistryStatus{Phase: registriesv1alpha1.RegistryStatusPhaseReady}
	case registriesv1alpha1.RegistryStatusPhaseReady:
		err := r.assertExistingRegistry(ctx, harborClient, registry)
		if err != nil {
			return ctrl.Result{}, err
		}

	case registriesv1alpha1.RegistryStatusPhaseTerminating:
		// Delete the registry via harbor API
		err := r.assertDeletedRegistry(ctx, reqLogger, harborClient, registry)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	return r.updateRegistryCR(ctx, harbor, originalRegistry, registry)
}

// updateRegistryCR compares the new CR status and finalizers with the pre-existing ones and updates them accordingly.
func (r *RegistryReconciler) updateRegistryCR(ctx context.Context, parentInstance *registriesv1alpha1.Instance, originalRegistry, registry *registriesv1alpha1.Registry) (ctrl.Result, error) {
	if originalRegistry == nil || registry == nil {
		return ctrl.Result{}, fmt.Errorf("cannot update registry '%s' because the original registry is nil", registry.Spec.Name)
	}

	// Update Status
	if !reflect.DeepEqual(originalRegistry.Status, registry.Status) {
		originalRegistry.Status = registry.Status
		if err := r.Status().Update(ctx, originalRegistry); err != nil {
			return ctrl.Result{}, err
		}
	}

	// set owner
	if (len(originalRegistry.OwnerReferences) == 0) && parentInstance != nil {
		err := ctrl.SetControllerReference(parentInstance, originalRegistry, r.Scheme)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	// Update Finalizer
	if !reflect.DeepEqual(originalRegistry.Finalizers, registry.Finalizers) {
		originalRegistry.SetFinalizers(registry.Finalizers)
	}

	if err := r.Update(ctx, originalRegistry); err != nil {
		return ctrl.Result{Requeue: true}, err
	}

	return ctrl.Result{}, nil
}

// assertExistingRegistry checks a harbor registry for existence and creates it accordingly.
func (r *RegistryReconciler) assertExistingRegistry(ctx context.Context, harborClient *h.RESTClient,
	originalRegistry *registriesv1alpha1.Registry) error {
	_, err := harborClient.GetRegistry(ctx, originalRegistry.Spec.Name)
	if err != nil {
		switch err.Error() {
		case registryapi.ErrRegistryNotFoundMsg:
			rReq, err := r.buildRegistryFromCR(ctx, originalRegistry)
			if err != nil {
				return err
			}

			_, err = harborClient.NewRegistry(
				ctx,
				rReq.Name,
				rReq.Type,
				rReq.URL,
				rReq.Credential,
				rReq.Insecure,
			)

			if err != nil {
				return err
			}
		default:
			return err
		}
	}

	return r.ensureRegistry(ctx, harborClient, originalRegistry)
}

func parseURL(raw string) (string, error) {
	var parsed *url.URL

	parsed, err := url.ParseRequestURI(raw)
	if err != nil {
		return "", err
	}

	return parsed.String(), nil
}

// enumRegistryType enumerates a string against valid GarbageCollection types.
func enumRegistryType(receivedRegistryType registriesv1alpha1.RegistryType) (registriesv1alpha1.RegistryType, error) {
	switch receivedRegistryType {
	case
		registriesv1alpha1.RegistryTypeHarbor,
		registriesv1alpha1.RegistryTypeDockerHub,
		registriesv1alpha1.RegistryTypeDockerRegistry,
		registriesv1alpha1.RegistryTypeHuaweiSWR,
		registriesv1alpha1.RegistryTypeGoogleGCR,
		registriesv1alpha1.RegistryTypeAwsECR,
		registriesv1alpha1.RegistryTypeAzureECR,
		registriesv1alpha1.RegistryTypeAliACR,
		registriesv1alpha1.RegistryTypeJfrogArtifactory,
		registriesv1alpha1.RegistryTypeQuayIo,
		registriesv1alpha1.RegistryTypeGitlab,
		registriesv1alpha1.RegistryTypeHelmHub:
		return receivedRegistryType, nil
	default:
		return "", fmt.Errorf("invalid garbage collection schedule type provided: '%s'", receivedRegistryType)
	}
}

// ensureRegistry gets and compares the spec of the registry held by the harbor API with the spec of the existing CR.
func (r *RegistryReconciler) ensureRegistry(ctx context.Context, harborClient *h.RESTClient,
	originalRegistry *registriesv1alpha1.Registry) error {
	// Get the registry held by harbor
	heldRegistry, err := harborClient.GetRegistry(ctx, originalRegistry.Spec.Name)
	if err != nil {
		return err
	}

	// Use the registry's ID from the Harbor instance and write it back to the the CR's status field
	if originalRegistry.Status.ID != heldRegistry.ID {
		originalRegistry.Status.ID = heldRegistry.ID

		if err := r.Client.Status().Update(ctx, originalRegistry); err != nil {
			return err
		}
	}

	// Construct a registry from the CR spec
	newReg, err := r.buildRegistryFromCR(ctx, originalRegistry)
	if err != nil {
		return err
	}

	if newReg.Credential == nil {
		newReg.Credential = &legacymodel.RegistryCredential{}
	}
	// Compare the registries and update accordingly
	if !reflect.DeepEqual(heldRegistry, newReg) {
		return r.updateRegistry(ctx, harborClient, newReg)
	}

	return nil
}

// updateRegistry triggers the update of a registry.
func (r *RegistryReconciler) updateRegistry(ctx context.Context, harborClient *h.RESTClient,
	reg *legacymodel.Registry) error {
	return harborClient.UpdateRegistry(ctx, reg)
}

// buildRegistryFromCR constructs and returns a Harbor registry object from the CR object's spec.
func (r *RegistryReconciler) buildRegistryFromCR(ctx context.Context, originalRegistry *registriesv1alpha1.Registry) (*legacymodel.Registry,
	error) {
	parsedURL, err := parseURL(originalRegistry.Spec.URL)
	if err != nil {
		return nil, err
	}

	registryType, err := enumRegistryType(originalRegistry.Spec.Type)
	if err != nil {
		return nil, err
	}

	var credential *legacymodel.RegistryCredential
	if originalRegistry.Spec.Credential != nil {
		credential, err = originalRegistry.Spec.Credential.ToHarborRegistryCredential(ctx, r.Client, originalRegistry.Namespace)
		if err != nil {
			return nil, err
		}
	}

	return &legacymodel.Registry{
		ID:          originalRegistry.Status.ID,
		Name:        originalRegistry.Spec.Name,
		Description: originalRegistry.Spec.Description,
		Type:        string(registryType),
		URL:         parsedURL,
		Credential:  credential,
		Insecure:    originalRegistry.Spec.Insecure,
	}, nil
}

// assertDeletedRegistry deletes a registry, first ensuring its existence.
func (r *RegistryReconciler) assertDeletedRegistry(ctx context.Context, log logr.Logger, harborClient *h.RESTClient,
	registry *registriesv1alpha1.Registry) error {
	reg, err := harborClient.GetRegistry(ctx, registry.Name)
	if err != nil {
		return err
	}

	if reg != nil {
		err := harborClient.DeleteRegistry(ctx, reg)
		if err != nil {
			return err
		}
		log.Info("pulling finalizers")
		helper.PullFinalizer(registry, internal.FinalizerName)
	}

	return nil
}
