package registry

import (
	"context"
	"fmt"
	"net/url"
	"reflect"
	"time"

	modelv1 "github.com/mittwald/goharbor-client/model/v1_10_0"

	controllerruntime "sigs.k8s.io/controller-runtime"

	"github.com/go-logr/logr"
	h "github.com/mittwald/goharbor-client"
	"github.com/mittwald/harbor-operator/pkg/controller/internal"
	"github.com/mittwald/harbor-operator/pkg/internal/helper"

	registriesv1alpha1 "github.com/mittwald/harbor-operator/pkg/apis/registries/v1alpha1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const FinalizerName = "harbor-operator.registries.mittwald.de"

var log = logf.Log.WithName("controller_registry")

// Add creates a new Registry Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler.
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileRegistry{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler.
func add(mgr manager.Manager, r reconcile.Reconciler) error {
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

// blank assignment to verify that ReconcileRegistry implements reconcile.Reconciler.
var _ reconcile.Reconciler = &ReconcileRegistry{}

// ReconcileRegistry reconciles a Registry object.
type ReconcileRegistry struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a Registry object and makes changes based on the state read
// and what is in the Registry.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileRegistry) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling Registry")

	now := metav1.Now()
	ctx := context.Background()

	var result reconcile.Result

	// Fetch the Registry instance
	registry := &registriesv1alpha1.Registry{}

	err := r.client.Get(ctx, request.NamespacedName, registry)
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	originalRegistry := registry.DeepCopy()

	if registry.ObjectMeta.DeletionTimestamp != nil &&
		registry.Status.Phase != registriesv1alpha1.RegistryStatusPhaseTerminating {
		registry.Status = registriesv1alpha1.RegistryStatus{Phase: registriesv1alpha1.RegistryStatusPhaseTerminating}
		result = reconcile.Result{Requeue: true}

		return r.updateRegistryCR(ctx, nil, originalRegistry, registry, result)
	}

	// Fetch the Instance
	harbor, err := internal.FetchReadyHarborInstance(ctx, registry.Namespace, registry.Spec.ParentInstance.Name, r.client)
	if err != nil {
		if _, ok := err.(internal.ErrInstanceNotFound); ok {
			helper.PullFinalizer(registry, FinalizerName)

			result = reconcile.Result{}
		} else if _, ok := err.(internal.ErrInstanceNotReady); ok {
			return reconcile.Result{RequeueAfter: 30 * time.Second}, err
		} else {
			registry.Status = registriesv1alpha1.RegistryStatus{LastTransition: &now}
			result = reconcile.Result{RequeueAfter: 120 * time.Second}
		}

		return r.updateRegistryCR(ctx, nil, originalRegistry, registry, result)
	}

	// Build a client to connect to the harbor API
	harborClient, err := internal.BuildClient(ctx, r.client, harbor)
	if err != nil {
		return reconcile.Result{Requeue: true}, err
	}

	switch registry.Status.Phase {
	default:
		return reconcile.Result{}, nil

	case registriesv1alpha1.RegistryStatusPhaseUnknown:
		registry.Status = registriesv1alpha1.RegistryStatus{Phase: registriesv1alpha1.RegistryStatusPhaseCreating}
		result = reconcile.Result{Requeue: true}

	case registriesv1alpha1.RegistryStatusPhaseCreating:
		helper.PushFinalizer(registry, FinalizerName)

		// Install the registry
		err = r.assertExistingRegistry(ctx, harborClient, registry)
		if err != nil {
			return reconcile.Result{}, err
		}

		registry.Status = registriesv1alpha1.RegistryStatus{Phase: registriesv1alpha1.RegistryStatusPhaseReady}
		result = reconcile.Result{Requeue: true}
	case registriesv1alpha1.RegistryStatusPhaseReady:
		err := r.assertExistingRegistry(ctx, harborClient, registry)
		if err != nil {
			return reconcile.Result{}, err
		}

		result = reconcile.Result{}

	case registriesv1alpha1.RegistryStatusPhaseTerminating:
		// Delete the registry via harbor API
		err := r.assertDeletedRegistry(ctx, reqLogger, harborClient, registry)
		if err != nil {
			return reconcile.Result{}, err
		}

		result = reconcile.Result{}
	}

	return r.updateRegistryCR(ctx, harbor, originalRegistry, registry, result)
}

// updateRegistryCR compares the new CR status and finalizers with the pre-existing ones and updates them accordingly.
func (r *ReconcileRegistry) updateRegistryCR(ctx context.Context, parentInstance *registriesv1alpha1.Instance,
	originalRegistry, registry *registriesv1alpha1.Registry, result reconcile.Result) (reconcile.Result, error) {
	if originalRegistry == nil || registry == nil {
		return reconcile.Result{}, fmt.Errorf("cannot update registry '%s' because the original registry is nil",
			registry.Spec.Name)
	}

	// Update Status
	if !reflect.DeepEqual(originalRegistry.Status, registry.Status) {
		originalRegistry.Status = registry.Status
		if err := r.client.Status().Update(ctx, originalRegistry); err != nil {
			return reconcile.Result{}, err
		}
	}

	// set owner
	if (len(originalRegistry.OwnerReferences) == 0) && parentInstance != nil {
		err := controllerruntime.SetControllerReference(parentInstance, originalRegistry, r.scheme)
		if err != nil {
			return reconcile.Result{}, err
		}
	}

	// Update Finalizer
	if !reflect.DeepEqual(originalRegistry.Finalizers, registry.Finalizers) {
		originalRegistry.SetFinalizers(registry.Finalizers)
	}

	if err := r.client.Update(ctx, originalRegistry); err != nil {
		return reconcile.Result{Requeue: true}, err
	}

	return reconcile.Result{}, nil
}

// assertExistingRegistry checks a harbor registry for existence and creates it accordingly.
func (r *ReconcileRegistry) assertExistingRegistry(ctx context.Context, harborClient *h.RESTClient,
	originalRegistry *registriesv1alpha1.Registry) error {
	_, err := harborClient.GetRegistry(ctx, originalRegistry.Name)

	if err == internal.ErrRegistryNotFound {
		rReq, err := r.buildRegistryFromSpec(originalRegistry)
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
	} else if err != nil {
		return err
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
func (r *ReconcileRegistry) ensureRegistry(ctx context.Context, harborClient *h.RESTClient,
	originalRegistry *registriesv1alpha1.Registry) error {
	// Get the registry held by harbor
	heldRegistry, err := harborClient.GetRegistry(ctx, originalRegistry.Spec.Name)
	if err != nil {
		return err
	}

	// Construct a registry from the CR spec
	newReg, err := r.buildRegistryFromSpec(originalRegistry)
	if err != nil {
		return err
	}

	// use id from harbor instance
	if originalRegistry.Spec.ID != heldRegistry.ID {
		originalRegistry.Spec.ID = heldRegistry.ID

		patch := client.MergeFrom(originalRegistry.DeepCopy())
		if err := r.client.Patch(ctx, originalRegistry, patch); err != nil {
			return err
		}
	}

	if newReg.Credential == nil {
		newReg.Credential = &modelv1.RegistryCredential{}
	}
	// Compare the registries and update accordingly
	if !reflect.DeepEqual(heldRegistry, newReg) {
		return r.updateRegistry(ctx, harborClient, newReg)
	}

	return nil
}

// updateRegistry triggers the update of a registry.
func (r *ReconcileRegistry) updateRegistry(ctx context.Context, harborClient *h.RESTClient,
	reg *modelv1.Registry) error {
	return harborClient.UpdateRegistry(ctx, reg)
}

// buildRegistryFromSpec constructs and returns a Harbor registry object from the CR object's spec.
func (r *ReconcileRegistry) buildRegistryFromSpec(originalRegistry *registriesv1alpha1.Registry) (*modelv1.Registry,
	error) {
	parsedURL, err := parseURL(originalRegistry.Spec.URL)
	if err != nil {
		return nil, err
	}

	registryType, err := enumRegistryType(originalRegistry.Spec.Type)
	if err != nil {
		return nil, err
	}

	return &modelv1.Registry{
		ID:          originalRegistry.Spec.ID,
		Name:        originalRegistry.Spec.Name,
		Description: originalRegistry.Spec.Description,
		Type:        string(registryType),
		URL:         parsedURL,
		Credential:  originalRegistry.Spec.Credential,
		Insecure:    originalRegistry.Spec.Insecure,
	}, nil
}

// assertDeletedRegistry deletes a registry, first ensuring its existence.
func (r *ReconcileRegistry) assertDeletedRegistry(ctx context.Context, log logr.Logger, harborClient *h.RESTClient,
	registry *registriesv1alpha1.Registry) error {
	reg, err := harborClient.GetRegistry(ctx, registry.Name)
	if err == nil {
		err = harborClient.DeleteRegistry(ctx, reg)
		if err != nil {
			return err
		}
	} else if err != internal.ErrRegistryNotFound {
		return err
	}

	log.Info("pulling finalizers")
	helper.PullFinalizer(registry, FinalizerName)

	return nil
}
