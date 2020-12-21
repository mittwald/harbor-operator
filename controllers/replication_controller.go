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
	"errors"
	"fmt"
	"reflect"
	"time"

	legacymodel "github.com/mittwald/goharbor-client/v3/apiv2/model/legacy"
	replicationapi "github.com/mittwald/goharbor-client/v3/apiv2/replication"
	v1 "k8s.io/api/core/v1"

	h "github.com/mittwald/goharbor-client/v3/apiv2"
	"github.com/mittwald/harbor-operator/controllers/helper"
	"github.com/mittwald/harbor-operator/controllers/internal"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	registriesv1alpha2 "github.com/mittwald/harbor-operator/api/v1alpha2"
)

// ReplicationReconciler reconciles a Replication object
type ReplicationReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// blank assignment to verify that ReplicationReconciler implements reconcile.Reconciler.
var _ reconcile.Reconciler = &ReplicationReconciler{}

// +kubebuilder:rbac:groups=registries.mittwald.de,resources=replications,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=registries.mittwald.de,resources=replications/status,verbs=get;update;patch
func (r *ReplicationReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	reqLogger := r.Log.WithValues("replication", req.NamespacedName)
	reqLogger.Info("Reconciling Replication")

	now := metav1.Now()
	ctx := context.Background()

	// Fetch the Replication instance
	replication := &registriesv1alpha2.Replication{}

	err := r.Client.Get(ctx, req.NamespacedName, replication)
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

	originalReplication := replication.DeepCopy()

	if replication.ObjectMeta.DeletionTimestamp != nil && replication.Status.Phase != registriesv1alpha2.ReplicationStatusPhaseTerminating {
		replication.Status.Phase = registriesv1alpha2.ReplicationStatusPhaseTerminating
		return r.updateReplicationCR(ctx, nil, originalReplication, replication)
	}

	// Fetch the Instance
	harbor, err := internal.FetchReadyHarborInstance(ctx,
		replication.Namespace,
		replication.Spec.ParentInstance.Name,
		r.Client)
	if err != nil {
		if _, ok := err.(internal.ErrInstanceNotFound); ok {
			helper.PullFinalizer(replication, internal.FinalizerName)
		} else if _, ok := err.(internal.ErrInstanceNotReady); ok {
			return ctrl.Result{RequeueAfter: 30 * time.Second}, err
		} else {
			replication.Status = registriesv1alpha2.ReplicationStatus{LastTransition: &now}
		}

		return r.updateReplicationCR(ctx, nil, originalReplication, replication)
	}

	// Build a client to connect to the harbor API
	harborClient, err := internal.BuildClient(ctx, r.Client, harbor)
	if err != nil {
		return ctrl.Result{Requeue: true}, err
	}

	switch replication.Status.Phase {
	default:
		return ctrl.Result{}, nil
	case registriesv1alpha2.ReplicationStatusPhaseUnknown:

		replication.Status.Phase = registriesv1alpha2.ReplicationStatusPhaseCreating

	case registriesv1alpha2.ReplicationStatusPhaseCreating:
		if err := r.assertExistingReplication(ctx, harborClient, replication); err != nil {
			return ctrl.Result{}, err
		}

		helper.PushFinalizer(replication, internal.FinalizerName)
		if replication.Spec.TriggerAfterCreation {
			replExec := &legacymodel.ReplicationExecution{
				PolicyID: replication.Status.ID,
				Trigger:  registriesv1alpha2.ReplicationTriggerTypeManual,
			}

			if err := harborClient.TriggerReplicationExecution(ctx, replExec); err != nil {
				reqLogger.Info(fmt.Sprintf("replication execution after creation could not be triggered: %s", err))
				return ctrl.Result{}, err
			}

			replication.Status.Phase = registriesv1alpha2.ReplicationStatusPhaseManualExecutionRunning

			return r.updateReplicationCR(ctx, harbor, originalReplication, replication)
		}

		replication.Status.Phase = registriesv1alpha2.ReplicationStatusPhaseCompleted

	case registriesv1alpha2.ReplicationStatusPhaseManualExecutionRunning:
		running, err := r.reconcileRunningReplicationExecution(ctx, replication, harborClient)
		if err != nil {
			reqLogger.Info(fmt.Sprintf("replication execution failed: %s", err))
			replication.Status.Phase = registriesv1alpha2.ReplicationStatusPhaseManualExecutionFailed
		}
		if running {
			return ctrl.Result{RequeueAfter: 1 * time.Minute}, nil
		}
		if !running {
			replication.Status.Phase = registriesv1alpha2.ReplicationStatusPhaseManualExecutionFinished
		}

	case registriesv1alpha2.ReplicationStatusPhaseManualExecutionFinished:
		err := r.reconcileFinishedReplicationExecution(ctx, replication, harborClient)
		if err != nil {
			reqLogger.Info(fmt.Sprintf("replication execution failed: %s", err))
			replication.Status.Phase = registriesv1alpha2.ReplicationStatusPhaseManualExecutionFailed
		}
		replication.Status.Phase = registriesv1alpha2.ReplicationStatusPhaseCompleted

	case registriesv1alpha2.ReplicationStatusPhaseManualExecutionFailed:
		replication.Status.Phase = registriesv1alpha2.ReplicationStatusPhaseCreating

	case registriesv1alpha2.ReplicationStatusPhaseCompleted:
		err := r.assertExistingReplication(ctx, harborClient, replication)
		if err != nil {
			return ctrl.Result{}, err
		}

	case registriesv1alpha2.ReplicationStatusPhaseTerminating:
		// Delete the replication via harbor API
		err := r.assertDeletedReplication(ctx, reqLogger, harborClient, replication)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	return r.updateReplicationCR(ctx, harbor, originalReplication, replication)
}

// reconcileRunningReplicationExecution fetches the newest replication execution of
// the replication and checks, whether it is still running or not.
// Returns an error, if no replication execution could be found for the replication policy.
func (r *ReplicationReconciler) reconcileRunningReplicationExecution(ctx context.Context, replication *registriesv1alpha2.Replication, harborClient *h.RESTClient) (bool, error) {
	replExec := &legacymodel.ReplicationExecution{
		PolicyID: replication.Status.ID,
		Trigger:  registriesv1alpha2.ReplicationTriggerTypeManual,
	}

	executions, err := harborClient.GetReplicationExecutions(ctx, replExec)
	if err != nil {
		return false, fmt.Errorf("no replication executions found: %s", err)
	}

	newestReplExecID := getNewestReplicationExecutionID(executions)

	execution, err := harborClient.GetReplicationExecutionByID(ctx, newestReplExecID)
	if err != nil {
		return false, fmt.Errorf("could not get the latest running replication execution, id: %d, error: %s", newestReplExecID, err)
	}

	if execution.InProgress > 0 {
		return true, nil
	}

	return false, nil
}

// reconcileFinishedReplicationExecution fetches the latest finished replication execution and returns an error when it has failed.
func (r *ReplicationReconciler) reconcileFinishedReplicationExecution(ctx context.Context, replication *registriesv1alpha2.Replication, harborClient *h.RESTClient) error {
	replExec := &legacymodel.ReplicationExecution{
		PolicyID: replication.Status.ID,
		Trigger:  registriesv1alpha2.ReplicationTriggerTypeManual,
	}

	executions, err := harborClient.GetReplicationExecutions(ctx, replExec)
	if err != nil {
		return fmt.Errorf("fetching successful replication executions failed for replication policy %s: %s", replication.Status.Name, err)
	}

	newestReplExecID := getNewestReplicationExecutionID(executions)

	execution, err := harborClient.GetReplicationExecutionByID(ctx, newestReplExecID)
	if err != nil {
		return fmt.Errorf("could not get the latest finished replication execution, id: %d, error: %s", newestReplExecID, err)
	}

	if execution.Failed > 0 || execution.Status == "Failed" {
		return fmt.Errorf("latest execution was not successful, replication policy: %s", replication.Spec.Name)
	}

	return nil
}

func (r *ReplicationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&registriesv1alpha2.Replication{}).
		Complete(r)
}

// getNewestReplicationExecutionID takes a slice of replication executions and returns the one with the highest ID.
func getNewestReplicationExecutionID(executions []*legacymodel.ReplicationExecution) int64 {
	max := executions[0].ID
	for i := range executions {
		if max < executions[i].ID {
			max = executions[i].ID
		}
	}
	return max
}

// updateReplicationCR compares the new CR status and finalizers with the pre-existing ones and updates them accordingly
func (r *ReplicationReconciler) updateReplicationCR(ctx context.Context, parentInstance *registriesv1alpha2.Instance,
	originalReplication, replication *registriesv1alpha2.Replication) (ctrl.Result, error) {
	if originalReplication == nil {
		return ctrl.Result{},
			fmt.Errorf("cannot update replication '%s' because the original replication is nil",
				replication.Spec.Name)
	}

	// Set owner
	if (len(originalReplication.OwnerReferences) == 0) && parentInstance != nil {
		err := ctrl.SetControllerReference(parentInstance, originalReplication, r.Scheme)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	// Update finalizer
	if !reflect.DeepEqual(originalReplication.Finalizers, replication.Finalizers) {
		originalReplication.SetFinalizers(replication.Finalizers)
	}

	// Update status
	if !reflect.DeepEqual(originalReplication.Status.Phase, replication.Status.Phase) {
		originalReplication.Status.Phase = replication.Status.Phase
		if err := r.Client.Status().Update(ctx, originalReplication); err != nil {
			return ctrl.Result{}, err
		}
	}

	if err := r.Client.Update(ctx, originalReplication); err != nil {
		return ctrl.Result{Requeue: true}, err
	}

	return ctrl.Result{}, nil
}

// assertExistingReplication checks a harbor replication for existence and creates it accordingly.
func (r *ReplicationReconciler) assertExistingReplication(ctx context.Context, harborClient *h.RESTClient,
	originalReplication *registriesv1alpha2.Replication) error {
	_, err := harborClient.GetReplicationPolicy(ctx, originalReplication.Spec.Name)
	if err != nil {
		switch err.Error() {
		case replicationapi.ErrReplicationNotFoundMsg:
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
	case registriesv1alpha2.ReplicationTriggerTypeEventBased,
		registriesv1alpha2.ReplicationTriggerTypeManual,
		registriesv1alpha2.ReplicationTriggerTypeScheduled:
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
	case registriesv1alpha2.ReplicationFilterTypeLabel, registriesv1alpha2.ReplicationFilterTypeName,
		registriesv1alpha2.ReplicationFilterTypeResource, registriesv1alpha2.ReplicationFilterTypeTag:
		return filterType, nil
	default:
		return "", fmt.Errorf("invalid replication filter type provided: '%s'", filterType)
	}
}

// ensureReplication gets and compares the spec of the replication held
// by the harbor API with the spec of the existing CR.
func (r *ReplicationReconciler) ensureReplication(ctx context.Context, harborClient *h.RESTClient,
	originalReplication *registriesv1alpha2.Replication) error {
	// Get the replication held by harbor
	heldReplication, err := harborClient.GetReplicationPolicy(ctx, originalReplication.Spec.Name)
	if err != nil {
		return err
	}

	if originalReplication.Status.ID != heldReplication.ID {
		originalReplication.Status.ID = heldReplication.ID

		if err := r.Client.Status().Update(ctx, originalReplication); err != nil {
			return err
		}
	}

	// Fill status fields
	if originalReplication.Spec.DestRegistry != nil {
		if originalReplication.Status.Destination != heldReplication.DestRegistry.Name {
			originalReplication.Status.Destination = heldReplication.DestRegistry.Name
			originalReplication.Status.Source = "harbor"
			if err := r.Client.Status().Update(ctx, originalReplication); err != nil {
				return err
			}
		}
	} else if originalReplication.Spec.SrcRegistry != nil {
		if originalReplication.Status.Source != heldReplication.SrcRegistry.Name {
			originalReplication.Status.Source = heldReplication.SrcRegistry.Name
			originalReplication.Status.Destination = "harbor"
			if err := r.Client.Status().Update(ctx, originalReplication); err != nil {
				return err
			}
		}
	}

	// Construct a replication from the CR spec
	newRep, err := r.buildReplicationFromCR(originalReplication)
	if err != nil {
		return err
	}

	// Compare the replications and update accordingly
	if !reflect.DeepEqual(heldReplication, newRep) {
		return harborClient.UpdateReplicationPolicy(ctx, newRep)
	}

	return nil
}

// buildReplicationFromCR returns an API conformed ReplicationPolicy object
func (r *ReplicationReconciler) buildReplicationFromCR(originalReplication *registriesv1alpha2.Replication) (
	*legacymodel.ReplicationPolicy, error) {
	newRep := &legacymodel.ReplicationPolicy{
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

		newRep.Trigger = &legacymodel.ReplicationTrigger{
			Type: triggerType,
		}

		if originalReplication.Spec.Trigger.Settings != nil {
			newRep.Trigger.TriggerSettings = &legacymodel.TriggerSettings{
				Cron: originalReplication.Spec.Trigger.Settings.Cron,
			}
		}
	}

	if originalReplication.Spec.SrcRegistry != nil && originalReplication.Spec.DestRegistry != nil {
		return &legacymodel.ReplicationPolicy{},
			fmt.Errorf("both dest_registry and src_registry are set! Please specify only one of them")
	}

	if originalReplication.Spec.SrcRegistry != nil {
		hReg, err := r.getHarborRegistryFromRef(context.Background(),
			originalReplication.Spec.SrcRegistry,
			originalReplication.Namespace)
		if err != nil {
			return &legacymodel.ReplicationPolicy{}, err
		}

		newRep.SrcRegistry = hReg
	} else if originalReplication.Spec.DestRegistry != nil {
		hReg, err := r.getHarborRegistryFromRef(context.Background(),
			originalReplication.Spec.DestRegistry,
			originalReplication.Namespace)
		if err != nil {
			return &legacymodel.ReplicationPolicy{}, err
		}

		newRep.DestRegistry = hReg
	}

	return newRep, nil
}

func addReplicationFilters(originalFilters []registriesv1alpha2.ReplicationFilter) (
	newFilters []*legacymodel.ReplicationFilter, err error) {
	for _, f := range originalFilters {
		filterType, err := enumReplicationFilterType(f.Type)
		if err != nil {
			return []*legacymodel.ReplicationFilter{}, err
		}

		newFilters = append(newFilters, &legacymodel.ReplicationFilter{
			Type:  filterType,
			Value: f.Value,
		})
	}

	return newFilters, nil
}

// getHarborRegistryFromRef retrieves the registryRef and returns a pointer to a goharbor-client Registry Object
func (r *ReplicationReconciler) getHarborRegistryFromRef(ctx context.Context, registryRef *v1.LocalObjectReference,
	namespace string) (*legacymodel.Registry, error) {
	var registry registriesv1alpha2.Registry

	err := r.Client.Get(ctx, client.ObjectKey{Namespace: namespace, Name: registryRef.Name}, &registry)
	if err != nil {
		return nil, err
	}

	if registry.Status.Phase != registriesv1alpha2.RegistryStatusPhaseReady {
		return nil, internal.ErrRegistryNotReady(registry.Name)
	}

	var credential *legacymodel.RegistryCredential
	if registry.Spec.Credential != nil {
		credential, err = helper.ToHarborRegistryCredential(ctx, r.Client, namespace, *registry.Spec.Credential)
		if err != nil {
			return nil, err
		}
	}

	return helper.ToHarborRegistry(registry.Spec, registry.Status.ID, credential), nil
}

// assertDeletedReplication deletes a replication, first ensuring its existence
func (r *ReplicationReconciler) assertDeletedReplication(ctx context.Context, log logr.Logger,
	harborClient *h.RESTClient, replication *registriesv1alpha2.Replication) error {
	receivedReplicationPolicy, err := harborClient.GetReplicationPolicy(ctx, replication.Name)
	if err != nil {
		return err
	}

	err = harborClient.DeleteReplicationPolicy(ctx, receivedReplicationPolicy)
	if err != nil {
		return err
	}

	log.Info("pulling finalizers")
	helper.PullFinalizer(replication, internal.FinalizerName)

	return nil
}
