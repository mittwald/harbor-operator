package user

import (
	"context"
	"fmt"
	"reflect"
	"time"

	modelv1 "github.com/mittwald/goharbor-client/model/v1_10_0"

	controllerruntime "sigs.k8s.io/controller-runtime"

	"github.com/go-logr/logr"
	h "github.com/mittwald/goharbor-client"
	harborClientUser "github.com/mittwald/goharbor-client/user"
	registriesv1alpha1 "github.com/mittwald/harbor-operator/pkg/apis/registries/v1alpha1"
	"github.com/mittwald/harbor-operator/pkg/controller/internal"
	"github.com/mittwald/harbor-operator/pkg/internal/helper"
	corev1 "k8s.io/api/core/v1"
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

// newReconciler returns a new reconcile.Reconciler.
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileUser{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler.
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

// blank assignment to verify that ReconcileUser implements reconcile.Reconciler.
var _ reconcile.Reconciler = &ReconcileUser{}

// ReconcileUser reconciles a User object.
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
func (r *ReconcileUser) Reconcile(request reconcile.Request) (result reconcile.Result, err error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling User")

	now := metav1.Now()
	ctx := context.Background()

	// Fetch the User instance
	user := &registriesv1alpha1.User{}

	err = r.client.Get(ctx, request.NamespacedName, user)
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

	originalUser := user.DeepCopy()

	if user.ObjectMeta.DeletionTimestamp != nil && user.Status.Phase != registriesv1alpha1.UserStatusPhaseTerminating {
		user.Status = registriesv1alpha1.UserStatus{Phase: registriesv1alpha1.UserStatusPhaseTerminating}
		result = reconcile.Result{Requeue: true}

		return r.updateUserCR(ctx, nil, originalUser, user, result)
	}

	// Fetch the Instance
	harbor, err := internal.FetchReadyHarborInstance(ctx, user.Namespace, user.Spec.ParentInstance.Name, r.client)
	if err != nil {
		if _, ok := err.(internal.ErrInstanceNotFound); ok {
			helper.PullFinalizer(user, FinalizerName)

			result = reconcile.Result{}
		} else if _, ok := err.(internal.ErrInstanceNotReady); ok {
			return reconcile.Result{RequeueAfter: 30 * time.Second}, err
		} else {
			user.Status = registriesv1alpha1.UserStatus{LastTransition: &now}
			result = reconcile.Result{RequeueAfter: 120 * time.Second}
		}

		return r.updateUserCR(ctx, nil, originalUser, user, result)
	}

	// Build a client to connect to the harbor API
	harborClient, err := internal.BuildClient(ctx, r.client, harbor)
	if err != nil {
		return reconcile.Result{Requeue: true}, err
	}

	// Handle user reconciliation
	switch user.Status.Phase {
	default:
		return reconcile.Result{}, nil

	case registriesv1alpha1.UserStatusPhaseUnknown:
		user.Status.Phase = registriesv1alpha1.UserStatusPhaseCreating
		if err := r.client.Status().Update(ctx, user); err != nil {
			return reconcile.Result{}, err
		}
		return reconcile.Result{Requeue: true}, nil

	case registriesv1alpha1.UserStatusPhaseCreating:
		if err := r.assertExistingUser(ctx, harborClient, user); err != nil {
			return reconcile.Result{}, err
		}

		helper.PushFinalizer(user, FinalizerName)

		user.Status.Phase = registriesv1alpha1.UserStatusPhaseReady
		result = reconcile.Result{Requeue: true}

	case registriesv1alpha1.UserStatusPhaseReady:
		err := r.assertExistingUser(ctx, harborClient, user)
		if err != nil {
			return reconcile.Result{}, err
		}

		result = reconcile.Result{}

	case registriesv1alpha1.UserStatusPhaseTerminating:
		err := r.assertDeletedUser(ctx, reqLogger, harborClient, user)
		if err != nil {
			return reconcile.Result{}, err
		}

		result = reconcile.Result{}
	}

	return r.updateUserCR(ctx, harbor, originalUser, user, result)
}

// updateUserCR compares the new CR status and finalizers with the pre-existing ones and updates them accordingly.
func (r *ReconcileUser) updateUserCR(ctx context.Context, parentInstance *registriesv1alpha1.Instance, originalUser,
	user *registriesv1alpha1.User, result reconcile.Result) (reconcile.Result, error) {
	if originalUser == nil || user == nil {
		return reconcile.Result{},
			fmt.Errorf("cannot update user because the original user is nil")
	}

	// Update Status
	if !reflect.DeepEqual(originalUser.Status, user.Status) {
		if err := r.client.Status().Update(ctx, user); err != nil {
			return reconcile.Result{}, err
		}
	}

	// set owner
	if len(originalUser.OwnerReferences) == 0 && parentInstance != nil {
		if err := controllerruntime.SetControllerReference(parentInstance, originalUser, r.scheme); err != nil {
			return reconcile.Result{}, err
		}
	}

	if !reflect.DeepEqual(originalUser, user) {
		if err := r.client.Update(ctx, user); err != nil {
			return reconcile.Result{}, err
		}
	}

	return result, nil
}

// assertExistingUser ensures the specified user's existence.
func (r *ReconcileUser) assertExistingUser(ctx context.Context, harborClient *h.RESTClient,
	usr *registriesv1alpha1.User) error {
	sec, err := r.getOrCreateSecretForUser(ctx, usr)
	if err != nil {
		return err
	}

	pw, err := helper.GetValueFromSecret(sec, "password")
	if err != nil {
		return err
	}

	pwHash, err := helper.GenerateHashFromInterfaces([]interface{}{pw})
	if err != nil {
		return err
	}

	heldUser, err := harborClient.GetUser(ctx, usr.Spec.Name)
	if err != nil {
		switch err.Error() {
		case harborClientUser.ErrUserNotFoundMsg:
			usr.Status.PasswordHash = pwHash.Short()
			return r.createUser(ctx, harborClient, usr, pw)
		default:
			return err
		}
	}

	if usr.Status.PasswordHash != pwHash.Short() {
		usr.Status.PasswordHash = pwHash.Short()

		if err = harborClient.UpdateUserPassword(ctx, heldUser.UserID,
			&modelv1.Password{
				NewPassword: pw,
			}); err != nil {
			return err
		}
	}

	return r.ensureUser(ctx, harborClient, heldUser, usr)
}

// createUser constructs a user request and triggers the Harbor API to create that user.
func (r *ReconcileUser) createUser(ctx context.Context, harborClient *h.RESTClient, user *registriesv1alpha1.User,
	newPassword string) error {
	_, err := harborClient.NewUser(ctx,
		user.Spec.Name,
		user.Spec.Email,
		user.Spec.RealName,
		newPassword,
		user.Spec.Comments)
	if err != nil {
		return err
	}

	return nil
}

// labelsForUserSecret returns a list of labels for a user's secret.
func (r *ReconcileUser) labelsForUserSecret(user *registriesv1alpha1.User, instanceName string) map[string]string {
	return map[string]string{
		labelUserRegistry:  instanceName,
		labelUserComponent: user.Spec.Name + "-secret",
		labelUserRelease:   "harbor",
	}
}

// ensureUser updates a users profile, if changed - Afterwards, it updates the password if changed.
func (r *ReconcileUser) ensureUser(ctx context.Context, harborClient *h.RESTClient,
	heldUser *modelv1.User, desiredUser *registriesv1alpha1.User) error {
	newUsr := &modelv1.User{
		UserID: heldUser.UserID,
	}

	newUsr.Username = desiredUser.Spec.Name
	newUsr.Email = desiredUser.Spec.Email
	newUsr.Realname = desiredUser.Spec.RealName
	newUsr.HasAdminRole = desiredUser.Spec.AdminRole

	if isUserRequestEqual(heldUser, newUsr) {
		return nil
	}

	// Update the user if spec has changed
	return harborClient.UpdateUser(ctx, newUsr)
}

// isUserRequestEqual compares the individual values of an existing user with a user request
func isUserRequestEqual(existing, new *modelv1.User) bool {
	if new.Username != existing.Username {
		return false
	} else if new.Email != existing.Email {
		return false
	} else if new.Realname != existing.Realname {
		return false
	} else if new.RoleID != existing.RoleID {
		return false
	} else if new.HasAdminRole != existing.HasAdminRole {
		return false
	}

	return true
}

// assertDeletedUser deletes a user, first ensuring its existence
func (r *ReconcileUser) assertDeletedUser(ctx context.Context, log logr.Logger, harborClient *h.RESTClient,
	user *registriesv1alpha1.User) error {
	harborUser, err := harborClient.GetUser(ctx, user.Spec.Name)
	if err != nil {
		return err
	}

	uReq := &modelv1.User{
		Username: harborUser.Username,
		UserID:   harborUser.UserID,
	}

	if err := harborClient.DeleteUser(ctx, uReq); err != nil {
		return err
	}

	log.Info("pulling finalizers")
	helper.PullFinalizer(user, FinalizerName)

	return nil
}
