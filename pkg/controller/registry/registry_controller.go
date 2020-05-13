package registry

import (
	"context"
	"net/url"
	"reflect"
	"time"

	"github.com/go-logr/logr"
	h "github.com/mittwald/goharbor-client"
	"github.com/mittwald/harbor-operator/pkg/controller/internal"
	"github.com/mittwald/harbor-operator/pkg/internal/helper"

	registriesv1alpha1 "github.com/mittwald/harbor-operator/pkg/apis/registries/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
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

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileRegistry{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
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

// blank assignment to verify that ReconcileRegistry implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileRegistry{}

// ReconcileRegistry reconciles a Registry object
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

	// Fetch the Registry instance
	registry := &registriesv1alpha1.Registry{}
	err := r.client.Get(ctx, request.NamespacedName, registry)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	originalRegistry := registry.DeepCopy()

	// Fetch the Instance
	harbor, err := internal.FetchReadyHarborInstance(ctx, registry.Namespace, registry.Spec.ParentInstance.Name, r.client)
	if err != nil {
		if _, ok := err.(internal.ErrInstanceNotFound); ok {
			err := r.SetRegistryStatus(
				registry,
				registriesv1alpha1.RegistryStatus{
					Name: string(registriesv1alpha1.RegistryStatusPhaseCreating)})
			if err != nil {
				return reconcile.Result{}, err
			}
			return reconcile.Result{RequeueAfter: 30 * time.Second}, nil
		} else if _, ok := err.(internal.ErrInstanceNotReady); ok {
			return reconcile.Result{RequeueAfter: 120 * time.Second}, err
		} else {
			registry.Status = registriesv1alpha1.RegistryStatus{LastTransition: &now}
		}
		res, err := r.patchRegistry(ctx, originalRegistry, registry)
		if err != nil {
			return res, err
		}
	}

	// Build a client to connet to the harbor API
	harborClient, err := internal.BuildClient(ctx, r.client, harbor)
	if err != nil {
		return reconcile.Result{Requeue: true}, err
	}

	// Add finalizers to the CR object
	if registry.DeletionTimestamp == nil {
		var hasFinalizer bool
		for i := range registry.Finalizers {
			if registry.Finalizers[i] == FinalizerName {
				hasFinalizer = true
			}
		}
		if !hasFinalizer {
			helper.PushFinalizer(registry, FinalizerName)
			return r.patchRegistry(ctx, originalRegistry, registry)
		}
	}

	switch registry.Status.Phase {
	default:
		return reconcile.Result{}, nil
	case registriesv1alpha1.RegistryStatusPhaseUnknown:
		err = r.SetRegistryStatus(registry,
			registriesv1alpha1.RegistryStatus{
				Phase: registriesv1alpha1.RegistryStatusPhaseCreating})
		if err != nil {
			return reconcile.Result{}, err
		}
	case registriesv1alpha1.RegistryStatusPhaseCreating:
		// Install the registry
		err = r.assertExistingRegistry(harborClient, registry)
		if err != nil {
			return reconcile.Result{}, err
		}
		err = r.SetRegistryStatus(registry,
			registriesv1alpha1.RegistryStatus{
				Phase: registriesv1alpha1.RegistryStatusPhaseReady})
		if err != nil {
			return reconcile.Result{}, err
		}
	case registriesv1alpha1.RegistryStatusPhaseReady:
		// Compare the state of spec to the state of what the API returns
		// If the Registry object is deleted, assume that the repository needs deletion, too
		if registry.ObjectMeta.DeletionTimestamp != nil {
			err = r.SetRegistryStatus(registry,
				registriesv1alpha1.RegistryStatus{
					Phase: registriesv1alpha1.RegistryStatusPhaseTerminating})
			if err != nil {
				return reconcile.Result{}, err
			}
			return r.patchRegistry(ctx, originalRegistry, registry)
		}

		err := r.assertExistingRegistry(harborClient, registry)
		if err != nil {
			return reconcile.Result{}, err
		}

	case registriesv1alpha1.RegistryStatusPhaseTerminating:
		// Delete the registry via harbor API
		err := r.assertDeletedRegistry(reqLogger, harborClient, registry)
		if err != nil {
			return reconcile.Result{}, err
		}

	}
	return r.patchRegistry(ctx, originalRegistry, registry)
}

func (r *ReconcileRegistry) SetRegistryStatus(registry *registriesv1alpha1.Registry, status registriesv1alpha1.RegistryStatus) error {
	now := metav1.Now()
	registry.Status = registriesv1alpha1.RegistryStatus{
		Name:           status.Name,
		Phase:          status.Phase,
		Message:        status.Message,
		LastTransition: &now,
	}
	return nil
}

// patchRegistry compares the new CR status and finalizers with the pre-existing ones and updates them accordingly
func (r *ReconcileRegistry) patchRegistry(ctx context.Context, originalRegistry, registry *registriesv1alpha1.Registry) (reconcile.Result, error) {
	// Update Status
	if !reflect.DeepEqual(originalRegistry.Status, registry.Status) {
		originalRegistry.Status = registry.Status
		if err := r.client.Status().Update(ctx, originalRegistry); err != nil {
			return reconcile.Result{}, err
		}
	}

	// Update Finalizers
	if !reflect.DeepEqual(originalRegistry.Finalizers, registry.Finalizers) {
		originalRegistry.Finalizers = registry.Finalizers
	}

	if err := r.client.Update(ctx, originalRegistry); err != nil {
		return reconcile.Result{Requeue: true}, err
	}

	return reconcile.Result{Requeue: true}, nil
}

// assertExistingRegistry checks a harbor registry for existence and creates it accordingly
func (r *ReconcileRegistry) assertExistingRegistry(harborClient *h.Client, originalRegistry *registriesv1alpha1.Registry) error {
	_, err := internal.GetRegistry(harborClient, originalRegistry)
	if err == internal.ErrRegistryNotFound {
		rReq, err := r.buildRegistryFromSpec(originalRegistry)
		if err != nil {
			return err
		}
		err = harborClient.Registries().CreateRegistry(rReq)
		if err != nil {
			return err
		}
	} else if err != nil {
		return err
	}
	return r.ensureRegistry(harborClient, originalRegistry)
}

func parseURL(raw string) (string, error) {
	var parsed *url.URL
	parsed, err := url.ParseRequestURI(raw)
	if err != nil {
		return "", err
	}
	return parsed.String(), nil
}

// ensureRegistry gets and compares the spec of the registry held by the harbor API with the spec of the existing CR
func (r *ReconcileRegistry) ensureRegistry(harborClient *h.Client, originalRegistry *registriesv1alpha1.Registry) error {
	// Get the registry held by harbor
	heldRegistry, err := internal.GetRegistry(harborClient, originalRegistry)
	if err != nil {
		return err
	}

	// Construct a registry from the CR spec
	newReg, err := r.buildRegistryFromSpec(originalRegistry)
	if err != nil {
		return err
	}

	if newReg.Credential == nil {
		newReg.Credential = &h.Credential{}
	}
	// Compare the registries and update accordingly
	if !reflect.DeepEqual(heldRegistry, newReg) {
		return r.updateRegistry(harborClient, newReg)
	}
	return nil

}

// updateRegistry triggers the update of a registry
func (r *ReconcileRegistry) updateRegistry(harborClient *h.Client, reg h.Registry) error {
	return harborClient.Registries().UpdateRegistryByID(reg)
}

// buildRegistryFromSpec constructs and returns a Harbor registry object from the CR object's spec
func (r *ReconcileRegistry) buildRegistryFromSpec(originalRegistry *registriesv1alpha1.Registry) (h.Registry, error) {
	parsedURL, err := parseURL(originalRegistry.Spec.URL)

	if err != nil {
		return h.Registry{}, err
	}

	tokenServiceURL := originalRegistry.Spec.TokenServiceURL
	if tokenServiceURL != "" {
		parsedTokenServiceURL, err := parseURL(originalRegistry.Spec.TokenServiceURL)
		if err != nil {
			return h.Registry{}, err
		}
		tokenServiceURL = parsedTokenServiceURL
	}

	return h.Registry{
		ID:              originalRegistry.Spec.ID,
		Name:            originalRegistry.Spec.Name,
		Description:     originalRegistry.Spec.Description,
		Type:            originalRegistry.Spec.Type,
		URL:             parsedURL,
		TokenServiceURL: tokenServiceURL,
		Credential:      originalRegistry.Spec.Credential,
		Insecure:        originalRegistry.Spec.Insecure,
	}, nil

}

// assertDeletedRegistry deletes a registry, first ensuring its existence
func (r *ReconcileRegistry) assertDeletedRegistry(log logr.Logger, harborClient *h.Client, registry *registriesv1alpha1.Registry) error {
	reg, err := internal.GetRegistry(harborClient, registry)
	if err != nil {
		return err
	}

	err = harborClient.Registries().DeleteRegistryByID(reg.ID)
	if err != nil {
		return err
	}

	log.Info("pulling finalizers")
	helper.PullFinalizer(registry, FinalizerName)

	return nil
}
