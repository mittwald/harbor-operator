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
	"time"

	controllererrors "github.com/mittwald/harbor-operator/controllers/registries/errors"

	clienterrors "github.com/mittwald/goharbor-client/v5/apiv2/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"

	"github.com/mittwald/harbor-operator/apis/registries/v1alpha2"

	"github.com/mittwald/harbor-operator/controllers/registries/internal"

	"k8s.io/apimachinery/pkg/runtime"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/go-logr/logr"
	"github.com/jinzhu/copier"
	h "github.com/mittwald/goharbor-client/v5/apiv2"
	"github.com/mittwald/goharbor-client/v5/apiv2/model"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
)

// ProjectReconciler reconciles a Project object
type ProjectReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

var (
	RegistryNotReadyError string = "RegistryNotReady"
	UserNotReadyError     string = "UserNotReady"
)

// +kubebuilder:rbac:groups=registries.mittwald.de,resources=projects,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=registries.mittwald.de,resources=projects/status,verbs=get;update;patch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *ProjectReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	reqLogger := r.Log.WithValues("Request.Namespace", req.Namespace, "Request.Name", req.Name)
	reqLogger.Info("Reconciling Project")

	// Fetch the Project instance
	project := &v1alpha2.Project{}

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

	original := project.DeepCopy()
	patch := client.MergeFrom(original)

	if project.ObjectMeta.DeletionTimestamp != nil &&
		project.Status.Phase != v1alpha2.ProjectStatusPhaseTerminating {
		project.Status = v1alpha2.ProjectStatus{Phase: v1alpha2.ProjectStatusPhaseTerminating}

		return ctrl.Result{}, r.Client.Status().Patch(ctx, project, patch)
	}

	// Fetch the goharbor instance if it exists and is properly set up.
	// If the above does not apply, pull the finalizer from the project object.
	harbor, err := internal.GetOperationalHarborInstance(ctx, client.ObjectKey{
		Namespace: project.Namespace,
		Name:      project.Spec.ParentInstance.Name,
	}, r.Client)
	if err != nil {
		switch err.Error() {
		case controllererrors.ErrInstanceNotInstalledMsg:
			reqLogger.Info("waiting till harbor instance is installed")
			return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
		case controllererrors.ErrInstanceNotFoundMsg:
			controllerutil.RemoveFinalizer(project, internal.FinalizerName)
			fallthrough
		default:
			return ctrl.Result{}, err
		}
	}

	// Set OwnerReference to the parent harbor instance
	err = ctrl.SetControllerReference(harbor, project, r.Scheme)
	if err != nil {
		return ctrl.Result{}, err
	}
	if !reflect.DeepEqual(original.ObjectMeta.OwnerReferences, project.ObjectMeta.OwnerReferences) {
		if err := r.Client.Patch(ctx, project, patch); err != nil {
			return ctrl.Result{}, err
		}
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
		return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
	}

	if err := r.assertProxyCacheRequirementsReady(ctx, harborClient, project, reqLogger); err != nil {
		if err.Error() == RegistryNotReadyError {
			return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
		}
		return ctrl.Result{}, err
	}

	if err := r.assertUserRequirementsReady(ctx, project, reqLogger); err != nil {
		if err.Error() == UserNotReadyError {
			return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
		}
		return ctrl.Result{}, err
	}

	switch project.Status.Phase {
	default:
		return ctrl.Result{}, nil

	case v1alpha2.ProjectStatusPhaseUnknown:
		project.Status.Phase = v1alpha2.ProjectStatusPhaseCreating
		project.Status.Message = "project is about to be created"

	case v1alpha2.ProjectStatusPhaseCreating:
		if err := r.assertExistingProject(ctx, harborClient, project, patch); err != nil {
			return ctrl.Result{}, err
		}

		project.Status = v1alpha2.ProjectStatus{Phase: v1alpha2.ProjectStatusPhaseReady}

	case v1alpha2.ProjectStatusPhaseReady:
		controllerutil.AddFinalizer(project, internal.FinalizerName)
		if err := r.Client.Patch(ctx, project, patch); err != nil {
			return ctrl.Result{}, err
		}

		if err := r.assertExistingProject(ctx, harborClient, project, patch); err != nil {
			return ctrl.Result{}, err
		}

	case v1alpha2.ProjectStatusPhaseTerminating:
		// Delete the project via harbor API
		err := r.assertDeletedProject(ctx, reqLogger, harborClient, project)
		if err != nil {
			return ctrl.Result{}, err
		}

		return ctrl.Result{}, r.Client.Patch(ctx, project, patch)
	}

	return ctrl.Result{}, r.Client.Status().Patch(ctx, project, patch)
}

func (r *ProjectReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha2.Project{}).
		Watches(&v1alpha2.User{}, handler.EnqueueRequestForOwner(r.Scheme, mgr.GetRESTMapper(),
			&v1alpha2.Project{})).
		WithOptions(controller.Options{
			MaxConcurrentReconciles: 1,
		}).
		Complete(r)
}

func (r *ProjectReconciler) assertProxyCacheRequirementsReady(ctx context.Context, harborClient *h.RESTClient, project *v1alpha2.Project, logger logr.Logger) error {
	if project.Spec.ProxyCache == nil || project.Spec.ProxyCache.Registry == nil {
		return nil
	}

	regName := project.Spec.ProxyCache.Registry.Name
	l := logger.WithValues("registry", regName)
	registry := &v1alpha2.Registry{}
	if err := r.Client.Get(ctx, client.ObjectKey{Name: regName, Namespace: project.Namespace}, registry); err != nil {
		l.Error(err, "could not find proxy-cache registry")
		return err
	}

	_, err := harborClient.GetRegistryByName(ctx, registry.Spec.Name)
	if err != nil {
		l.Error(err, "waiting till registry is ready")
		return errors.New(RegistryNotReadyError)
	}

	return nil
}

func (r *ProjectReconciler) assertUserRequirementsReady(ctx context.Context, project *v1alpha2.Project, logger logr.Logger) error {
	if len(project.Spec.MemberRequests) == 0 {
		return nil
	}

	for _, req := range project.Spec.MemberRequests {
		l := logger.WithValues("user", req.User.Name)
		user := &v1alpha2.User{}
		if err := r.Client.Get(ctx, client.ObjectKey{Name: req.User.Name, Namespace: project.Namespace}, user); err != nil {
			l.Error(err, "user not found")
			return err
		}

		if user.Status.Phase != v1alpha2.UserStatusPhaseReady {
			l.Info("user not ready")
			return errors.New(UserNotReadyError)
		}
	}

	return nil
}

// assertDeletedProject deletes a Harbor project, first ensuring its existence.
func (r *ProjectReconciler) assertDeletedProject(ctx context.Context, log logr.Logger, harborClient *h.RESTClient,
	project *v1alpha2.Project) error {
	p, exists, err := internal.FetchHarborProjectIfExists(ctx, harborClient, project.Name)
	if err != nil {
		return err
	}

	if exists {
		if err := internal.DeleteHarborProject(ctx, harborClient, p); err != nil {
			return err
		}
	}

	log.Info("removing finalizer", project.Name, project.Namespace)
	controllerutil.RemoveFinalizer(project, internal.FinalizerName)

	return nil
}

// assertExistingProject
// Check Harbor projects and their components for their existence,
// create and delete either of those to match the specification.
func (r *ProjectReconciler) assertExistingProject(ctx context.Context, harborClient *h.RESTClient,
	project *v1alpha2.Project, patch client.Patch) error {
	heldRepo, err := harborClient.GetProject(ctx, project.Spec.Name)

	var registry v1alpha2.Registry
	var registryID *int64
	if project.Spec.ProxyCache != nil {
		if err := r.Client.Get(ctx, client.ObjectKey{
			Namespace: project.Namespace,
			Name:      project.Spec.ProxyCache.Registry.Name,
		}, &registry); err != nil {
			return err
		}
		registryID = &registry.Status.ID
	}

	if errors.Is(err, &clienterrors.ErrProjectNotFound{}) {
		storageLimit := int64(project.Spec.StorageLimit)
		err := harborClient.NewProject(ctx, &model.ProjectReq{
			CVEAllowlist: nil,
			Metadata:     nil,
			ProjectName:  project.Spec.Name,
			Public:       nil,
			RegistryID:   registryID,
			StorageLimit: &storageLimit,
		})

		return err
	} else if err != nil {
		return err
	}

	return r.ensureProject(ctx, heldRepo, harborClient, project, patch)
}

func (r *ProjectReconciler) projectMemberExists(members []*model.ProjectMemberEntity, requestedMember *v1alpha2.User) bool {
	for i := range members {
		if members[i].EntityName == requestedMember.Spec.Name {
			return true
		}
	}
	return false
}

// addProjectMemberStatus
func (r *ProjectReconciler) addProjectMemberStatus(ctx context.Context, project *v1alpha2.Project,
	request *v1alpha2.MemberRequest, patch client.Patch) error {
	for i := range project.Status.Members {
		if project.Status.Members[i].Name == request.User.Name {
			continue
		}
	}
	project.Status.Members = append(project.Status.Members, request.User)
	return r.Client.Status().Patch(ctx, project, patch)
}

// deleteProjectMemberStatus
func (r *ProjectReconciler) deleteProjectMemberStatus(ctx context.Context, project *v1alpha2.Project,
	request *corev1.LocalObjectReference, patch client.Patch) error {
	for i := range project.Status.Members {
		if project.Status.Members[i].Name == request.Name {
			project.Status.Members = append(project.Status.Members[:i], project.Status.Members[i+1:]...)
		}
	}

	return r.Client.Status().Patch(ctx, project, patch)
}

// projectMemberShouldExist checks whether the 'existing' reference to a user is contained in the 'desired' requests.
func (r *ProjectReconciler) projectMemberShouldExist(existing corev1.LocalObjectReference, desired []v1alpha2.MemberRequest) bool {
	for i := range desired {
		if existing.Name == desired[i].User.Name {
			return true
		}
	}

	return false
}

func (r *ProjectReconciler) reconcileProjectMembers(ctx context.Context, project *v1alpha2.Project,
	harborClient *h.RESTClient, harborProject *model.Project, patch client.Patch) error {
	for i := range project.Spec.MemberRequests {
		userCR, err := r.getUserCRFromRef(ctx, project.Spec.MemberRequests[i].User, project.Namespace)
		if err != nil {
			return fmt.Errorf("the user specified in project %s's list of member requests does not exist: %w", project.Name, err)
		}

		harborUser, err := harborClient.GetUserByName(ctx, userCR.Spec.Name)
		if err != nil {
			return err
		}

		memberUser := model.UserEntity{
			UserID:   harborUser.UserID,
			Username: harborUser.Username,
		}

		// Look up if the user exists as a project member. If not, add the user as project member.
		harborProjectMembers, err := harborClient.ListProjectMembers(ctx, harborProject.Name, memberUser.Username)
		if err != nil {
			return err
		}
		if !r.projectMemberExists(harborProjectMembers, userCR) {
			err = harborClient.AddProjectMember(ctx, harborProject.Name, &model.ProjectMember{
				MemberUser: &memberUser,
				RoleID:     int64(project.Spec.MemberRequests[i].Role.ID()),
			})
			if err != nil {
				return err
			}
			// Once the member user's existence is certain, append it to the project CR's status
			if err = r.addProjectMemberStatus(ctx, project, &project.Spec.MemberRequests[i], patch); err != nil {
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
				return fmt.Errorf("the user specified in project %s's list of existing members does not exist: %w", project.Name, err)
			}

			harborUser, err := harborClient.GetUserByName(ctx, userCR.Spec.Name)
			if err != nil {
				return err
			}

			memberUser := &model.UserEntity{
				UserID:   harborUser.UserID,
				Username: harborUser.Username,
			}

			// Look up if the user exists as a project member.
			harborProjectMembers, err := harborClient.ListProjectMembers(ctx, harborProject.Name, harborUser.Username)
			if err != nil {
				return err
			}
			if r.projectMemberExists(harborProjectMembers, userCR) {
				err = harborClient.DeleteProjectMember(ctx, harborProject.Name, &model.ProjectMember{
					MemberUser: memberUser,
					RoleID:     int64(project.Spec.MemberRequests[i].Role.ID()),
				})
				if err != nil {
					return err
				}
				if err = r.deleteProjectMemberStatus(ctx, project, &project.Status.Members[i], patch); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func (r *ProjectReconciler) getUserCRFromRef(ctx context.Context, userRef corev1.LocalObjectReference,
	namespace string) (*v1alpha2.User, error) {
	var user v1alpha2.User
	err := r.Client.Get(ctx, client.ObjectKey{Name: userRef.Name, Namespace: namespace}, &user)

	return &user, err
}

// ensureProject triggers reconciliation of project members
// and compares the state of the CR object with the project held by Harbor
func (r *ProjectReconciler) ensureProject(ctx context.Context, heldProject *model.Project,
	harborClient *h.RESTClient, project *v1alpha2.Project, patch client.Patch) error {
	newProject := &model.Project{}
	// Copy the spec of the project held by Harbor into a new object of the same type *harbor.Project
	err := copier.Copy(&newProject, &heldProject)
	if err != nil {
		return err
	}

	err = r.reconcileProjectMembers(ctx, project, harborClient, heldProject, patch)
	if err != nil {
		return err
	}

	project.Status.ID = heldProject.ProjectID
	if err := r.Client.Status().Patch(ctx, project, patch); err != nil {
		return err
	}

	newProject.Metadata = internal.GenerateProjectMetadata(&project.Spec.Metadata)

	// The "storageLimit" of a Harbor project is not contained in it's metadata,
	// so it has to be compared to the previously set storage limit on the project CR.
	// If set to a negative value (e.g. -1 for unlimited), it cannot be updated via
	// the UpdateProject method. We have to use UpdateStorageQuotaByProjectID instead.
	storageLimit := int64(project.Spec.StorageLimit)
	storagePtr := &storageLimit

	if storageLimit <= 0 {
		if err := harborClient.UpdateStorageQuotaByProjectID(ctx, int64(heldProject.ProjectID), storageLimit); err != nil {
			return err
		}
		storagePtr = nil
	}

	return harborClient.UpdateProject(ctx, newProject, storagePtr)
}
