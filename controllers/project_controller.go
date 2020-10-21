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
	"strconv"
	"time"

	"github.com/mittwald/harbor-operator/controllers/internal"

	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"

	"k8s.io/apimachinery/pkg/runtime"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/mittwald/goharbor-client/v3/apiv2/model"
	legacymodel "github.com/mittwald/goharbor-client/v3/apiv2/model/legacy"
	projectapi "github.com/mittwald/goharbor-client/v3/apiv2/project"

	v1 "k8s.io/api/core/v1"

	"github.com/go-logr/logr"
	"github.com/jinzhu/copier"
	h "github.com/mittwald/goharbor-client/v3/apiv2"
	registriesv1alpha1 "github.com/mittwald/harbor-operator/api/v1alpha1"
	"github.com/mittwald/harbor-operator/controllers/helper"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"sigs.k8s.io/controller-runtime/pkg/source"
)

// ProjectReconciler reconciles a Project object
type ProjectReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=registries.registries.mittwald.de,resources=projects,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=registries.registries.mittwald.de,resources=projects/status,verbs=get;update;patch
func (r *ProjectReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	_ = context.Background()
	_ = r.Log.WithValues("project", req.NamespacedName)

	reqLogger := r.Log.WithValues("Request.Namespace", req.Namespace, "Request.Name", req.Name)
	reqLogger.Info("Reconciling Project")

	now := metav1.Now()
	ctx := context.Background()

	// Fetch the Project instance
	project := &registriesv1alpha1.Project{}

	err := r.Client.Get(ctx, req.NamespacedName, project)
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

	originalProject := project.DeepCopy()

	if project.ObjectMeta.DeletionTimestamp != nil &&
		project.Status.Phase != registriesv1alpha1.ProjectStatusPhaseTerminating {
		project.Status = registriesv1alpha1.ProjectStatus{Phase: registriesv1alpha1.ProjectStatusPhaseTerminating}

		return r.updateProjectCR(ctx, nil, originalProject, project)
	}

	// Fetch the Instance
	harbor, err := internal.FetchReadyHarborInstance(ctx, project.Namespace,
		project.Spec.ParentInstance.Name, r)
	if err != nil {
		if _, ok := err.(internal.ErrInstanceNotFound); ok {
			helper.PullFinalizer(project, internal.FinalizerName)
		} else if _, ok := err.(internal.ErrInstanceNotReady); ok {
			return ctrl.Result{RequeueAfter: 30 * time.Second}, err
		} else {
			project.Status = registriesv1alpha1.ProjectStatus{LastTransition: &now}
		}

		return r.updateProjectCR(ctx, nil, originalProject, project)
	}

	// Build a client to connect to the harbor API
	harborClient, err := internal.BuildClient(ctx, r, harbor)
	if err != nil {
		return ctrl.Result{Requeue: true}, err
	}

	switch project.Status.Phase {
	default:
		return ctrl.Result{}, nil

	case registriesv1alpha1.ProjectStatusPhaseUnknown:
		project.Status = registriesv1alpha1.ProjectStatus{Phase: registriesv1alpha1.ProjectStatusPhaseCreating}

	case registriesv1alpha1.ProjectStatusPhaseCreating:
		if err := r.assertExistingProject(ctx, harborClient, project); err != nil {
			return ctrl.Result{}, err
		}

		helper.PushFinalizer(project, internal.FinalizerName)

		project.Status = registriesv1alpha1.ProjectStatus{Phase: registriesv1alpha1.ProjectStatusPhaseReady}

	case registriesv1alpha1.ProjectStatusPhaseReady:
		err := r.assertExistingProject(ctx, harborClient, project)
		if err != nil {
			return ctrl.Result{}, err
		}

	case registriesv1alpha1.ProjectStatusPhaseTerminating:
		// Delete the project via harbor API
		err := r.assertDeletedProject(ctx, reqLogger, harborClient, project)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	return r.updateProjectCR(ctx, harbor, originalProject, project)
}

func (r *ProjectReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// Create a new controller
	c, err := controller.New("project-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource Project
	err = c.Watch(&source.Kind{Type: &registriesv1alpha1.Project{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// Add a handler to watch for changes in the secondary resource, User
	watchHandler := &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &registriesv1alpha1.Project{},
	}

	// Watch for changes to secondary resources
	err = c.Watch(&source.Kind{Type: &registriesv1alpha1.User{}}, watchHandler)
	if err != nil {
		return err
	}

	return nil
}

// updateProjectCR compares the new CR status and finalizers with the pre-existing ones and updates them accordingly.
func (r *ProjectReconciler) updateProjectCR(ctx context.Context, parentInstance *registriesv1alpha1.Instance, originalProject, project *registriesv1alpha1.Project) (ctrl.Result, error) {
	if originalProject == nil || project == nil {
		return ctrl.Result{},
			fmt.Errorf("cannot update project '%s' because the original project is nil", project.Spec.Name)
	}

	// Update Status
	if !reflect.DeepEqual(originalProject.Status, project.Status) {
		originalProject.Status = project.Status
		if err := r.Client.Status().Update(ctx, originalProject); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Set owner reference
	if (originalProject.OwnerReferences == nil ||
		len(originalProject.OwnerReferences) == 0) &&
		parentInstance != nil {
		err := ctrl.SetControllerReference(parentInstance, originalProject, r.Scheme)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	// Update Finalizer
	if !reflect.DeepEqual(originalProject.Finalizers, project.Finalizers) {
		originalProject.SetFinalizers(project.Finalizers)
	}

	if err := r.Client.Update(ctx, originalProject); err != nil {
		return ctrl.Result{Requeue: true}, err
	}

	return ctrl.Result{}, nil
}

// assertDeletedProject deletes a Harbor project, first ensuring its existence.
func (r *ProjectReconciler) assertDeletedProject(ctx context.Context, log logr.Logger, harborClient *h.RESTClient,
	project *registriesv1alpha1.Project) error {
	_, err := harborClient.GetProjectByName(ctx, project.Name)
	if err != nil {
		return err
	}

	log.Info("pulling finalizers", project.Name, project.Namespace)
	helper.PullFinalizer(project, internal.FinalizerName)

	return nil
}

// assertExistingProject
// Check Harbor projects and their components for their existence,
// create and delete either of those to match the specification.
func (r *ProjectReconciler) assertExistingProject(ctx context.Context, harborClient *h.RESTClient,
	project *registriesv1alpha1.Project) error {
	heldRepo, err := harborClient.GetProjectByName(ctx, project.Spec.Name)

	if errors.Is(err, &projectapi.ErrProjectNotFound{}) {
		_, err := harborClient.NewProject(ctx, project.Spec.Name, project.Spec.StorageLimit)

		return err
	} else if err != nil {
		return err
	}

	return r.ensureProject(ctx, heldRepo, harborClient, project)
}

// generateProjectMetadata constructs the project metadata for a Harbor project
func (r *ProjectReconciler) generateProjectMetadata(
	projectMeta *registriesv1alpha1.ProjectMetadata) *model.ProjectMetadata {
	autoScan := strconv.FormatBool(projectMeta.AutoScan)
	enableContentTrust := strconv.FormatBool(projectMeta.EnableContentTrust)
	preventVul := strconv.FormatBool(projectMeta.PreventVul)
	public := strconv.FormatBool(projectMeta.Public)
	reuseSysCVEAllowlist := strconv.FormatBool(projectMeta.ReuseSysSVEWhitelist)
	retentionID := strconv.Itoa(projectMeta.RetentionID)

	pm := model.ProjectMetadata{
		AutoScan:             &autoScan,
		EnableContentTrust:   &enableContentTrust,
		PreventVul:           &preventVul,
		Public:               public,
		ReuseSysCveAllowlist: &reuseSysCVEAllowlist,
		Severity:             &projectMeta.Severity,
		RetentionID:          &retentionID,
	}

	return &pm
}

// reconcileProjectMembers reconciles the user-defined project members
// for a project based on an existing Harbor user
func (r *ProjectReconciler) reconcileProjectMembers(ctx context.Context, project *registriesv1alpha1.Project,
	harborClient *h.RESTClient, heldProject *model.Project) error {
	// List Harbor project members
	members, err := harborClient.ListProjectMembers(ctx, heldProject)
	if err != nil {
		return err
	}

	// Range over the defined member users of a project
	for _, memberRequestUser := range project.Spec.MemberRequests {
		// Fetch the user resource
		user, err := r.getUserFromRef(memberRequestUser.User, project.Namespace)
		if err != nil {
			return err
		}

		// Check for an existing user
		harborUser, err := harborClient.GetUser(ctx, user.Spec.Name)
		if err != nil {
			return err
		}

		roleID := memberRequestUser.Role.ID()

		member := getMemberUserFromList(members, user)

		if member == nil {
			err = harborClient.AddProjectMember(ctx, heldProject, harborUser, int(roleID))
			if err != nil {
				return err
			}

			break
		} else {
			// Update the Harbor project member
			if roleID == registriesv1alpha1.MemberRoleID(member.RoleID) {
				break
			}
			if err := harborClient.UpdateProjectMemberRole(ctx, heldProject, harborUser, int(roleID)); err != nil {
				return err
			}
		}
	}

	return nil
}

func (r *ProjectReconciler) getUserFromRef(userRef v1.LocalObjectReference,
	namespace string) (*registriesv1alpha1.User, error) {
	var user registriesv1alpha1.User
	err := r.Client.Get(context.Background(), client.ObjectKey{Name: userRef.Name, Namespace: namespace}, &user)

	return &user, err
}

// getMemberUserFromList returns a project member from a list of members, filtered by the username
func getMemberUserFromList(members []*legacymodel.ProjectMemberEntity,
	user *registriesv1alpha1.User) *legacymodel.ProjectMemberEntity {
	for i := range members {
		if members[i].EntityName == user.Spec.Name {
			return members[i]
		}
	}

	return nil
}

// ensureProject triggers reconciliation of project members
// and compares the state of the CR object with the project held by Harbor
func (r *ProjectReconciler) ensureProject(ctx context.Context, heldProject *model.Project,
	harborClient *h.RESTClient, originalProject *registriesv1alpha1.Project) error {
	newProject := &model.Project{}
	// Copy the spec of the project held by Harbor into a new object of the same type *harbor.Project
	err := copier.Copy(&newProject, &heldProject)
	if err != nil {
		return err
	}

	err = r.reconcileProjectMembers(ctx, originalProject, harborClient, heldProject)
	if err != nil {
		return err
	}

	if originalProject.Status.ID != heldProject.ProjectID {
		originalProject.Status.ID = heldProject.ProjectID
		if err := r.Client.Status().Update(ctx, originalProject); err != nil {
			return err
		}
	}

	newProject.Metadata = r.generateProjectMetadata(&originalProject.Spec.Metadata)

	if newProject != heldProject {
		return harborClient.UpdateProject(ctx, newProject, originalProject.Spec.StorageLimit)
	}

	return nil
}
