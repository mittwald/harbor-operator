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
	"reflect"
	"strconv"
	"time"

	"github.com/go-logr/logr"
	h "github.com/mittwald/goharbor-client/v3/apiv2"
	"github.com/mittwald/goharbor-client/v3/apiv2/model"
	projectapi "github.com/mittwald/goharbor-client/v3/apiv2/project"
	registriesv1alpha2 "github.com/mittwald/harbor-operator/apis/registries/v1alpha2"
	"github.com/mittwald/harbor-operator/controllers/registries/helper"
	"github.com/mittwald/harbor-operator/controllers/registries/internal"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// RetentionReconciler reconciles a Retention object
type RetentionReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// reconcileRetention reconciles a retention.
// If a project is found via the ProjectRef in a Retention's spec,
// the retention is created and updated as per the configuration.

// When a Retention object is deleted it is disabled on the Harbor API.
// If no retention policy is specified in spec but observed via the API, it gets disabled.
// The retention is written into to the project status.
// +kubebuilder:rbac:groups=registries.mittwald.de,resources=retentions,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=registries.mittwald.de,resources=retentions/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=registries.mittwald.de,resources=retentions/finalizers,verbs=update
func (r *RetentionReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	reqLogger := r.Log.WithValues("Request.Namespace", req.Namespace, "Request.Name", req.Name)
	reqLogger.Info("Reconciling Retention")

	now := metav1.Now()
	ctx := context.Background()

	retention := &registriesv1alpha2.Retention{}

	err := r.Client.Get(ctx, req.NamespacedName, retention)
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

	originalRetention := retention.DeepCopy()

	if retention.ObjectMeta.DeletionTimestamp != nil &&
		retention.Status.Phase != registriesv1alpha2.RetentionStatusPhaseTerminating {
		retention.Status = registriesv1alpha2.RetentionStatus{
			Phase:          registriesv1alpha2.RetentionStatusPhaseTerminating,
			LastTransition: &now,
		}

		_ = r.Client.Status().Update(ctx, retention)
	}

	// Fetch the Instance
	harbor, err := internal.FetchReadyHarborInstance(ctx,
		retention.Namespace,
		retention.Spec.ParentInstance.Name,
		r.Client)
	if err != nil {
		if _, ok := err.(internal.ErrInstanceNotFound); ok {
			helper.PullFinalizer(retention, internal.FinalizerName)
		} else if _, ok := err.(internal.ErrInstanceNotReady); ok {
			return ctrl.Result{RequeueAfter: 30 * time.Second}, err
		} else {
			retention.Status = registriesv1alpha2.RetentionStatus{LastTransition: &now}
		}
		return r.updateRetentionCR(ctx, originalRetention, retention)
	}

	// Build a client to connect to the harbor API
	harborClient, err := internal.BuildClient(ctx, r, harbor)
	if err != nil {
		return ctrl.Result{Requeue: true}, err
	}

	switch retention.Status.Phase {
	default:
		return ctrl.Result{}, nil

	case registriesv1alpha2.RetentionStatusPhaseUnknown:
		retention.Status = registriesv1alpha2.RetentionStatus{
			Phase:          registriesv1alpha2.RetentionStatusPhaseCreating,
			LastTransition: &now,
		}
	case registriesv1alpha2.RetentionStatusPhaseCreating:
		if err := r.assertExistingRetention(ctx, harborClient, retention); err != nil {
			return ctrl.Result{}, err
		}

		helper.PushFinalizer(retention, internal.FinalizerName)

		return r.updateRetentionCR(ctx, originalRetention, retention)
	case registriesv1alpha2.RetentionStatusPhaseActive:
		if err := r.assertExistingRetention(ctx, harborClient, retention); err != nil {
			return ctrl.Result{}, err
		}

		return r.updateRetentionCR(ctx, originalRetention, retention)
	case registriesv1alpha2.RetentionStatusPhaseTerminating:
		if err := r.assertDisabledRetentionPolicy(ctx, harborClient, retention); err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

// getRetentionIDIfExists returns the pointer to a retention ID if contained in a projects metadata.
// Returns nil if no retention ID is found.
func (r *RetentionReconciler) getRetentionIDIfExists(ctx context.Context, harborClient *h.RESTClient, harborProject *model.Project) (*int64, error) {
	idStr, err := harborClient.GetProjectMetadataValue(ctx, int64(harborProject.ProjectID), projectapi.ProjectMetadataKeyRetentionID)
	if err != nil {
		if errors.Is(err, &projectapi.ErrProjectMetadataValueRetentionIDUndefined{}) {
			return nil, nil
		}
		return nil, err
	}

	retentionID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return nil, err
	}

	return &retentionID, nil
}

// updateRetentionCR compares the new CR status and finalizers with the existing and updates them accordingly.
func (r *RetentionReconciler) updateRetentionCR(ctx context.Context, originalRetention,
	retention *registriesv1alpha2.Retention) (ctrl.Result, error) {
	if originalRetention == nil || retention == nil {
		return ctrl.Result{},
			fmt.Errorf("cannot update retention '%s' because the original retention is nil", retention.Spec.Name)
	}

	// Update status
	if !reflect.DeepEqual(originalRetention.Status, retention.Status) {
		originalRetention.Status = retention.Status
		if err := r.Client.Status().Update(ctx, originalRetention); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Update owner references
	if !reflect.DeepEqual(originalRetention.OwnerReferences, retention.OwnerReferences) {
		originalRetention.SetOwnerReferences(retention.OwnerReferences)
	}

	// Update Finalizer
	if !reflect.DeepEqual(originalRetention.Finalizers, retention.Finalizers) {
		originalRetention.SetFinalizers(retention.Finalizers)
	}

	if err := r.Client.Update(ctx, originalRetention); err != nil {
		return ctrl.Result{Requeue: true}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *RetentionReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&registriesv1alpha2.Retention{}).
		Complete(r)
}

// createRetentionPolicy constructs a retention policy object that is passed to the REST client.
// Asserts an existing retention policy for every ProjectReference provided in the Retention spec.
// Note: retention policies are not deleted on the Harbor API, they get disabled.
// If a project previously had a retention policy set up, it will be updated instead of newly created.
func (r *RetentionReconciler) assertExistingRetention(ctx context.Context, harborClient *h.RESTClient, retention *registriesv1alpha2.Retention) error {
	var observedProjectReferences []registriesv1alpha2.ProjectReference
	for _, projectRef := range retention.Spec.ProjectReferences {
		var project registriesv1alpha2.Project

		projectExists, err := helper.ObjExists(ctx, r.Client, projectRef.Name, retention.Namespace, &project)
		if err != nil {
			return err
		}

		if !projectExists {
			return fmt.Errorf("the specified project does not exist: %q", projectRef.Name)
		}

		// Fetch the project by the name provided via ProjectReference
		harborProject, err := harborClient.GetProjectByName(ctx, projectRef.Name)
		if err != nil {
			return err
		}

		// Fetch the retention ID if it exists.
		retentionID, err := r.getRetentionIDIfExists(ctx, harborClient, harborProject)
		if err != nil {
			return err
		}

		desiredRp := helper.ToHarborRetentionPolicy(retention, retentionID)

		// Assert that there has previously been a retention policy for this project,
		// if the "retentionID" pointer is not nil.
		// Note: This condition is also true if the Retention CR got updated.
		if retentionID != nil {

			err = harborClient.UpdateRetentionPolicy(ctx, desiredRp)
		} else {
			err = harborClient.NewRetentionPolicy(ctx, desiredRp)
			if err != nil {
				return err
			}
		}

		// Fetch the existing retention policy to add it to the status of the CR.
		observedRetentionPolicy, err := harborClient.GetRetentionPolicyByProject(ctx, harborProject)
		if err != nil {
			return err
		}

		observedProjectReferences = append(observedProjectReferences, registriesv1alpha2.ProjectReference{
			Name:        project.Name,
			ProjectID:   harborProject.ProjectID,
			RetentionID: int32(observedRetentionPolicy.ID),
		})
	}

	now := metav1.Now()

	retention.Status = registriesv1alpha2.RetentionStatus{
		Phase:             registriesv1alpha2.RetentionStatusPhaseActive,
		LastTransition:    &now,
		ProjectReferences: observedProjectReferences,
	}

	return nil
}

func (r *RetentionReconciler) assertDisabledRetentionPolicy(ctx context.Context, harborClient *h.RESTClient,
	retention *registriesv1alpha2.Retention) error {

	// Fetch the held retention and disable it

	// harborClient.DisableRetentionPolicy(ctx, rp)
	//
	// // The retention policy is disabled, write the state to the project's status.
	// originalProject.Status.RetentionPolicy = desiredRp

	return nil
}
