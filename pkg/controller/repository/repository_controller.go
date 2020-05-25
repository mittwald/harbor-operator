package repository

import (
	"context"
	"reflect"
	"time"

	v1 "k8s.io/api/core/v1"

	"github.com/go-logr/logr"
	"github.com/jinzhu/copier"
	h "github.com/mittwald/goharbor-client"
	registriesv1alpha1 "github.com/mittwald/harbor-operator/pkg/apis/registries/v1alpha1"
	"github.com/mittwald/harbor-operator/pkg/controller/internal"
	"github.com/mittwald/harbor-operator/pkg/internal/helper"
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

var log = logf.Log.WithName("controller_repository")

// Add creates a new Repository Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileRepository{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
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

// blank assignment to verify that ReconcileRepository implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileRepository{}

// ReconcileRepository reconciles a Repository object
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
func (r *ReconcileRepository) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling Repository")

	now := metav1.Now()
	ctx := context.Background()

	// Fetch the Repository instance
	repository := &registriesv1alpha1.Repository{}

	err := r.client.Get(ctx, request.NamespacedName, repository)
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

	originalRepository := repository.DeepCopy()

	if repository.ObjectMeta.DeletionTimestamp != nil {
		repository.Status = registriesv1alpha1.RepositoryStatus{Phase: registriesv1alpha1.RepositoryStatusPhaseTerminating}
		return r.patchRepository(ctx, originalRepository, repository)
	}

	// Fetch the Instance
	harbor, err := internal.FetchReadyHarborInstance(ctx, repository.Namespace, repository.Spec.ParentInstance.Name, r.client)
	if err != nil {
		if _, ok := err.(internal.ErrInstanceNotFound); ok {
			repository.Status = registriesv1alpha1.RepositoryStatus{Name: string(registriesv1alpha1.RepositoryStatusPhaseCreating)}
			// Requeue, the instance might not have been created yet
			return reconcile.Result{RequeueAfter: 30 * time.Second}, nil
		} else if _, ok := err.(internal.ErrInstanceNotReady); ok {
			return reconcile.Result{RequeueAfter: 120 * time.Second}, err
		} else {
			repository.Status = registriesv1alpha1.RepositoryStatus{LastTransition: &now}
		}
		res, err := r.patchRepository(ctx, originalRepository, repository)
		if err != nil {
			return res, err
		}
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

	case registriesv1alpha1.RepositoryStatusPhaseCreating:
		helper.PushFinalizer(repository, FinalizerName)

		// Install the repository
		err = r.assertExistingRepository(harborClient, repository)
		if err != nil {
			return reconcile.Result{}, err
		}
		repository.Status = registriesv1alpha1.RepositoryStatus{Phase: registriesv1alpha1.RepositoryStatusPhaseReady}

	case registriesv1alpha1.RepositoryStatusPhaseReady:
		err := r.assertExistingRepository(harborClient, repository)
		if err != nil {
			return reconcile.Result{}, err
		}

	case registriesv1alpha1.RepositoryStatusPhaseTerminating:
		// Delete the repository via harbor API
		err := r.assertDeletedRepository(reqLogger, harborClient, repository)
		if err != nil {
			return reconcile.Result{}, err
		}
	}

	return r.patchRepository(ctx, originalRepository, repository)
}

// patchRepository compares the new CR status and finalizers with the pre-existing ones and updates them accordingly
func (r *ReconcileRepository) patchRepository(ctx context.Context, originalRepository, repository *registriesv1alpha1.Repository) (reconcile.Result, error) {
	// Update Status
	if !reflect.DeepEqual(originalRepository.Status, repository.Status) {
		originalRepository.Status = repository.Status
		if err := r.client.Status().Update(ctx, originalRepository); err != nil {
			return reconcile.Result{}, err
		}
	}

	// Update Finalizers
	if !reflect.DeepEqual(originalRepository.Finalizers, repository.Finalizers) {
		originalRepository.Finalizers = repository.Finalizers
	}

	if err := r.client.Update(ctx, originalRepository); err != nil {
		return reconcile.Result{Requeue: true}, err
	}

	return reconcile.Result{Requeue: true}, nil
}

// assertDeletedRepository deletes a Harbor project, first ensuring its existence
func (r *ReconcileRepository) assertDeletedRepository(log logr.Logger, harborClient *h.Client, repository *registriesv1alpha1.Repository) error {
	opt := h.ListProjectsOptions{Name: repository.Spec.Name}
	repos, err := harborClient.Projects().ListProjects(opt)
	if err != nil {
		return err
	}

	if len(repos) > 0 {
		err := harborClient.Projects().DeleteProject(repos[0].ProjectID)
		if err != nil {
			return err
		}
	}

	log.Info("pulling finalizers", repository.Name, repository.Namespace)
	helper.PullFinalizer(repository, FinalizerName)

	return nil
}

// assertExistingRepository
// Check harbor projects and their components for their existence,
// create and delete either of those to match the specification
func (r *ReconcileRepository) assertExistingRepository(harborClient *h.Client, repository *registriesv1alpha1.Repository) error {
	opt := h.ListProjectsOptions{Name: repository.Spec.Name}
	heldRepos, err := harborClient.Projects().ListProjects(opt)
	if err != nil {
		return err
	}

	if len(heldRepos) == 0 {
		specRepositoryMeta := repository.Spec.Metadata
		// Generate new repository metadata from spec
		newRepositoryMeta, err := r.generateRepositoryMetadata(&specRepositoryMeta)
		if err != nil {
			return err
		}

		pReq := h.ProjectRequest{
			Name:     repository.Spec.Name,
			Metadata: newRepositoryMeta,
		}
		return harborClient.Projects().CreateProject(pReq)
	}
	return r.ensureRepository(&heldRepos[0], harborClient, repository)
}

// generateRepositoryMetadata constructs the repository metadata for a Harbor project
func (r *ReconcileRepository) generateRepositoryMetadata(projectMeta *registriesv1alpha1.RepositoryMetadata) (map[string]string, error) {
	pm := map[string]string{
		"auto_scan":               helper.BoolToString(projectMeta.AutoScan),
		"enable_content_trust":    helper.BoolToString(projectMeta.EnableContentTrust),
		"prevent_vul":             helper.BoolToString(projectMeta.PreventVul),
		"public":                  helper.BoolToString(projectMeta.Public),
		"reuse_sys_sve_whitelist": helper.BoolToString(projectMeta.ReuseSysSVEWhitelist),
		"severity":                projectMeta.Severity,
	}
	return pm, nil
}

// reconcileProjectMembers reconciles the user-defined project members for a repository based on the actual Harbor user
func (r *ReconcileRepository) reconcileProjectMembers(repository *registriesv1alpha1.Repository, harborClient *h.Client, heldRepository *h.Project) error {
	members, err := harborClient.Projects().GetProjectMembers(heldRepository.ProjectID)
	if err != nil {
		return err
	}

	for _, memberRequestUser := range repository.Spec.MemberRequests {
		user, err := r.getUserFromRef(memberRequestUser.User, repository.Namespace)
		if err != nil {
			return err
		}

		harborUser, err := internal.GetUser(user, harborClient)
		if err != nil {
			return err
		}

		roleID := memberRequestUser.Role.ID()

		projectMember := h.MemberReq{
			Role: roleID,
			MemberUser: h.User{
				Username: harborUser.Username,
				UserID:   harborUser.UserID,
			},
		}

		member := getMemberUserFromList(members, user)

		if member == nil {
			err = harborClient.Projects().AddProjectMember(heldRepository.ProjectID, projectMember)
			if err != nil {
				return err
			}
			break
		}

		// Update the project member
		role, err := harborClient.Projects().GetProjectMember(heldRepository.ProjectID, int64(member.ID))
		if err != nil {
			return err
		}

		if roleID == role.RoleID {
			break
		}

		err = harborClient.Projects().UpdateProjectMember(
			heldRepository.ProjectID,
			int64(member.ID),
			h.RoleRequest{Role: roleID})
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *ReconcileRepository) getUserFromRef(userRef v1.LocalObjectReference, namespace string) (*registriesv1alpha1.User, error) {
	var user registriesv1alpha1.User
	err := r.client.Get(context.Background(), client.ObjectKey{Name: userRef.Name, Namespace: namespace}, &user)
	return &user, err
}

// getMemberUserFromList returns a project member from a list of members, filtered by the username
func getMemberUserFromList(members []h.Member, user *registriesv1alpha1.User) *h.Member {
	for i := range members {
		if members[i].Entityname == user.Spec.Name {
			return &members[i]
		}
	}
	return nil
}

// ensureRepository triggers reconciliation of project members and compares the state of the CR object with the project held by Harbor
func (r *ReconcileRepository) ensureRepository(heldRepository *h.Project, harborClient *h.Client, repository *registriesv1alpha1.Repository) error {
	updatedRepository := &h.Project{}
	// Copy the held projects spec into a new object of the same type* harbor.Repository
	err := copier.Copy(&updatedRepository, &heldRepository)
	if err != nil {
		return err
	}

	err = r.reconcileProjectMembers(repository, harborClient, heldRepository)
	if err != nil {
		return err
	}

	updatedRepository.Metadata, err = r.generateRepositoryMetadata(&repository.Spec.Metadata)
	if err != nil {
		return err
	}

	if updatedRepository != heldRepository {
		return harborClient.Projects().UpdateProject(heldRepository.ProjectID, *updatedRepository)
	}
	return nil
}
