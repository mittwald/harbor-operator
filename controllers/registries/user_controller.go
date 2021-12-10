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
	controllererrors "github.com/mittwald/harbor-operator/controllers/registries/errors"
	"time"

	"github.com/mittwald/goharbor-client/v5/apiv2/model"
	clienterrors "github.com/mittwald/goharbor-client/v5/apiv2/pkg/errors"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/go-logr/logr"
	h "github.com/mittwald/goharbor-client/v5/apiv2"
	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/mittwald/harbor-operator/apis/registries/v1alpha2"

	"github.com/mittwald/harbor-operator/controllers/registries/helper"
	"github.com/mittwald/harbor-operator/controllers/registries/internal"
)

// UserReconciler reconciles a User object
type UserReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

const (
	labelUserRegistry  = "users.registries.mittwald.de/registry"
	labelUserComponent = "users.registries.mittwald.de/component"
	labelUserRelease   = "instances.registries.mittwald.de/release"
)

func (r *UserReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha2.User{}).
		Owns(&corev1.Secret{}).
		WithOptions(controller.Options{
			MaxConcurrentReconciles: 1,
		}).
		Complete(r)
}

// +kubebuilder:rbac:groups=registries.mittwald.de,resources=users,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=registries.mittwald.de,resources=users/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="",resources=secrets;configmaps,verbs=get;list;watch;create;update;delete;patch
// +kubebuilder:rbac:groups="",resources=pods,verbs=get;list;delete;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *UserReconciler) Reconcile(ctx context.Context, req ctrl.Request) (res ctrl.Result, err error) {
	reqLogger := r.Log.WithValues("user", req.NamespacedName)
	reqLogger.Info("Reconciling User")

	// Fetch the User instance
	user := &v1alpha2.User{}

	err = r.Client.Get(ctx, req.NamespacedName, user)
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

	patch := client.MergeFrom(user.DeepCopy())

	if user.ObjectMeta.DeletionTimestamp != nil && user.Status.Phase != v1alpha2.UserStatusPhaseTerminating {
		user.Status = v1alpha2.UserStatus{Phase: v1alpha2.UserStatusPhaseTerminating}

		return ctrl.Result{}, r.Client.Status().Patch(ctx, user, patch)
	}

	// Fetch the goharbor instance if it exists and is properly set up.
	// If the above does not apply, pull the finalizer from the user object.
	harbor, err := internal.GetOperationalHarborInstance(ctx, client.ObjectKey{
		Namespace: user.Namespace,
		Name:      user.Spec.ParentInstance.Name,
	}, r.Client)
	if err != nil {
		switch err.Error() {
		case controllererrors.ErrInstanceNotInstalledMsg:
			reqLogger.Info("waiting till harbor instance is installed")
			return ctrl.Result{RequeueAfter: 30*time.Second}, nil
		case controllererrors.ErrInstanceNotFoundMsg:
			controllerutil.RemoveFinalizer(user, internal.FinalizerName)
			fallthrough
		default:
			return ctrl.Result{}, err
		}
	}

	// Set OwnerReference to the parent harbor instance
	err = ctrl.SetControllerReference(harbor, user, r.Scheme)
	if err != nil {
		return ctrl.Result{}, err
	}
	if err := r.Client.Status().Patch(ctx, user, patch); err != nil {
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
		return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
	}

	// Handle user reconciliation
	switch user.Status.Phase {
	default:
		return ctrl.Result{}, nil

	case v1alpha2.UserStatusPhaseUnknown:
		user.Status.Phase = v1alpha2.UserStatusPhaseCreating
		user.Status.Message = "user is about to be created"
		user.Status.PasswordHash = ""

		return ctrl.Result{}, r.Client.Status().Patch(ctx, user, patch)

	case v1alpha2.UserStatusPhaseCreating:
		// Prior to handling operations on the user instance, the harbor 'admin' user has to exist.
		adminUserExists, err := helper.AdminUserExists(ctx, harborClient)
		if err != nil {
			return ctrl.Result{}, err
		}
		if !adminUserExists {
			r.Log.Info("harbor admin user does not yet exist")
			return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
		}
		if err := r.assertExistingUser(ctx, harborClient, user); err != nil {
			return ctrl.Result{}, err
		}

		controllerutil.AddFinalizer(user, internal.FinalizerName)

		user.Status.Phase = v1alpha2.UserStatusPhaseReady

	case v1alpha2.UserStatusPhaseReady:
		controllerutil.AddFinalizer(user, internal.FinalizerName)
		if err := r.Client.Patch(ctx, user, patch); err != nil {
			return ctrl.Result{}, err
		}

		if err := r.assertExistingUser(ctx, harborClient, user); err != nil {
			return ctrl.Result{}, err
		}

	case v1alpha2.UserStatusPhaseTerminating:
		err := r.assertDeletedUser(ctx, reqLogger, harborClient, user, patch)
		if err != nil {
			return ctrl.Result{}, err
		}

		return ctrl.Result{}, nil
	}

	return ctrl.Result{}, r.Client.Status().Patch(ctx, user, patch)
}

// assertExistingUser ensures the specified user's existence.
func (r *UserReconciler) assertExistingUser(ctx context.Context, harborClient *h.RESTClient,
	usr *v1alpha2.User) error {
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

	heldUser, err := harborClient.GetUserByName(ctx, usr.Spec.Name)
	if err != nil {
		switch err.Error() {
		case clienterrors.ErrUserNotFoundMsg:
			usr.Status.PasswordHash = pwHash.Short()

			return r.createUser(ctx, harborClient, usr, pw)
		default:
			return err
		}
	}

	if usr.Status.PasswordHash == "" {
		usr.Status.PasswordHash = pwHash.Short()
	} else if usr.Status.PasswordHash != pwHash.Short() {
		usr.Status.PasswordHash = pwHash.Short()

		if err = harborClient.UpdateUserPassword(ctx, heldUser.UserID, &model.PasswordReq{
			NewPassword: pw,
		}); err != nil {
			return err
		}
	}

	return r.ensureUser(ctx, harborClient, heldUser, usr)
}

// createUser constructs a user request and triggers the Harbor API to create that user.
func (r *UserReconciler) createUser(ctx context.Context, harborClient *h.RESTClient, user *v1alpha2.User,
	newPassword string) error {

	return harborClient.NewUser(ctx,
		user.Spec.Name,
		user.Spec.Email,
		user.Spec.RealName,
		newPassword,
		user.Spec.Comments)
}

// labelsForUserSecret returns a list of labels for a user's secret.
func (r *UserReconciler) labelsForUserSecret(user *v1alpha2.User, instanceName string) map[string]string {
	return map[string]string{
		labelUserRegistry:  instanceName,
		labelUserComponent: user.Spec.Name + "-secret",
		labelUserRelease:   "harbor",
	}
}

// ensureUser updates a users profile, if changed.
func (r *UserReconciler) ensureUser(ctx context.Context, harborClient *h.RESTClient,
	heldUser *model.UserResp, desiredUser *v1alpha2.User) error {
	newUserProfile := &model.UserProfile{
		Comment:  desiredUser.Spec.Comments,
		Email:    desiredUser.Spec.Email,
		Realname: desiredUser.Spec.RealName,
	}

	// Default the 'sysAdmin' toggle to false, if no value is provided.
	if desiredUser.Spec.SysAdmin != nil {
		if err := harborClient.SetUserSysAdmin(ctx, heldUser.UserID, *desiredUser.Spec.SysAdmin); err != nil {
			return err
		}
	}

	if isUserProfileRequestEqual(heldUser, newUserProfile) {
		return nil
	}

	// Update the user profile if spec has changed
	return harborClient.UpdateUserProfile(ctx, heldUser.UserID, newUserProfile)
}

// isUserRequestEqual compares the individual values of an existing user with a 'new' user profile.
func isUserProfileRequestEqual(existing *model.UserResp, new *model.UserProfile) bool {
	if new.Email != existing.Email {
		return false
	} else if new.Realname != existing.Realname {
		return false
	} else if new.Comment != existing.Comment {
		return false
	}

	return true
}

// assertDeletedUser deletes a user, first ensuring its existence
func (r *UserReconciler) assertDeletedUser(ctx context.Context, log logr.Logger, harborClient *h.RESTClient,
	user *v1alpha2.User, patch client.Patch) error {
	harborUser, err := harborClient.GetUserByName(ctx, user.Spec.Name)
	if err != nil {
		if errors.Is(err, &clienterrors.ErrUserNotFound{}) {
			log.Info("pulling finalizer")
			controllerutil.RemoveFinalizer(user, internal.FinalizerName)
			return r.Client.Patch(ctx, user, patch)
		}
		return err
	}

	if err := harborClient.DeleteUser(ctx, harborUser.UserID); err != nil {
		return err
	}

	log.Info("pulling finalizer")
	controllerutil.RemoveFinalizer(user, internal.FinalizerName)

	return r.Client.Patch(ctx, user, patch)
}
