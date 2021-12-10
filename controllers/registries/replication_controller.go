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
	controllererrors "github.com/mittwald/harbor-operator/controllers/registries/errors"
	"reflect"
	"time"

	clienterrors "github.com/mittwald/goharbor-client/v5/apiv2/pkg/errors"

	h "github.com/mittwald/goharbor-client/v5/apiv2"
	"github.com/mittwald/goharbor-client/v5/apiv2/model"
	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/mittwald/harbor-operator/apis/registries/v1alpha2"
	"github.com/mittwald/harbor-operator/controllers/registries/helper"
	"github.com/mittwald/harbor-operator/controllers/registries/internal"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ReplicationReconciler reconciles a Replication object
type ReplicationReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=registries.mittwald.de,resources=replications,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=registries.mittwald.de,resources=replications/status,verbs=get;update;patch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *ReplicationReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	reqLogger := r.Log.WithValues("replication", req.NamespacedName)
	reqLogger.Info("Reconciling Replication")

	// Fetch the Replication instance
	replication := &v1alpha2.Replication{}

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

	patch := client.MergeFrom(replication.DeepCopy())

	if replication.ObjectMeta.DeletionTimestamp != nil && replication.Status.Phase != v1alpha2.ReplicationStatusPhaseTerminating {
		replication.Status.Phase = v1alpha2.ReplicationStatusPhaseTerminating
		return ctrl.Result{}, r.Client.Status().Patch(ctx, replication, patch)
	}

	// Fetch the goharbor instance if it exists and is properly set up.
	// If the above does not apply, pull the finalizer from the replication object.
	harbor, err := internal.GetOperationalHarborInstance(ctx, client.ObjectKey{
		Namespace: replication.Namespace,
		Name:      replication.Spec.ParentInstance.Name,
	}, r.Client)
	if err != nil {
		switch err.Error() {
		case controllererrors.ErrInstanceNotInstalledMsg:
			reqLogger.Info("waiting till harbor instance is installed")
			return ctrl.Result{RequeueAfter: 30*time.Second}, nil
		case controllererrors.ErrInstanceNotFoundMsg:
			controllerutil.RemoveFinalizer(replication, internal.FinalizerName)
			fallthrough
		default:
			return ctrl.Result{}, err
		}
	}

	// Set OwnerReference to the parent harbor instance
	err = ctrl.SetControllerReference(harbor, replication, r.Scheme)
	if err != nil {
		return ctrl.Result{}, err
	}
	if err := r.Client.Status().Patch(ctx, replication, patch); err != nil {
		return ctrl.Result{}, err
	}

	// Build a client to connect to the harbor API
	harborClient, err := internal.BuildClient(ctx, r.Client, harbor)
	if err != nil {
		return ctrl.Result{}, err
	}

	// Check the Harbor API if it's reporting as healthy
	err = internal.AssertHealthyHarborInstance(ctx, harborClient)
	if err != nil {
		reqLogger.Info("waiting till harbor instance is healthy")
		return ctrl.Result{RequeueAfter: 30*time.Second}, nil
	}

	switch replication.Status.Phase {
	default:
		return ctrl.Result{}, nil

	case v1alpha2.ReplicationStatusPhaseUnknown:
		replication.Status.Phase = v1alpha2.ReplicationStatusPhaseCreating
		replication.Status.Message = "replication is about to be created"

	case v1alpha2.ReplicationStatusPhaseCreating:
		// Fetch the parent registry the replication should be owned by.
		registry := v1alpha2.Registry{}
		var registryName string
		if replication.Spec.SrcRegistry != nil {
			registryName = replication.Spec.SrcRegistry.Name
		} else if replication.Spec.DestRegistry != nil {
			registryName = replication.Spec.DestRegistry.Name
		}

		err = r.Client.Get(ctx, client.ObjectKey{Namespace: replication.Namespace, Name: registryName}, &registry)
		if err != nil {
			return ctrl.Result{}, err
		}

		if registry.Status.Phase == v1alpha2.RegistryStatusPhaseTerminating {
			replication.Status.Phase = v1alpha2.ReplicationStatusPhaseTerminating
			return ctrl.Result{}, r.Client.Status().Patch(ctx, replication, patch)
		}

		// Set OwnerReference to the parent (source- or destination) registry
		err = ctrl.SetControllerReference(&registry, replication, r.Scheme)
		if err != nil {
			return ctrl.Result{}, err
		}

		if err := r.assertExistingReplication(ctx, harborClient, replication, patch); err != nil {
			return ctrl.Result{}, err
		}

		controllerutil.AddFinalizer(replication, internal.FinalizerName)
		if err := r.Client.Patch(ctx, replication, patch); err != nil {
			return ctrl.Result{}, err
		}

		if replication.Spec.TriggerAfterCreation {
			replExec := &model.StartReplicationExecution{
				PolicyID: replication.Status.ID,
			}

			if err := harborClient.TriggerReplicationExecution(ctx, replExec); err != nil {
				reqLogger.Info(fmt.Sprintf("replication execution after creation could not be triggered: %s", err))
				return ctrl.Result{}, err
			}

			replication.Status.Phase = v1alpha2.ReplicationStatusPhaseManualExecutionRunning

			return ctrl.Result{}, r.Client.Status().Patch(ctx, replication, patch)
		}

		replication.Status.Phase = v1alpha2.ReplicationStatusPhaseCompleted

	case v1alpha2.ReplicationStatusPhaseManualExecutionRunning:
		running, err := r.reconcileRunningReplicationExecution(ctx, replication, harborClient)
		if err != nil {
			reqLogger.Info(fmt.Sprintf("replication execution failed: %s", err))
			replication.Status.Phase = v1alpha2.ReplicationStatusPhaseManualExecutionFailed
		}
		if !running {
			replication.Status.Phase = v1alpha2.ReplicationStatusPhaseManualExecutionFinished
		}
		return ctrl.Result{RequeueAfter: 1 * time.Minute}, r.Client.Status().Patch(ctx, replication, patch)

	case v1alpha2.ReplicationStatusPhaseManualExecutionFinished:
		err := r.reconcileFinishedReplicationExecution(ctx, replication, harborClient)
		if err != nil {
			reqLogger.Info(fmt.Sprintf("replication execution failed: %s", err))
			replication.Status.Phase = v1alpha2.ReplicationStatusPhaseManualExecutionFailed
		}
		replication.Status.Phase = v1alpha2.ReplicationStatusPhaseCompleted

	case v1alpha2.ReplicationStatusPhaseManualExecutionFailed:
		replication.Status.Phase = v1alpha2.ReplicationStatusPhaseCreating

	case v1alpha2.ReplicationStatusPhaseCompleted:
		err := r.assertExistingReplication(ctx, harborClient, replication, patch)
		if err != nil {
			return ctrl.Result{}, err
		}

	case v1alpha2.ReplicationStatusPhaseTerminating:
		// Delete the replication via harbor API
		if err := internal.AssertDeletedReplication(ctx, reqLogger, harborClient, replication); err != nil {
			return ctrl.Result{}, err
		}

		return ctrl.Result{}, r.Client.Patch(ctx, replication, patch)
	}

	return ctrl.Result{}, r.Client.Status().Patch(ctx, replication, patch)
}

// reconcileRunningReplicationExecution fetches the newest replication execution of
// the replication and checks whether it is still running or not.
// Returns an error if no replication execution could be found for the replication policy.
func (r *ReplicationReconciler) reconcileRunningReplicationExecution(ctx context.Context, replication *v1alpha2.Replication, harborClient *h.RESTClient) (bool, error) {
	triggerType := v1alpha2.ReplicationTriggerTypeManual

	executions, err := harborClient.ListReplicationExecutions(ctx, &replication.Status.ID, nil, &triggerType)
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
func (r *ReplicationReconciler) reconcileFinishedReplicationExecution(ctx context.Context, replication *v1alpha2.Replication, harborClient *h.RESTClient) error {
	triggerType := v1alpha2.ReplicationTriggerTypeManual

	executions, err := harborClient.ListReplicationExecutions(ctx, &replication.Status.ID, nil, &triggerType)
	if err != nil {
		return fmt.Errorf("fetching successful replication executions failed for replication policy %s: %s", replication.Name, err)
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
		For(&v1alpha2.Replication{}).
		Complete(r)
}

// getNewestReplicationExecutionID takes a slice of replication executions and returns the one with the highest ID.
func getNewestReplicationExecutionID(executions []*model.ReplicationExecution) int64 {
	max := executions[0].ID
	for i := range executions {
		if max < executions[i].ID {
			max = executions[i].ID
		}
	}
	return max
}

// assertExistingReplication checks a harbor replication for existence and creates it accordingly.
func (r *ReplicationReconciler) assertExistingReplication(ctx context.Context, harborClient *h.RESTClient,
	replication *v1alpha2.Replication, patch client.Patch) error {

	rReq, err := r.buildReplicationFromCR(replication)
	if err != nil {
		return err
	}

	heldReplication, err := harborClient.GetReplicationPolicyByName(ctx, replication.Spec.Name)
	if err != nil {
		if errors.Is(&clienterrors.ErrNotFound{}, err) {
			if err := harborClient.NewReplicationPolicy(ctx,
				rReq.DestRegistry,
				rReq.SrcRegistry,
				rReq.ReplicateDeletion,
				rReq.Override,
				rReq.Enabled,
				rReq.Filters,
				rReq.Trigger,
				rReq.DestNamespace,
				rReq.Description,
				rReq.Name); err != nil {
				return err
			}
		} else {
			return err
		}
	}

	if replication.Spec.DestRegistry != nil {
		replication.Status.Destination = heldReplication.DestRegistry.Name
		replication.Status.Source = "harbor"
	} else if replication.Spec.SrcRegistry != nil {
		replication.Status.Source = heldReplication.SrcRegistry.Name
		replication.Status.Destination = "harbor"
	}

	// Construct a replication from the CR spec
	newRep, err := r.buildReplicationFromCR(replication)
	if err != nil {
		return err
	}

	// Compare the replications and update accordingly
	if !reflect.DeepEqual(heldReplication, newRep) {
		return harborClient.UpdateReplicationPolicy(ctx, newRep, heldReplication.ID)
	}

	return r.Client.Status().Patch(ctx, replication, patch)
}

func enumReplicationTrigger(receivedTrigger string) (string, error) {
	if receivedTrigger == "" {
		return "", errors.New("empty replication trigger provided")
	}

	switch receivedTrigger {
	case v1alpha2.ReplicationTriggerTypeEventBased,
		v1alpha2.ReplicationTriggerTypeManual,
		v1alpha2.ReplicationTriggerTypeScheduled:
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
	case v1alpha2.ReplicationFilterTypeLabel, v1alpha2.ReplicationFilterTypeName,
		v1alpha2.ReplicationFilterTypeResource, v1alpha2.ReplicationFilterTypeTag:
		return filterType, nil
	default:
		return "", fmt.Errorf("invalid replication filter type provided: '%s'", filterType)
	}
}

// buildReplicationFromCR returns an API conformed ReplicationPolicy object
func (r *ReplicationReconciler) buildReplicationFromCR(originalReplication *v1alpha2.Replication) (
	*model.ReplicationPolicy, error) {
	newRep := &model.ReplicationPolicy{
		Description:       originalReplication.Spec.Description,
		DestNamespace:     originalReplication.Spec.DestNamespace,
		Enabled:           originalReplication.Spec.Enabled,
		Name:              originalReplication.Spec.Name,
		Override:          originalReplication.Spec.Override,
		ReplicateDeletion: originalReplication.Spec.ReplicateDeletion,
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

		newRep.Trigger = &model.ReplicationTrigger{
			Type: triggerType,
		}

		if originalReplication.Spec.Trigger.Settings != nil {
			newRep.Trigger.TriggerSettings = &model.ReplicationTriggerSettings{
				Cron: originalReplication.Spec.Trigger.Settings.Cron,
			}
		}
	}

	if originalReplication.Spec.SrcRegistry != nil && originalReplication.Spec.DestRegistry != nil {
		return &model.ReplicationPolicy{},
			fmt.Errorf("both dest_registry and src_registry are set! Please specify only one of them")
	}

	if originalReplication.Spec.SrcRegistry != nil {
		hReg, err := r.getHarborRegistryFromRef(context.Background(),
			originalReplication.Spec.SrcRegistry,
			originalReplication.Namespace)
		if err != nil {
			return &model.ReplicationPolicy{}, err
		}

		newRep.SrcRegistry = hReg
	} else if originalReplication.Spec.DestRegistry != nil {
		hReg, err := r.getHarborRegistryFromRef(context.Background(),
			originalReplication.Spec.DestRegistry,
			originalReplication.Namespace)
		if err != nil {
			return &model.ReplicationPolicy{}, err
		}

		newRep.DestRegistry = hReg
	}

	return newRep, nil
}

func addReplicationFilters(originalFilters []v1alpha2.ReplicationFilter) (
	newFilters []*model.ReplicationFilter, err error) {
	for _, f := range originalFilters {
		filterType, err := enumReplicationFilterType(f.Type)
		if err != nil {
			return []*model.ReplicationFilter{}, err
		}

		newFilters = append(newFilters, &model.ReplicationFilter{
			Type:  filterType,
			Value: f.Value,
		})
	}

	return newFilters, nil
}

// getHarborRegistryFromRef retrieves the registryRef and returns a pointer to a goharbor-client Registry Object
func (r *ReplicationReconciler) getHarborRegistryFromRef(ctx context.Context, registryRef *corev1.LocalObjectReference,
	namespace string) (*model.Registry, error) {
	var registry v1alpha2.Registry

	err := r.Client.Get(ctx, client.ObjectKey{Namespace: namespace, Name: registryRef.Name}, &registry)
	if err != nil {
		return nil, err
	}

	var credential *model.RegistryCredential
	if registry.Spec.Credential != nil {
		credential, err = helper.ToHarborRegistryCredential(ctx, r.Client, namespace, *registry.Spec.Credential)
		if err != nil {
			return nil, err
		}
	}

	return helper.ToHarborRegistry(registry.Spec, registry.Status.ID, credential), nil
}
