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
	"fmt"
	"reflect"
	"time"

	h "github.com/mittwald/goharbor-client/v3/apiv2"
	legacymodel "github.com/mittwald/goharbor-client/v3/apiv2/model/legacy"
	userapi "github.com/mittwald/goharbor-client/v3/apiv2/user"
	"github.com/mittwald/harbor-operator/controllers/helper"
	"github.com/mittwald/harbor-operator/controllers/internal"
	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	registriesv1alpha1 "github.com/mittwald/harbor-operator/api/v1alpha1"
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

// +kubebuilder:rbac:groups=registries.mittwald.de,resources=users,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=registries.mittwald.de,resources=users/status,verbs=get;update;patch
func (r *UserReconciler) Reconcile(req ctrl.Request) (res ctrl.Result, err error) {
	reqLogger := r.Log.WithValues("user", req.NamespacedName)
	reqLogger.Info("Reconciling User")

	now := metav1.Now()
	ctx := context.Background()

	// Fetch the User instance
	user := &registriesv1alpha1.User{}

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

	if user.ObjectMeta.DeletionTimestamp != nil && user.Status.Phase != registriesv1alpha1.UserStatusPhaseTerminating {
		user.Status = registriesv1alpha1.UserStatus{Phase: registriesv1alpha1.UserStatusPhaseTerminating}
		res = ctrl.Result{Requeue: true}

		return r.updateUserCR(ctx, nil, originalUser, user, res)
	}

	// Fetch the Instance
	harbor, err := internal.FetchReadyHarborInstance(ctx, user.Namespace, user.Spec.ParentInstance.Name, r.Client)
	if err != nil {
		if _, ok := err.(internal.ErrInstanceNotFound); ok {
			helper.PullFinalizer(user, internal.FinalizerName)

			res = ctrl.Result{}
		} else if _, ok := err.(internal.ErrInstanceNotReady); ok {
			return ctrl.Result{RequeueAfter: 30 * time.Second}, err
		} else {
			user.Status = registriesv1alpha1.UserStatus{LastTransition: &now}
			res = ctrl.Result{RequeueAfter: 120 * time.Second}
		}

		return r.updateUserCR(ctx, nil, originalUser, user, res)
	}

	// Build a client to connect to the harbor API
	harborClient, err := internal.BuildClient(ctx, r.Client, harbor)
	if err != nil {
		return reconcile.Result{Requeue: true}, err
	}

	// Handle user reconciliation
	switch user.Status.Phase {
	default:
		return ctrl.Result{}, nil

	case registriesv1alpha1.UserStatusPhaseUnknown:
		user.Status.Phase = registriesv1alpha1.UserStatusPhaseCreating
		if err := r.Client.Status().Update(ctx, user); err != nil {
			return ctrl.Result{}, err
		}
		return reconcile.Result{Requeue: true}, nil

	case registriesv1alpha1.UserStatusPhaseCreating:
		if err := r.assertExistingUser(ctx, harborClient, user); err != nil {
			return ctrl.Result{}, err
		}

		helper.PushFinalizer(user, internal.FinalizerName)

		user.Status.Phase = registriesv1alpha1.UserStatusPhaseReady
		res = reconcile.Result{Requeue: true}

	case registriesv1alpha1.UserStatusPhaseReady:
		err := r.assertExistingUser(ctx, harborClient, user)
		if err != nil {
			return ctrl.Result{}, err
		}

		res = ctrl.Result{}

	case registriesv1alpha1.UserStatusPhaseTerminating:
		err := r.assertDeletedUser(ctx, reqLogger, harborClient, user)
		if err != nil {
			return ctrl.Result{}, err
		}

		res = ctrl.Result{}
	}

	return r.updateUserCR(ctx, harbor, originalUser, user, res)
}

// updateUserCR compares the new CR status and finalizers with the pre-existing ones and updates them accordingly.
func (r *UserReconciler) updateUserCR(ctx context.Context, parentInstance *registriesv1alpha1.Instance, originalUser,
	user *registriesv1alpha1.User, result reconcile.Result) (ctrl.Result, error) {
	if originalUser == nil || user == nil {
		return ctrl.Result{},
			fmt.Errorf("cannot update user because the original user has not been set")
	}

	// Update Status
	if !reflect.DeepEqual(originalUser.Status, user.Status) {
		if err := r.Client.Status().Update(ctx, user); err != nil {
			return ctrl.Result{}, err
		}
	}

	// set owner
	if len(user.OwnerReferences) == 0 && parentInstance != nil {
		if err := controllerruntime.SetControllerReference(parentInstance, user, r.Scheme); err != nil {
			return ctrl.Result{}, err
		}
	}

	if !reflect.DeepEqual(originalUser, user) {
		if err := r.Client.Update(ctx, user); err != nil {
			return ctrl.Result{}, err
		}
	}

	return result, nil
}

// assertExistingUser ensures the specified user's existence.
func (r *UserReconciler) assertExistingUser(ctx context.Context, harborClient *h.RESTClient,
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
func (r *UserReconciler) createUser(ctx context.Context, harborClient *h.RESTClient, user *registriesv1alpha1.User,
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
func (r *UserReconciler) labelsForUserSecret(user *registriesv1alpha1.User, instanceName string) map[string]string {
	return map[string]string{
		labelUserRegistry:  instanceName,
		labelUserComponent: user.Spec.Name + "-secret",
		labelUserRelease:   "harbor",
	}
}

// ensureUser updates a users profile, if changed - Afterwards, it updates the password if changed.
func (r *UserReconciler) ensureUser(ctx context.Context, harborClient *h.RESTClient,
	heldUser *legacymodel.User, desiredUser *registriesv1alpha1.User) error {
	newUsr := &legacymodel.User{
		UserID: heldUser.UserID,
	}

	newUsr.Username = desiredUser.Spec.Name
	newUsr.Email = desiredUser.Spec.Email
	newUsr.Realname = desiredUser.Spec.RealName
	newUsr.SysadminFlag = desiredUser.Spec.SysAdmin

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
	user *registriesv1alpha1.User) error {
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

	return nil
}
