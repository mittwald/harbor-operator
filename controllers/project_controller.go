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
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"strconv"
	"time"

	corev1 "k8s.io/api/core/v1"

	"github.com/mittwald/harbor-operator/controllers/internal"

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
	registriesv1alpha2 "github.com/mittwald/harbor-operator/api/v1alpha2"
	"github.com/mittwald/harbor-operator/controllers/helper"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	reqLogger := r.Log.WithValues("Request.Namespace", req.Namespace, "Request.Name", req.Name)
	reqLogger.Info("Reconciling Project")

	now := metav1.Now()
	ctx := context.Background()

	// Fetch the Project instance
	project := &registriesv1alpha2.Project{}

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
		project.Status.Phase != registriesv1alpha2.ProjectStatusPhaseTerminating {
		project.Status = registriesv1alpha2.ProjectStatus{Phase: registriesv1alpha2.ProjectStatusPhaseTerminating}

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
			project.Status = registriesv1alpha2.ProjectStatus{LastTransition: &now}
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

	case registriesv1alpha2.ProjectStatusPhaseUnknown:
		project.Status = registriesv1alpha2.ProjectStatus{Phase: registriesv1alpha2.ProjectStatusPhaseCreating}

	case registriesv1alpha2.ProjectStatusPhaseCreating:
		if err := r.assertExistingProject(ctx, harborClient, project); err != nil {
			return ctrl.Result{}, err
		}

		helper.PushFinalizer(project, internal.FinalizerName)

		project.Status = registriesv1alpha2.ProjectStatus{Phase: registriesv1alpha2.ProjectStatusPhaseReady}

	case registriesv1alpha2.ProjectStatusPhaseReady:
		err := r.assertExistingProject(ctx, harborClient, project)
		if err != nil {
			return ctrl.Result{}, err
		}

	case registriesv1alpha2.ProjectStatusPhaseTerminating:
		// Delete the project via harbor API
		err := r.assertDeletedProject(ctx, reqLogger, harborClient, project)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	return r.updateProjectCR(ctx, harbor, originalProject, project)
}

func (r *ProjectReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&registriesv1alpha2.Project{}).
		Owns(&registriesv1alpha2.User{}).
		WithOptions(controller.Options{
			MaxConcurrentReconciles: 1,
		}).
		Complete(r)
}

// updateProjectCR compares the new CR status and finalizers with the pre-existing ones and updates them accordingly.
func (r *ProjectReconciler) updateProjectCR(ctx context.Context, parentInstance *registriesv1alpha2.Instance, originalProject, project *registriesv1alpha2.Project) (ctrl.Result, error) {
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
	project *registriesv1alpha2.Project) error {
	p, err := harborClient.GetProjectByName(ctx, project.Name)
	if err != nil {
		if errors.Is(&projectapi.ErrProjectUnknownResource{}, err) {
			return nil
		}
		return err
	}

	if err := harborClient.DeleteProject(ctx, p); err != nil {
		if errors.Is(&projectapi.ErrProjectMismatch{}, err) {
			return nil
		}
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
	project *registriesv1alpha2.Project) error {
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
	projectMeta *registriesv1alpha2.ProjectMetadata) *model.ProjectMetadata {
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

func (r *ProjectReconciler) projectMemberExists(members []*legacymodel.ProjectMemberEntity, requestedMember *registriesv1alpha2.User) bool {
	for i := range members {
		if members[i].EntityName == requestedMember.Spec.Name {
			return true
		}
	}
	return false
}

// addProjectMemberStatus
func (r *ProjectReconciler) addProjectMemberStatus(ctx context.Context, project *registriesv1alpha2.Project, request *registriesv1alpha2.MemberRequest) error {
	for i := range project.Status.Members {
		if project.Status.Members[i].Name == request.User.Name {
			continue
		}
	}
	project.Status.Members = append(project.Status.Members, request.User)
	return r.Client.Status().Update(ctx, project)
}

// deleteProjectMemberStatus
func (r *ProjectReconciler) deleteProjectMemberStatus(ctx context.Context, project *registriesv1alpha2.Project, request *corev1.LocalObjectReference) error {
	for i := range project.Status.Members {
		if project.Status.Members[i].Name == request.Name {
			project.Status.Members = append(project.Status.Members[:i], project.Status.Members[i+1:]...)
		}
	}

	return r.Client.Status().Update(ctx, project)
}

// projectMemberShouldExist
func (r *ProjectReconciler) projectMemberShouldExist(existing corev1.LocalObjectReference, desired []registriesv1alpha2.MemberRequest) bool {
	for i := range desired {
		if existing.Name == desired[i].User.Name {
			return true
		}
	}

	return false
}

func (r *ProjectReconciler) reconcileProjectMembers(ctx context.Context, project *registriesv1alpha2.Project,
	harborClient *h.RESTClient, harborProject *model.Project) error {
	for i := range project.Spec.MemberRequests {
		userCR, err := r.getUserCRFromRef(ctx, project.Spec.MemberRequests[i].User, project.Namespace)
		if err != nil {
			return fmt.Errorf("the user specified in project %s's list of member requests does not exist: %s", project.Spec.MemberRequests[i].User.Name, err)
		}

		harborUser, err := harborClient.GetUser(ctx, userCR.Spec.Name)
		if err != nil {
			return err
		}

		// Look up if the user exists as a project member. If not, add the user as project member.
		harborProjectMembers, err := harborClient.ListProjectMembers(ctx, harborProject)
		if err != nil {
			return err
		}
		if !r.projectMemberExists(harborProjectMembers, userCR) {
			err = harborClient.AddProjectMember(ctx, harborProject, harborUser, int(project.Spec.MemberRequests[i].Role.ID()))
			if err != nil {
				return err
			}
			// Once the member user's existence is certain, append it to the project CR's status
			if err = r.addProjectMemberStatus(ctx, project, &project.Spec.MemberRequests[i]); err != nil {
				return err
			}
		}
	}

	// Range over the references in the project status and compare them to the spec.
	// This determines if a user should be absent.
	for i := range project.Status.Members {
		if !r.projectMemberShouldExist(project.Status.Members[i], project.Spec.MemberRequests) {
			userCR, err := r.getUserCRFromRef(ctx, project.Status.Members[i], project.Namespace)
			if err != nil {
				return fmt.Errorf("the user specified in project %s's list of existing members does not exist: %s", project.Status.Members[i].Name, err)
			}

			harborUser, err := harborClient.GetUser(ctx, userCR.Spec.Name)
			if err != nil {
				return err
			}

			// Look up if the user exists as a project member.
			harborProjectMembers, err := harborClient.ListProjectMembers(ctx, harborProject)
			if err != nil {
				return err
			}
			if r.projectMemberExists(harborProjectMembers, userCR) {
				err = harborClient.DeleteProjectMember(ctx, harborProject, harborUser)
				if err != nil {
					return err
				}
				if err = r.deleteProjectMemberStatus(ctx, project, &project.Status.Members[i]); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func (r *ProjectReconciler) getUserCRFromRef(ctx context.Context, userRef v1.LocalObjectReference,
	namespace string) (*registriesv1alpha2.User, error) {
	var user registriesv1alpha2.User
	err := r.Client.Get(ctx, client.ObjectKey{Name: userRef.Name, Namespace: namespace}, &user)

	return &user, err
}

// ensureProject triggers reconciliation of project members
// and compares the state of the CR object with the project held by Harbor
func (r *ProjectReconciler) ensureProject(ctx context.Context, heldProject *model.Project,
	harborClient *h.RESTClient, originalProject *registriesv1alpha2.Project) error {
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
