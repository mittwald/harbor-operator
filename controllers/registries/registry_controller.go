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

package registries

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"reflect"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/go-logr/logr"
	h "github.com/mittwald/goharbor-client/v4/apiv2"
	legacymodel "github.com/mittwald/goharbor-client/v4/apiv2/model/legacy"
	registryapi "github.com/mittwald/goharbor-client/v4/apiv2/registry"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/mittwald/harbor-operator/apis/registries/v1alpha2"

	"github.com/mittwald/harbor-operator/controllers/registries/helper"
	"github.com/mittwald/harbor-operator/controllers/registries/internal"
)

// RegistryReconciler reconciles a Registry object
type RegistryReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=registries.mittwald.de,resources=registries,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=registries.mittwald.de,resources=registries/status,verbs=get;update;patch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *RegistryReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	reqLogger := r.Log.WithValues("registry", req.NamespacedName)
	reqLogger.Info("Reconciling Registry")

	// Fetch the Registry instance
	registry := &v1alpha2.Registry{}

	err := r.Client.Get(ctx, req.NamespacedName, registry)
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return ctrl.Result{}, err
	}

	patch := client.MergeFrom(registry.DeepCopy())

	if registry.ObjectMeta.DeletionTimestamp != nil &&
		registry.Status.Phase != v1alpha2.RegistryStatusPhaseTerminating {
		registry.Status = v1alpha2.RegistryStatus{Phase: v1alpha2.RegistryStatusPhaseTerminating}

		return ctrl.Result{}, r.Client.Status().Patch(ctx, registry, patch)
	}

	// Fetch the goharbor instance if it exists and is properly set up.
	// If the above does not apply, pull the finalizer from the registry object.
	harbor, err := internal.GetOperationalHarborInstance(ctx, client.ObjectKey{
		Namespace: registry.Namespace,
		Name:      registry.Spec.ParentInstance.Name,
	}, r.Client)
	if err != nil {
		controllerutil.RemoveFinalizer(registry, internal.FinalizerName)
		return ctrl.Result{}, r.Client.Status().Patch(ctx, registry, patch)
	}

	// Build a client to connect to the harbor API
	harborClient, err := internal.BuildClient(ctx, r.Client, harbor)
	if err != nil {
		return ctrl.Result{}, err
	}

	// Check the Harbor API if it's reporting as healthy
	instanceIsHealthy, err := internal.HarborInstanceIsHealthy(ctx, harborClient)
	if err != nil {
		return ctrl.Result{}, err
	}
	if !instanceIsHealthy {
		return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
	}

	switch registry.Status.Phase {
	default:
		return ctrl.Result{}, nil

	case v1alpha2.RegistryStatusPhaseUnknown:
		registry.Status.Phase = v1alpha2.RegistryStatusPhaseCreating
		registry.Status.Message = "registry is about to be created"

		return ctrl.Result{}, r.Client.Status().Patch(ctx, registry, patch)
	case v1alpha2.RegistryStatusPhaseCreating:
		if err := r.assertExistingRegistry(ctx, harborClient, registry, patch); err != nil {
			return ctrl.Result{}, err
		}
		controllerutil.AddFinalizer(registry, internal.FinalizerName)

		registry.Status = v1alpha2.RegistryStatus{Phase: v1alpha2.RegistryStatusPhaseReady}

		return ctrl.Result{}, r.Client.Status().Patch(ctx, registry, patch)
	case v1alpha2.RegistryStatusPhaseReady:
		if !controllerutil.ContainsFinalizer(registry, internal.FinalizerName) {
			controllerutil.AddFinalizer(registry, internal.FinalizerName)
			err := r.Client.Patch(ctx, registry, patch)
			if err != nil {
				return ctrl.Result{}, err
			}
		}

		err := r.assertExistingRegistry(ctx, harborClient, registry, patch)
		if err != nil {
			return ctrl.Result{}, err
		}

	case v1alpha2.RegistryStatusPhaseTerminating:
		// Delete the registry via harbor API
		err := r.assertDeletedRegistry(ctx, reqLogger, harborClient, registry, patch)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

// assertExistingRegistry checks a harbor registry for existence and creates it accordingly.
func (r *RegistryReconciler) assertExistingRegistry(ctx context.Context, harborClient *h.RESTClient,
	registry *v1alpha2.Registry, patch client.Patch) error {
	_, err := harborClient.GetRegistry(ctx, registry.Spec.Name)
	if err != nil {
		switch err.Error() {
		case registryapi.ErrRegistryNotFoundMsg:
			rReq, err := r.buildRegistryFromCR(ctx, registry)
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

	// Get the registry held by harbor
	heldRegistry, err := harborClient.GetRegistry(ctx, registry.Spec.Name)
	if err != nil {
		return err
	}

	// Use the registry's ID from the Harbor instance and write it back to the CR's status field
	if registry.Status.ID != heldRegistry.ID {
		registry.Status.ID = heldRegistry.ID

		if err := r.Client.Status().Patch(ctx, registry, patch); err != nil {
			return err
		}
	}

	// Construct a registry from the CR spec
	newReg, err := r.buildRegistryFromCR(ctx, registry)
	if err != nil {
		return err
	}

	if newReg.Credential == nil {
		newReg.Credential = &legacymodel.RegistryCredential{}
	}
	// Compare the registries and update accordingly
	if !reflect.DeepEqual(heldRegistry, newReg) {
		return harborClient.UpdateRegistry(ctx, newReg)
	}

	return nil
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
func enumRegistryType(receivedRegistryType v1alpha2.RegistryType) (v1alpha2.RegistryType, error) {
	switch receivedRegistryType {
	case
		v1alpha2.RegistryTypeHarbor,
		v1alpha2.RegistryTypeDockerHub,
		v1alpha2.RegistryTypeDockerRegistry,
		v1alpha2.RegistryTypeHuaweiSWR,
		v1alpha2.RegistryTypeGoogleGCR,
		v1alpha2.RegistryTypeAwsECR,
		v1alpha2.RegistryTypeAzureECR,
		v1alpha2.RegistryTypeAliACR,
		v1alpha2.RegistryTypeJfrogArtifactory,
		v1alpha2.RegistryTypeQuay,
		v1alpha2.RegistryTypeGitlab,
		v1alpha2.RegistryTypeHelmHub:
		return receivedRegistryType, nil
	default:
		return "", fmt.Errorf("invalid registry type provided: '%s'", receivedRegistryType)
	}
}

// buildRegistryFromCR constructs and returns a Harbor registry object from the CR object's spec.
func (r *RegistryReconciler) buildRegistryFromCR(ctx context.Context, registry *v1alpha2.Registry) (*legacymodel.Registry,
	error) {
	parsedURL, err := parseURL(registry.Spec.URL)
	if err != nil {
		return nil, err
	}

	registryType, err := enumRegistryType(registry.Spec.Type)
	if err != nil {
		return nil, err
	}

	var credential *legacymodel.RegistryCredential
	if registry.Spec.Credential != nil {
		credential, err = helper.ToHarborRegistryCredential(ctx, r.Client, registry.Namespace, *registry.Spec.Credential)
		if err != nil {
			return nil, err
		}
	}

	return &legacymodel.Registry{
		ID:          registry.Status.ID,
		Name:        registry.Spec.Name,
		Description: registry.Spec.Description,
		Type:        string(registryType),
		URL:         parsedURL,
		Credential:  credential,
		Insecure:    registry.Spec.Insecure,
	}, nil
}

// assertDeletedRegistry deletes a registry, first ensuring all controlled replications are deleted.
func (r *RegistryReconciler) assertDeletedRegistry(ctx context.Context, log logr.Logger, harborClient *h.RESTClient,
	registry *v1alpha2.Registry, patch client.Patch) error {

	// List all replications that reference the parent registry resource and mark them for deletion.
	replicationList := v1alpha2.ReplicationList{}

	if err := r.Client.List(ctx, &replicationList, client.MatchingFields{"metadata.ownerReferences.uid": string(registry.UID)}); err != nil {
		return err
	}

	for _, i := range replicationList.Items {
		patch := client.MergeFrom(i.DeepCopy())
		err := internal.AssertDeletedReplication(ctx, log, harborClient, &i)
		if err != nil {
			return err
		}

		if err := r.Client.Delete(ctx, &i); err != nil {
			return err
		}

		if err := r.Client.Patch(ctx, &i, patch); err != nil {
			return err
		}
	}

	reg, err := harborClient.GetRegistry(ctx, registry.Name)
	if err != nil {
		if errors.Is(err, &registryapi.ErrRegistryNotFound{}) {
			log.Info("registry does not exist on the server side, pulling finalizers")
			controllerutil.RemoveFinalizer(registry, internal.FinalizerName)
			if err := r.Client.Patch(ctx, registry, patch); err != nil {
				return err
			}
		}
		return err
	}

	if reg != nil {
		err := harborClient.DeleteRegistry(ctx, reg)
		if err != nil {
			return err
		}
		log.Info("pulling finalizer")
		controllerutil.RemoveFinalizer(registry, internal.FinalizerName)
		if err := r.Client.Patch(ctx, registry, patch); err != nil {
			return err
		}
	}

	return nil
}

func (r *RegistryReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := mgr.GetFieldIndexer().IndexField(
		context.Background(), &v1alpha2.Replication{},
		"metadata.ownerReferences.uid",
		func(obj client.Object) []string {
			m, ok := obj.(metav1.Object)
			if ok {
				o := m.GetOwnerReferences()
				if len(o) == 0 {
					return []string{}
				}
				return []string{string(o[0].UID)}
			}
			return []string{}
		}); err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha2.Registry{}).
		Owns(&corev1.Secret{}).
		Complete(r)
}
