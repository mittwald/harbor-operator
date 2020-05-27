package replication

import (
	"context"
	"fmt"
	"reflect"
	"time"

	v1 "k8s.io/api/core/v1"

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

var log = logf.Log.WithName("controller_replication")

// Add creates a new Replication Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileReplication{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
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

// blank assignment to verify that ReconcileReplication implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileReplication{}

// ReconcileReplication reconciles a Replication object
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
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	originalReplication := replication.DeepCopy()

	if replication.ObjectMeta.DeletionTimestamp != nil && replication.Status.Phase != registriesv1alpha1.ReplicationStatusPhaseTerminating {
		replication.Status = registriesv1alpha1.ReplicationStatus{Phase: registriesv1alpha1.ReplicationStatusPhaseTerminating}
		return r.patchReplication(ctx, originalReplication, replication)
	}

	// Fetch the Instance
	harbor, err := internal.FetchReadyHarborInstance(ctx, replication.Namespace, replication.Spec.ParentInstance.Name, r.client)
	if err != nil {
		if _, ok := err.(internal.ErrInstanceNotFound); ok {
			replication.Status = registriesv1alpha1.ReplicationStatus{Name: string(registriesv1alpha1.ReplicationStatusPhaseCreating)}
			// Requeue, the instance might not have been created yet
			return reconcile.Result{RequeueAfter: 30 * time.Second}, nil
		} else if _, ok := err.(internal.ErrInstanceNotReady); ok {
			return reconcile.Result{RequeueAfter: 120 * time.Second}, err
		} else {
			replication.Status = registriesv1alpha1.ReplicationStatus{LastTransition: &now}
		}
		res, err := r.patchReplication(ctx, originalReplication, replication)
		if err != nil {
			return res, err
		}
	}

	// Build a client to connet to the harbor API
	harborClient, err := internal.BuildClient(ctx, r.client, harbor)
	if err != nil {
		return reconcile.Result{Requeue: true}, err
	}

	switch replication.Status.Phase {
	default:
		return reconcile.Result{}, nil
	case registriesv1alpha1.ReplicationStatusPhaseUnknown:
		replication.Status = registriesv1alpha1.ReplicationStatus{Phase: registriesv1alpha1.ReplicationStatusPhaseCreating}

	case registriesv1alpha1.ReplicationStatusPhaseCreating:
		helper.PushFinalizer(replication, FinalizerName)

		// Install the replication
		err = r.assertExistingReplication(ctx, harborClient, replication)
		if err != nil {
			return reconcile.Result{}, err
		}

		if replication.Spec.TriggerAfterCreation {
			replExec := h.ReplicationExecution{
				PolicyID: replication.Spec.ID,
				Trigger:  "manual",
			}
			if err = harborClient.Replications().TriggerReplicationExecution(replExec); err != nil {
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
		err := r.assertDeletedReplication(reqLogger, harborClient, replication)
		if err != nil {
			return reconcile.Result{}, err
		}
	}
	return r.patchReplication(ctx, originalReplication, replication)
}

// patchReplication compares the new CR status and finalizers with the pre-existing ones and updates them accordingly
func (r *ReconcileReplication) patchReplication(ctx context.Context, originalReplication, replication *registriesv1alpha1.Replication) (reconcile.Result, error) {
	// Update Status
	if !reflect.DeepEqual(originalReplication.Status, replication.Status) {
		originalReplication.Status = replication.Status
		if err := r.client.Status().Update(ctx, originalReplication); err != nil {
			return reconcile.Result{}, err
		}
	}

	// Update Finalizers
	if !reflect.DeepEqual(originalReplication.Finalizers, replication.Finalizers) {
		originalReplication.Finalizers = replication.Finalizers
	}

	if err := r.client.Update(ctx, originalReplication); err != nil {
		return reconcile.Result{Requeue: true}, err
	}

	return reconcile.Result{Requeue: true}, nil
}

// assertExistingReplication checks a harbor replication for existence and creates it accordingly
func (r *ReconcileReplication) assertExistingReplication(ctx context.Context, harborClient *h.Client, originalReplication *registriesv1alpha1.Replication) error {
	_, err := internal.GetReplication(harborClient, originalReplication)
	if err == internal.ErrReplicationNotFound {
		rReq, err := r.buildReplicationFromSpec(originalReplication)
		if err != nil {
			return err
		}
		err = harborClient.Replications().CreateReplicationPolicy(rReq)
		if err != nil {
			return err
		}
	} else if err != nil {
		return err
	}
	return r.ensureReplication(ctx, harborClient, originalReplication)
}

// ensureReplication gets and compares the spec of the replication held by the harbor API with the spec of the existing CR
func (r *ReconcileReplication) ensureReplication(ctx context.Context, harborClient *h.Client, originalReplication *registriesv1alpha1.Replication) error {
	// Get the registry held by harbor
	heldReplication, err := internal.GetReplication(harborClient, originalReplication)
	if err != nil {
		return err
	}

	// Construct a replication from the CR spec
	newRep, err := r.buildReplicationFromSpec(originalReplication)
	if err != nil {
		return err
	}

	// use id from harbor instance
	if originalReplication.Spec.ID != heldReplication.ID {
		patch := client.MergeFrom(originalReplication.DeepCopy())
		originalReplication.Spec.ID = heldReplication.ID
		if err := r.client.Patch(ctx, originalReplication, patch); err != nil {
			return err
		}
	}

	// Compare the replications and update accordingly
	if !reflect.DeepEqual(heldReplication, newRep) {
		return r.updateReplication(harborClient, newRep)
	}
	return nil
}

// updateReplication triggers an update of a replication (policy)
func (r *ReconcileReplication) updateReplication(harborClient *h.Client, rep h.ReplicationPolicy) error {
	return harborClient.Replications().UpdateReplicationPolicyByID(rep.ID, rep)
}

// buildReplicationFromSpec returns an API conformed ReplicationPolicy object
func (r *ReconcileReplication) buildReplicationFromSpec(originalReplication *registriesv1alpha1.Replication) (h.ReplicationPolicy, error) {
	var hf []*h.Filter
	hf = append(hf, &h.Filter{})
	if originalReplication.Spec.Filters != nil {
		for _, v := range originalReplication.Spec.Filters {
			err := internal.CheckFilterType(v.Type)
			if err != nil {
				return h.ReplicationPolicy{}, err
			}
			hf = append(hf, &h.Filter{
				Type:  v.Type,
				Value: v.Value,
			})
		}
	}

	var ht = &h.Trigger{}
	if originalReplication.Spec.Trigger != nil {
		validatedType, err := internal.CheckAndGetReplicationTriggerType(originalReplication.Spec.Trigger.Type)
		if err != nil {
			return h.ReplicationPolicy{}, err
		}
		ht.Type = validatedType
		ht.Settings = &h.TriggerSettings{Cron: originalReplication.Spec.Trigger.Settings.Cron}
	}

	newRep := h.ReplicationPolicy{
		ID:            originalReplication.Spec.ID,
		Name:          originalReplication.Spec.Name,
		Description:   originalReplication.Spec.Description,
		Creator:       originalReplication.Spec.Creator,
		DestNamespace: originalReplication.Spec.DestNamespace,
		Override:      originalReplication.Spec.Override,
		Enabled:       originalReplication.Spec.Enabled,
		Trigger:       ht,
		Filters:       hf,
		Deletion:      originalReplication.Spec.Deletion,
	}

	if originalReplication.Spec.SrcRegistry != nil && originalReplication.Spec.DestRegistry != nil {
		return h.ReplicationPolicy{}, fmt.Errorf("both dest_registry and src_registry are set! Please specify only one of them")
	}
	if originalReplication.Spec.SrcRegistry != nil {
		hReg, err := r.getHarborRegistryFromRef(context.Background(), originalReplication.Spec.SrcRegistry, originalReplication.Namespace)
		if err != nil {
			return h.ReplicationPolicy{}, err
		}
		newRep.SrcRegistry = hReg
	} else if originalReplication.Spec.DestRegistry != nil {
		hReg, err := r.getHarborRegistryFromRef(context.Background(), originalReplication.Spec.DestRegistry, originalReplication.Namespace)
		if err != nil {
			return h.ReplicationPolicy{}, err
		}
		newRep.DestRegistry = hReg
	}

	return newRep, nil
}

// getHarborRegistryFromRef retrieves the registryRef and returns a pointer to a goharbor-client Registry Object
func (r *ReconcileReplication) getHarborRegistryFromRef(ctx context.Context, registryRef *v1.LocalObjectReference, namespace string) (*h.Registry, error) {
	var registry registriesv1alpha1.Registry
	err := r.client.Get(ctx, client.ObjectKey{Namespace: namespace, Name: registryRef.Name}, &registry)
	if err != nil {
		return nil, err
	}

	return registry.Spec.ToHarborRegistry(), nil
}

// assertDeletedReplication deletes a replication, first ensuring its existence
func (r *ReconcileReplication) assertDeletedReplication(log logr.Logger, harborClient *h.Client, replication *registriesv1alpha1.Replication) error {
	rep, err := internal.GetReplication(harborClient, replication)
	if err != nil {
		return err
	}

	err = harborClient.Replications().DeleteReplicationPolicyByID(rep.ID)
	if err != nil {
		return err
	}

	log.Info("pulling finalizers")
	helper.PullFinalizer(replication, FinalizerName)

	return nil
}
