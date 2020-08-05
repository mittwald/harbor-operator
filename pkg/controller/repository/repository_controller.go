package repository

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"time"

	modelv1 "github.com/mittwald/goharbor-client/model/v1_10_0"
	"github.com/mittwald/goharbor-client/project"

	controllerruntime "sigs.k8s.io/controller-runtime"

	v1 "k8s.io/api/core/v1"

	"github.com/go-logr/logr"
	"github.com/jinzhu/copier"
	h "github.com/mittwald/goharbor-client"
	registriesv1alpha1 "github.com/mittwald/harbor-operator/pkg/apis/registries/v1alpha1"
	"github.com/mittwald/harbor-operator/pkg/controller/internal"
	"github.com/mittwald/harbor-operator/pkg/internal/helper"
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

var log = logf.Log.WithName("controller_repository")

// Add creates a new Repository Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler.
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileRepository{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler.
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("repository-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource Repository
	err = c.Watch(&source.Kind{Type: &registriesv1alpha1.Repository{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// Add a handler to watch for changes in the secondary resource, User
	watchHandler := &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &registriesv1alpha1.Repository{},
	}

	// Watch for changes to secondary resources
	err = c.Watch(&source.Kind{Type: &registriesv1alpha1.User{}}, watchHandler)
	if err != nil {
		return err
	}

	return nil
}

// blank assignment to verify that ReconcileRepository implements reconcile.Reconciler.
var _ reconcile.Reconciler = &ReconcileRepository{}

// ReconcileRepository reconciles a Repository object.
type ReconcileRepository struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a Repository object and makes changes based on the state read
// and what is in the Repository.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileRepository) Reconcile(request reconcile.Request) (result reconcile.Result, err error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling Repository")

	now := metav1.Now()
	ctx := context.Background()

	// Fetch the Repository instance
	repository := &registriesv1alpha1.Repository{}

	err = r.client.Get(ctx, request.NamespacedName, repository)
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

	originalRepository := repository.DeepCopy()

	if repository.ObjectMeta.DeletionTimestamp != nil &&
		repository.Status.Phase != registriesv1alpha1.RepositoryStatusPhaseTerminating {
		repository.Status = registriesv1alpha1.RepositoryStatus{Phase: registriesv1alpha1.RepositoryStatusPhaseTerminating}
		result = reconcile.Result{Requeue: true}

		return r.updateRepositoryCR(ctx, nil, originalRepository, repository)
	}

	// Fetch the Instance
	harbor, err := internal.FetchReadyHarborInstance(ctx, repository.Namespace,
		repository.Spec.ParentInstance.Name, r.client)
	if err != nil {
		if _, ok := err.(internal.ErrInstanceNotFound); ok {
			helper.PullFinalizer(repository, FinalizerName)

			result = reconcile.Result{}
		} else if _, ok := err.(internal.ErrInstanceNotReady); ok {
			return reconcile.Result{RequeueAfter: 30 * time.Second}, err
		} else {
			repository.Status = registriesv1alpha1.RepositoryStatus{LastTransition: &now}
			result = reconcile.Result{RequeueAfter: 120 * time.Second}
		}

		return r.updateRepositoryCR(ctx, nil, originalRepository, repository)
	}

	// Build a client to connect to the harbor API
	harborClient, err := internal.BuildClient(ctx, r.client, harbor)
	if err != nil {
		return reconcile.Result{Requeue: true}, err
	}

	switch repository.Status.Phase {
	default:
		return reconcile.Result{}, nil

	case registriesv1alpha1.RepositoryStatusPhaseUnknown:
		repository.Status = registriesv1alpha1.RepositoryStatus{Phase: registriesv1alpha1.RepositoryStatusPhaseCreating}
		result = reconcile.Result{Requeue: true}

	case registriesv1alpha1.RepositoryStatusPhaseCreating:
		helper.PushFinalizer(repository, FinalizerName)

		// Install the repository
		err = r.assertExistingRepository(ctx, harborClient, repository)
		if err != nil {
			return reconcile.Result{}, err
		}

		repository.Status = registriesv1alpha1.RepositoryStatus{Phase: registriesv1alpha1.RepositoryStatusPhaseReady}
		result = reconcile.Result{Requeue: true}

	case registriesv1alpha1.RepositoryStatusPhaseReady:
		err := r.assertExistingRepository(ctx, harborClient, repository)
		if err != nil {
			return reconcile.Result{}, err
		}

		result = reconcile.Result{}

	case registriesv1alpha1.RepositoryStatusPhaseTerminating:
		// Delete the repository via harbor API
		err := r.assertDeletedRepository(ctx, reqLogger, harborClient, repository)
		if err != nil {
			return reconcile.Result{}, err
		}

		result = reconcile.Result{}
	}

	return r.updateRepositoryCR(ctx, harbor, originalRepository, repository)
}

// updateRepositoryCR compares the new CR status and finalizers with the pre-existing ones and updates them accordingly.
func (r *ReconcileRepository) updateRepositoryCR(ctx context.Context, parentInstance *registriesv1alpha1.Instance, originalRepository, repository *registriesv1alpha1.Repository) (reconcile.Result, error) {
	if originalRepository == nil || repository == nil {
		return reconcile.Result{},
			fmt.Errorf("cannot update repository '%s' because the original repository is nil", repository.Spec.Name)
	}

	// Update Status
	if !reflect.DeepEqual(originalRepository.Status, repository.Status) {
		originalRepository.Status = repository.Status
		if err := r.client.Status().Update(ctx, originalRepository); err != nil {
			return reconcile.Result{}, err
		}
	}

	// Set owner reference
	if (originalRepository.OwnerReferences == nil ||
		len(originalRepository.OwnerReferences) == 0) &&
		parentInstance != nil {
		err := controllerruntime.SetControllerReference(parentInstance, originalRepository, r.scheme)
		if err != nil {
			return reconcile.Result{}, err
		}
	}

	// Update Finalizer
	if !reflect.DeepEqual(originalRepository.Finalizers, repository.Finalizers) {
		originalRepository.SetFinalizers(repository.Finalizers)
	}

	if err := r.client.Update(ctx, originalRepository); err != nil {
		return reconcile.Result{Requeue: true}, err
	}

	return reconcile.Result{}, nil
}

// assertDeletedRepository deletes a Harbor project, first ensuring its existence.
func (r *ReconcileRepository) assertDeletedRepository(ctx context.Context, log logr.Logger, harborClient *h.RESTClient,
	repository *registriesv1alpha1.Repository) error {
	_, err := harborClient.GetProject(ctx, repository.Name)
	if err != nil {
		return err
	}

	log.Info("pulling finalizers", repository.Name, repository.Namespace)
	helper.PullFinalizer(repository, FinalizerName)

	return nil
}

// assertExistingRepository
// Check Harbor projects and their components for their existence,
// create and delete either of those to match the specification.
func (r *ReconcileRepository) assertExistingRepository(ctx context.Context, harborClient *h.RESTClient,
	repository *registriesv1alpha1.Repository) error {
	heldRepo, err := harborClient.GetProject(ctx, repository.Name)

	if errors.Is(err, &project.ErrProjectNotFound{}) {
		_, err := harborClient.NewProject(ctx, repository.Spec.Name, repository.Spec.CountLimit,
			repository.Spec.StorageLimit)
		return err
	} else if err != nil {
		return err
	}

	return r.ensureRepository(ctx, heldRepo, harborClient, repository)
}

// generateRepositoryMetadata constructs the repository metadata for a Harbor project
func (r *ReconcileRepository) generateRepositoryMetadata(
	repositoryMeta *registriesv1alpha1.RepositoryMetadata) *modelv1.ProjectMetadata {
	pm := modelv1.ProjectMetadata{
		AutoScan:             helper.BoolToString(repositoryMeta.AutoScan),
		EnableContentTrust:   helper.BoolToString(repositoryMeta.EnableContentTrust),
		PreventVul:           helper.BoolToString(repositoryMeta.PreventVul),
		Public:               helper.BoolToString(repositoryMeta.Public),
		ReuseSysCveWhitelist: helper.BoolToString(repositoryMeta.ReuseSysSVEWhitelist),
		Severity:             repositoryMeta.Severity,
	}

	return &pm
}

// reconcileRepositoryMembers reconciles the user-defined project members
// for a repository based on an existing Harbor user
func (r *ReconcileRepository) reconcileRepositoryMembers(ctx context.Context, repository *registriesv1alpha1.Repository,
	harborClient *h.RESTClient, heldProject *modelv1.Project) error {
	// List Harbor project members
	members, err := harborClient.ListProjectMembers(ctx, heldProject)
	if err != nil {
		return err
	}

	// Range over the defined member users of a project
	for _, memberRequestUser := range repository.Spec.MemberRequests {
		// Fetch the user resource
		user, err := r.getUserFromRef(memberRequestUser.User, repository.Namespace)
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

func (r *ReconcileRepository) getUserFromRef(userRef v1.LocalObjectReference,
	namespace string) (*registriesv1alpha1.User, error) {
	var user registriesv1alpha1.User
	err := r.client.Get(context.Background(), client.ObjectKey{Name: userRef.Name, Namespace: namespace}, &user)

	return &user, err
}

// getMemberUserFromList returns a project member from a list of members, filtered by the username
func getMemberUserFromList(members []*modelv1.ProjectMemberEntity,
	user *registriesv1alpha1.User) *modelv1.ProjectMemberEntity {
	for i := range members {
		if members[i].EntityName == user.Spec.Name {
			return members[i]
		}
	}

	return nil
}

// ensureRepository triggers reconciliation of project members
// and compares the state of the CR object with the project held by Harbor
func (r *ReconcileRepository) ensureRepository(ctx context.Context, heldRepository *modelv1.Project,
	harborClient *h.RESTClient, repository *registriesv1alpha1.Repository) error {
	updatedProject := &modelv1.Project{}
	// Copy the spec of the project held by Harbor into a new object of the same type *harbor.Repository
	err := copier.Copy(&updatedProject, &heldRepository)
	if err != nil {
		return err
	}

	err = r.reconcileRepositoryMembers(ctx, repository, harborClient, heldRepository)
	if err != nil {
		return err
	}

	updatedProject.Metadata = r.generateRepositoryMetadata(&repository.Spec.Metadata)

	if updatedProject != heldRepository {
		return harborClient.UpdateProject(ctx, heldRepository, repository.Spec.StorageLimit, repository.Spec.CountLimit)
	}

	return nil
}
