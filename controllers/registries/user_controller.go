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

	"github.com/mittwald/harbor-operator/apis/registries/v1alpha2"
	controllererrors "github.com/mittwald/harbor-operator/controllers/registries/errors"
	"sigs.k8s.io/controller-runtime/pkg/controller"

	"github.com/go-logr/logr"
	h "github.com/mittwald/goharbor-client/v3/apiv2"
	legacymodel "github.com/mittwald/goharbor-client/v3/apiv2/model/legacy"
	userapi "github.com/mittwald/goharbor-client/v3/apiv2/user"
	"github.com/mittwald/harbor-operator/controllers/registries/helper"
	"github.com/mittwald/harbor-operator/controllers/registries/internal"
	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
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

	originalUser := user.DeepCopy()

	if user.ObjectMeta.DeletionTimestamp != nil && user.Status.Phase != v1alpha2.UserStatusPhaseTerminating {
		user.Status = v1alpha2.UserStatus{Phase: v1alpha2.UserStatusPhaseTerminating}

		return r.updateUserCR(ctx, nil, originalUser, user)
	}

	// Fetch the goharbor instance if it exists and is properly set up.
	// If the above does not apply, pull the finalizer from the user object.
	harbor, err := helper.GetOperationalHarborInstance(ctx, client.ObjectKey{
		Namespace: user.Namespace,
		Name:      user.Spec.ParentInstance.Name,
	}, r.Client)
	if err != nil {
		if errors.Is(err, &controllererrors.ErrInstanceNotFound{}) ||
			errors.Is(err, &controllererrors.ErrInstanceNotInstalled{}) {
			helper.PullFinalizer(user, internal.FinalizerName)
			helper.PullFinalizer(user, internal.OldFinalizerName)
			return r.updateUserCR(ctx, harbor, originalUser, user)
		}
		return ctrl.Result{}, err
	}

	// Build a client to connect to the harbor API
	harborClient, err := internal.BuildClient(ctx, r.Client, harbor)
	if err != nil {
		return ctrl.Result{}, err
	}

	// Check the Harbor API if it's reporting as healthy
	instanceIsHealthy, err := internal.HarborInstanceIsHealthy(ctx, harborClient)
	if err != nil {
		return ctrl.Result{}, err
	}
	if !instanceIsHealthy {
		return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
	}

	// Handle user reconciliation
	switch user.Status.Phase {
	default:
		return ctrl.Result{}, nil

	case v1alpha2.UserStatusPhaseUnknown:
		user.Status.Phase = v1alpha2.UserStatusPhaseCreating
		if err := r.Client.Status().Update(ctx, user); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil

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

		helper.PushFinalizer(user, internal.FinalizerName)

		user.Status.Phase = v1alpha2.UserStatusPhaseReady
		res = ctrl.Result{Requeue: true}

	case v1alpha2.UserStatusPhaseReady:
		err := r.assertExistingUser(ctx, harborClient, user)
		if err != nil {
			return ctrl.Result{}, err
		}

		res = ctrl.Result{}

	case v1alpha2.UserStatusPhaseTerminating:
		err := r.assertDeletedUser(ctx, reqLogger, harborClient, user)
		if err != nil {
			return ctrl.Result{}, err
		}

		res = ctrl.Result{}
	}

	return r.updateUserCR(ctx, harbor, originalUser, user)
}

// updateUserCR compares the new CR status and finalizers with the pre-existing ones and updates them accordingly.
func (r *UserReconciler) updateUserCR(ctx context.Context, parentInstance *v1alpha2.Instance, originalUser,
	user *v1alpha2.User) (ctrl.Result, error) {
	if originalUser == nil || user == nil {
		return ctrl.Result{}, fmt.Errorf("cannot update user because the original user has not been set")
	}

	// Update Status
	if !reflect.DeepEqual(originalUser.Status, user.Status) {
		if err := r.Client.Status().Update(ctx, user); err != nil {
			return ctrl.Result{}, err
		}
	}

	// set owner
	if len(user.OwnerReferences) == 0 && parentInstance != nil {
		if err := ctrl.SetControllerReference(parentInstance, user, r.Scheme); err != nil {
			return ctrl.Result{}, err
		}
	}

	if !reflect.DeepEqual(originalUser, user) {
		if err := r.Client.Update(ctx, user); err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
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

	heldUser, err := harborClient.GetUser(ctx, usr.Spec.Name)
	if err != nil {
		switch err.Error() {
		case userapi.ErrUserNotFoundMsg:
			usr.Status.PasswordHash = pwHash.Short()

			return r.createUser(ctx, harborClient, usr, pw)
		default:
			return err
		}
	}

	if usr.Status.PasswordHash != pwHash.Short() {
		usr.Status.PasswordHash = pwHash.Short()

		if err = harborClient.UpdateUserPassword(ctx, heldUser.UserID,
			&legacymodel.Password{
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
func (r *UserReconciler) labelsForUserSecret(user *v1alpha2.User, instanceName string) map[string]string {
	return map[string]string{
		labelUserRegistry:  instanceName,
		labelUserComponent: user.Spec.Name + "-secret",
		labelUserRelease:   "harbor",
	}
}

// ensureUser updates a users profile, if changed - Afterwards, it updates the password if changed.
func (r *UserReconciler) ensureUser(ctx context.Context, harborClient *h.RESTClient,
	heldUser *legacymodel.User, desiredUser *v1alpha2.User) error {
	newUsr := &legacymodel.User{
		UserID: heldUser.UserID,
	}

	newUsr.Username = desiredUser.Spec.Name
	newUsr.Email = desiredUser.Spec.Email
	newUsr.Realname = desiredUser.Spec.RealName

	// Default the 'sysAdmin' toggle to false, if no value is provided.
	if desiredUser.Spec.SysAdmin != nil {
		newUsr.SysadminFlag = *desiredUser.Spec.SysAdmin
	} else {
		newUsr.SysadminFlag = false
	}

	if isUserRequestEqual(heldUser, newUsr) {
		return nil
	}

	// Update the user if spec has changed
	return harborClient.UpdateUser(ctx, newUsr)
}

// isUserRequestEqual compares the individual values of an existing user with a user request
func isUserRequestEqual(existing, new *legacymodel.User) bool {
	if new.Username != existing.Username {
		return false
	} else if new.Email != existing.Email {
		return false
	} else if new.Realname != existing.Realname {
		return false
	} else if new.RoleID != existing.RoleID {
		return false
	} else if new.SysadminFlag != existing.SysadminFlag {
		return false
	}

	return true
}

// assertDeletedUser deletes a user, first ensuring its existence
func (r *UserReconciler) assertDeletedUser(ctx context.Context, log logr.Logger, harborClient *h.RESTClient,
	user *v1alpha2.User) error {
	harborUser, err := harborClient.GetUser(ctx, user.Spec.Name)
	if err != nil {
		return err
	}

	uReq := &legacymodel.User{
		Username: harborUser.Username,
		UserID:   harborUser.UserID,
	}

	if err := harborClient.DeleteUser(ctx, uReq); err != nil {
		return err
	}

	log.Info("pulling finalizers")
	helper.PullFinalizer(user, internal.FinalizerName)
	helper.PullFinalizer(user, internal.OldFinalizerName)

	return nil
}
