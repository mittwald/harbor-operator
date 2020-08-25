package replication

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"time"

	replicationClient "github.com/mittwald/goharbor-client/replication"

	modelv1 "github.com/mittwald/goharbor-client/model/v1_10_0"

	controllerruntime "sigs.k8s.io/controller-runtime"

	v1 "k8s.io/api/core/v1"

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

var log = logf.Log.WithName("controller_replication")

// Add creates a new Replication Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler.
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileReplication{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler.
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("replication-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource Replication
	err = c.Watch(&source.Kind{Type: &registriesv1alpha1.Replication{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	return nil
}

// blank assignment to verify that ReconcileReplication implements reconcile.Reconciler.
var _ reconcile.Reconciler = &ReconcileReplication{}

// ReconcileReplication reconciles a Replication object.
type ReconcileReplication struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a Replication object and makes changes based on the state read
// and what is in the Replication.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileReplication) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling Replication")

	now := metav1.Now()
	ctx := context.Background()

	// Fetch the Replication instance
	replication := &registriesv1alpha1.Replication{}

	err := r.client.Get(context.TODO(), request.NamespacedName, replication)
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

	originalReplication := replication.DeepCopy()

	if replication.ObjectMeta.DeletionTimestamp != nil &&
		replication.Status.Phase != registriesv1alpha1.ReplicationStatusPhaseTerminating {
		replication.Status = registriesv1alpha1.ReplicationStatus{
			Phase: registriesv1alpha1.ReplicationStatusPhaseTerminating,
		}

		return r.updateReplicationCR(ctx, nil, originalReplication, replication)
	}

	// Fetch the Instance
	harbor, err := internal.FetchReadyHarborInstance(ctx,
		replication.Namespace,
		replication.Spec.ParentInstance.Name,
		r.client)
	if err != nil {
		if _, ok := err.(internal.ErrInstanceNotFound); ok {
			helper.PullFinalizer(replication, FinalizerName)
		} else if _, ok := err.(internal.ErrInstanceNotReady); ok {
			return reconcile.Result{RequeueAfter: 30 * time.Second}, err
		} else {
			replication.Status = registriesv1alpha1.ReplicationStatus{LastTransition: &now}
		}

		return r.updateReplicationCR(ctx, nil, originalReplication, replication)
	}

	// Build a client to connect to the harbor API
	harborClient, err := internal.BuildClient(ctx, r.client, harbor)
	if err != nil {
		return reconcile.Result{Requeue: true}, err
	}

	switch replication.Status.Phase {
	default:
		return reconcile.Result{}, nil
	case registriesv1alpha1.ReplicationStatusPhaseUnknown:
		replication.Status = registriesv1alpha1.ReplicationStatus{
			Phase: registriesv1alpha1.ReplicationStatusPhaseCreating,
		}

	case registriesv1alpha1.ReplicationStatusPhaseCreating:
		helper.PushFinalizer(replication, FinalizerName)

		// Install the replication
		err = r.assertExistingReplication(ctx, harborClient, replication)
		if err != nil {
			return reconcile.Result{}, err
		}

		if replication.Spec.TriggerAfterCreation {
			replExec := &modelv1.ReplicationExecution{
				PolicyID: replication.Status.ID,
				Trigger:  registriesv1alpha1.ReplicationTriggerTypeManual,
			}

			if err = harborClient.TriggerReplicationExecution(ctx, replExec); err != nil {
				return reconcile.Result{}, err
			}
		}

		replication.Status = registriesv1alpha1.ReplicationStatus{Phase: registriesv1alpha1.ReplicationStatusPhaseReady}

	case registriesv1alpha1.ReplicationStatusPhaseReady:
		err := r.assertExistingReplication(ctx, harborClient, replication)
		if err != nil {
			return reconcile.Result{}, err
		}

	case registriesv1alpha1.ReplicationStatusPhaseTerminating:
		// Delete the replication via harbor API
		err := r.assertDeletedReplication(ctx, reqLogger, harborClient, replication)
		if err != nil {
			return reconcile.Result{}, err
		}
	}

	return r.updateReplicationCR(ctx, harbor, originalReplication, replication)
}

// updateReplicationCR compares the new CR status and finalizers with the pre-existing ones and updates them accordingly
func (r *ReconcileReplication) updateReplicationCR(ctx context.Context, parentInstance *registriesv1alpha1.Instance,
	originalReplication, replication *registriesv1alpha1.Replication) (reconcile.Result, error) {
	if originalReplication == nil {
		return reconcile.Result{},
			fmt.Errorf("cannot update replication '%s' because the original replication is nil",
				replication.Spec.Name)
	}

	// Update status
	if !reflect.DeepEqual(originalReplication.Status, replication.Status) {
		originalReplication.Status = replication.Status
		if err := r.client.Status().Update(ctx, originalReplication); err != nil {
			return reconcile.Result{}, err
		}
	}

	// Set owner
	if (len(originalReplication.OwnerReferences) == 0) && parentInstance != nil {
		err := controllerruntime.SetControllerReference(parentInstance, originalReplication, r.scheme)
		if err != nil {
			return reconcile.Result{}, err
		}
	}

	// Update finalizer
	if !reflect.DeepEqual(originalReplication.Finalizers, replication.Finalizers) {
		originalReplication.SetFinalizers(replication.Finalizers)
	}

	if err := r.client.Update(ctx, originalReplication); err != nil {
		return reconcile.Result{Requeue: true}, err
	}

	return reconcile.Result{}, nil
}

// assertExistingReplication checks a harbor replication for existence and creates it accordingly.
func (r *ReconcileReplication) assertExistingReplication(ctx context.Context, harborClient *h.RESTClient,
	originalReplication *registriesv1alpha1.Replication) error {
	_, err := harborClient.GetReplicationPolicy(ctx, originalReplication.Name)
	if err != nil {
		switch err.Error() {
		case replicationClient.ErrReplicationNotFoundMsg:
			rReq, err := r.buildReplicationFromCR(originalReplication)
			if err != nil {
				return err
			}

			_, err = harborClient.NewReplicationPolicy(ctx,
				rReq.DestRegistry,
				rReq.SrcRegistry,
				rReq.Deletion,
				rReq.Override,
				rReq.Enabled,
				rReq.Filters,
				rReq.Trigger,
				rReq.DestNamespace,
				rReq.Description,
				rReq.Name)
			if err != nil {
				return err
			}
		default:
			return err
		}
	}

	return r.ensureReplication(ctx, harborClient, originalReplication)
}

func enumReplicationTrigger(receivedTrigger string) (string, error) {
	if receivedTrigger == "" {
		return "", errors.New("empty replication trigger provided")
	}

	switch receivedTrigger {
	case registriesv1alpha1.ReplicationTriggerTypeEventBased,
		registriesv1alpha1.ReplicationTriggerTypeManual,
		registriesv1alpha1.ReplicationTriggerTypeScheduled:
		return receivedTrigger, nil
	default:
		return "", fmt.Errorf("invalid replication trigger type provided: '%s'", receivedTrigger)
	}
}

func enumReplicationFilterType(filterType string) (string, error) {
	if filterType == "" {
		return "", errors.New("empty replication filter type provided")
	}

	switch filterType {
	case registriesv1alpha1.ReplicationFilterTypeLabel, registriesv1alpha1.ReplicationFilterTypeName,
		registriesv1alpha1.ReplicationFilterTypeResource, registriesv1alpha1.ReplicationFilterTypeTag:
		return filterType, nil
	default:
		return "", fmt.Errorf("invalid replication filter type provided: '%s'", filterType)
	}
}

// ensureReplication gets and compares the spec of the replication held
// by the harbor API with the spec of the existing CR.
func (r *ReconcileReplication) ensureReplication(ctx context.Context, harborClient *h.RESTClient,
	originalReplication *registriesv1alpha1.Replication) error {
	// Get the replication held by harbor
	heldReplication, err := harborClient.GetReplicationPolicy(ctx, originalReplication.Name)
	if err != nil {
		return err
	}

	// Construct a replication from the CR spec
	newRep, err := r.buildReplicationFromCR(originalReplication)
	if err != nil {
		return err
	}

	if originalReplication.Status.ID != heldReplication.ID {
		originalReplication.Status.ID = heldReplication.ID

		if err := r.client.Status().Update(ctx, originalReplication); err != nil {
			return err
		}
	}

	// Fill status fields
	if originalReplication.Spec.DestRegistry != nil {
		if originalReplication.Status.Destination != heldReplication.DestRegistry.Name {
			originalReplication.Status.Destination = heldReplication.DestRegistry.Name
			originalReplication.Status.Source = "harbor"
			if err := r.client.Status().Update(ctx, originalReplication); err != nil {
				return err
			}
		}
	} else if originalReplication.Spec.SrcRegistry != nil {
		if originalReplication.Status.Source != heldReplication.SrcRegistry.Name {
			originalReplication.Status.Source = heldReplication.SrcRegistry.Name
			originalReplication.Status.Destination = "harbor"
			if err := r.client.Status().Update(ctx, originalReplication); err != nil {
				return err
			}
		}
	}

	// Compare the replications and update accordingly
	if !reflect.DeepEqual(heldReplication, newRep) {
		return harborClient.UpdateReplicationPolicy(ctx, newRep)
	}

	return nil
}

// buildReplicationFromCR returns an API conformed ReplicationPolicy object
func (r *ReconcileReplication) buildReplicationFromCR(originalReplication *registriesv1alpha1.Replication) (
	*modelv1.ReplicationPolicy, error) {
	newRep := &modelv1.ReplicationPolicy{
		ID:            originalReplication.Status.ID,
		Name:          originalReplication.Spec.Name,
		Description:   originalReplication.Spec.Description,
		DestNamespace: originalReplication.Spec.DestNamespace,
		Override:      originalReplication.Spec.Override,
		Enabled:       originalReplication.Spec.Enabled,
		Deletion:      originalReplication.Spec.ReplicateDeletion,
	}

	filters, err := addReplicationFilters(originalReplication.Spec.Filters)
	if err != nil {
		return nil, err
	}

	newRep.Filters = filters

	if originalReplication.Spec.Trigger != nil {
		triggerType, err := enumReplicationTrigger(originalReplication.Spec.Trigger.Type)
		if err != nil {
			return nil, err
		}

		newRep.Trigger = &modelv1.ReplicationTrigger{
			TriggerSettings: &modelv1.TriggerSettings{
				Cron: originalReplication.Spec.Trigger.Settings.Cron,
			},
			Type: triggerType,
		}
	}

	if originalReplication.Spec.SrcRegistry != nil && originalReplication.Spec.DestRegistry != nil {
		return &modelv1.ReplicationPolicy{},
			fmt.Errorf("both dest_registry and src_registry are set! Please specify only one of them")
	}

	if originalReplication.Spec.SrcRegistry != nil {
		hReg, err := r.getHarborRegistryFromRef(context.Background(),
			originalReplication.Spec.SrcRegistry,
			originalReplication.Namespace)
		if err != nil {
			return &modelv1.ReplicationPolicy{}, err
		}

		newRep.SrcRegistry = hReg
	} else if originalReplication.Spec.DestRegistry != nil {
		hReg, err := r.getHarborRegistryFromRef(context.Background(),
			originalReplication.Spec.DestRegistry,
			originalReplication.Namespace)
		if err != nil {
			return &modelv1.ReplicationPolicy{}, err
		}

		newRep.DestRegistry = hReg
	}

	return newRep, nil
}

func addReplicationFilters(originalFilters []registriesv1alpha1.ReplicationFilter) (
	newFilters []*modelv1.ReplicationFilter, err error) {
	for _, f := range originalFilters {
		filterType, err := enumReplicationFilterType(f.Type)
		if err != nil {
			return []*modelv1.ReplicationFilter{}, err
		}

		newFilters = append(newFilters, &modelv1.ReplicationFilter{
			Type:  filterType,
			Value: f.Value,
		})
	}

	return newFilters, nil
}

// getHarborRegistryFromRef retrieves the registryRef and returns a pointer to a goharbor-client Registry Object
func (r *ReconcileReplication) getHarborRegistryFromRef(ctx context.Context, registryRef *v1.LocalObjectReference,
	namespace string) (*modelv1.Registry, error) {
	var registry registriesv1alpha1.Registry

	err := r.client.Get(ctx, client.ObjectKey{Namespace: namespace, Name: registryRef.Name}, &registry)
	if err != nil {
		return nil, err
	}

	if registry.Status.Phase != registriesv1alpha1.RegistryStatusPhaseReady {
		return nil, internal.ErrRegistryNotReady(registry.Name)
	}

	var registryID int64

	if registry.Status.ID != 0 {
		registryID = registry.Status.ID
	}

	return registry.Spec.ToHarborRegistry(registryID), nil
}

// assertDeletedReplication deletes a replication, first ensuring its existence
func (r *ReconcileReplication) assertDeletedReplication(ctx context.Context, log logr.Logger,
	harborClient *h.RESTClient, replication *registriesv1alpha1.Replication) error {
	receivedReplicationPolicy, err := harborClient.GetReplicationPolicy(ctx, replication.Name)
	if err != nil {
		return err
	} else {
		err = harborClient.DeleteReplicationPolicy(ctx, receivedReplicationPolicy)
		if err != nil {
			return err
		}
	}

	log.Info("pulling finalizers")
	helper.PullFinalizer(replication, FinalizerName)

	return nil
}
