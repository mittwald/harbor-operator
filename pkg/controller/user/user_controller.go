package user

import (
	"context"
	"reflect"
	"time"

	"github.com/go-logr/logr"
	h "github.com/mittwald/goharbor-client"
	registriesv1alpha1 "github.com/mittwald/harbor-operator/pkg/apis/registries/v1alpha1"
	"github.com/mittwald/harbor-operator/pkg/controller/internal"
	"github.com/mittwald/harbor-operator/pkg/internal/helper"
	corev1 "k8s.io/api/core/v1"
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

const (
	labelUserRegistry  = "users.registries.mittwald.de/registry"
	labelUserComponent = "users.registries.mittwald.de/component"
	labelUserRelease   = "instances.registries.mittwald.de/release"
)

const FinalizerName = "harbor-operator.registries.mittwald.de"

var log = logf.Log.WithName("controller_user")

// Add creates a new User Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileUser{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("user-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource User
	err = c.Watch(&source.Kind{Type: &registriesv1alpha1.User{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// Add a handler to watch for changes in the secondary resource, Secrets
	watchHandler := &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &registriesv1alpha1.User{},
	}

	// Watch for changes to secondary resources
	err = c.Watch(&source.Kind{Type: &corev1.Secret{}}, watchHandler)
	if err != nil {
		return err
	}

	return nil
}

// blank assignment to verify that ReconcileUser implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileUser{}

// ReconcileUser reconciles a User object
type ReconcileUser struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a User object and makes changes based on the state read
// and what is in the User.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileUser) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling User")

	now := metav1.Now()
	ctx := context.Background()

	// Fetch the User instance
	user := &registriesv1alpha1.User{}

	err := r.client.Get(ctx, request.NamespacedName, user)
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

	originalUser := user.DeepCopy()

	// Fetch the Instance
	harbor, err := internal.FetchReadyHarborInstance(ctx, user.Namespace, user.Spec.ParentInstance.Name, r.client)
	if err != nil {
		if _, ok := err.(internal.ErrInstanceNotFound); ok {
			user.Status = registriesv1alpha1.UserStatus{Name: string(registriesv1alpha1.UserStatusPhaseCreating)}
			// Requeue, the instance might not have been created yet
			return reconcile.Result{RequeueAfter: 30 * time.Second}, nil
		} else if _, ok := err.(internal.ErrInstanceNotReady); ok {
			return reconcile.Result{RequeueAfter: 120 * time.Second}, err
		} else {
			user.Status = registriesv1alpha1.UserStatus{LastTransition: &now}
		}
		res, err := r.patchUser(ctx, originalUser, user)
		if err != nil {
			return res, err
		}
	}

	// Build a client to connect to the harbor API
	harborClient, err := internal.BuildClient(ctx, r.client, harbor)
	if err != nil {
		return reconcile.Result{Requeue: true}, err
	}

	// Add finalizers to the CR object
	if user.DeletionTimestamp == nil {
		var hasFinalizer bool
		for i := range user.Finalizers {
			if user.Finalizers[i] == FinalizerName {
				hasFinalizer = true
			}
		}
		if !hasFinalizer {
			helper.PushFinalizer(user, FinalizerName)
			return r.patchUser(ctx, originalUser, user)
		}
	}

	// Handle user reconciliation
	switch user.Status.Phase {
	default:
		return reconcile.Result{}, nil

	case registriesv1alpha1.UserStatusPhaseUnknown:
		user.Status = registriesv1alpha1.UserStatus{Phase: registriesv1alpha1.UserStatusPhaseCreating}

	case registriesv1alpha1.UserStatusPhaseCreating:
		err = r.assertExistingUser(ctx, harborClient, user)
		if err != nil {
			return reconcile.Result{}, err
		}
		user.Status = registriesv1alpha1.UserStatus{Phase: registriesv1alpha1.UserStatusPhaseReady}

	case registriesv1alpha1.UserStatusPhaseReady:
		if user.ObjectMeta.DeletionTimestamp != nil {
			user.Status = registriesv1alpha1.UserStatus{Phase: registriesv1alpha1.UserStatusPhaseTerminating}
			return r.patchUser(ctx, originalUser, user)
		}
		err := r.assertExistingUser(ctx, harborClient, user)
		if err != nil {
			return reconcile.Result{}, err
		}

	case registriesv1alpha1.UserStatusPhaseTerminating:
		err := r.assertDeletedUser(reqLogger, harborClient, user)
		if err != nil {
			return reconcile.Result{}, err
		}
	}

	return r.patchUser(ctx, originalUser, user)
}

// patchUser compares the new CR status and finalizers with the pre-existing ones and updates them accordingly
func (r *ReconcileUser) patchUser(ctx context.Context, originalUser, user *registriesv1alpha1.User) (reconcile.Result, error) {
	// Update Status
	if !reflect.DeepEqual(originalUser.Status, user.Status) {
		originalUser.Status = user.Status
		if err := r.client.Status().Update(ctx, originalUser); err != nil {
			return reconcile.Result{}, err
		}
	}

	// Update Finalizers
	if !reflect.DeepEqual(originalUser.Finalizers, user.Finalizers) {
		originalUser.Finalizers = user.Finalizers
		if err := r.client.Update(ctx, originalUser); err != nil {
			return reconcile.Result{Requeue: true}, err
		}
	}

	return reconcile.Result{Requeue: true}, nil
}

// assertExistingUser ensures the specified user's existence
func (r *ReconcileUser) assertExistingUser(ctx context.Context, harborClient *h.Client, user *registriesv1alpha1.User) error {
	sec, err := r.getOrCreateSecretForUser(ctx, user)
	if err != nil {
		return err
	}
	pw, err := helper.GetKeyFromSecret(sec, "password")
	if err != nil {
		return err
	}

	heldUser, err := internal.GetUser(user, harborClient)
	if err != nil && err != internal.ErrUserNotFound {
		return err
	}
	if err == internal.ErrUserNotFound {
		return r.createUser(harborClient, user, pw)
	}

	return r.ensureUser(harborClient, heldUser, user, pw)
}

// createUser constructs a user request and triggers the Harbor API to create that user
func (r *ReconcileUser) createUser(harborClient *h.Client, user *registriesv1alpha1.User, newPassword string) error {
	usr := r.newUserRequest(user, newPassword)

	return harborClient.Users().AddUser(usr)
}

// labelsForUserSecret returns a list of labels for a user's secret
func (r *ReconcileUser) labelsForUserSecret(user *registriesv1alpha1.User, instanceName string) map[string]string {
	return map[string]string{
		labelUserRegistry:  instanceName,
		labelUserComponent: user.Spec.Name + "-secret",
		labelUserRelease:   "harbor",
	}
}

// newUserRequest builds a new user request from a user CR object
func (r *ReconcileUser) newUserRequest(user *registriesv1alpha1.User, pw string) h.UserRequest {
	userReq := h.UserRequest{
		Username:     user.Spec.Name,
		RealName:     user.Spec.RealName,
		Email:        user.Spec.Email,
		Password:     pw,
		HasAdminRole: user.Spec.AdminRole,
	}

	return userReq
}

// ensureUser updates a users profile, if changed - Afterwards, it updates the password if changed
func (r *ReconcileUser) ensureUser(harborClient *h.Client, heldUser h.User, desiredUser *registriesv1alpha1.User, password string) error {
	newUsr := h.UserRequest{
		UserID: int64(heldUser.UserID),
	}

	newUsr.Username = desiredUser.Spec.Name
	newUsr.Email = desiredUser.Spec.Email
	newUsr.RealName = desiredUser.Spec.RealName
	newUsr.HasAdminRole = desiredUser.Spec.AdminRole

	if !isUserRequestEqual(heldUser, newUsr) {
		// Update the user if spec has changed
		err := harborClient.Users().UpdateUserProfile(newUsr)
		if err != nil {
			return err
		}
	}

	return harborClient.Users().UpdateUserPasswordAsAdmin(newUsr.UserID, password)
}

// isUserRequestEqual compares the individual values of an existing user with a user request
func isUserRequestEqual(existing h.User, new h.UserRequest) bool {
	if new.Username != existing.Username {
		return false
	} else if new.Email != existing.Email {
		return false
	} else if new.RealName != existing.Realname {
		return false
	} else if new.Role != existing.Role {
		return false
	} else if new.HasAdminRole != existing.SysAdminFlag {
		return false
	}
	return true
}

// assertDeletedUser deletes a user, first ensuring its existence
func (r *ReconcileUser) assertDeletedUser(log logr.Logger, harborClient *h.Client, user *registriesv1alpha1.User) error {
	harborUser, err := internal.GetUser(user, harborClient)
	if err != nil {
		return err
	}

	uReq := h.UserRequest{UserID: int64(harborUser.UserID)}
	err = harborClient.Users().DeleteUser(uReq)
	if err != nil {
		return err
	}

	log.Info("pulling finalizers")
	helper.PullFinalizer(user, FinalizerName)

	return nil
}
