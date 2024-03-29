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

	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/mittwald/harbor-operator/apis/registries/v1alpha2"

	"github.com/go-logr/logr"
	"helm.sh/helm/v3/pkg/repo"
	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/mittwald/harbor-operator/controllers/registries/config"
	"github.com/mittwald/harbor-operator/controllers/registries/helper"
)

// InstanceChartRepositoryReconciler reconciles a InstanceChartRepository object
type InstanceChartRepositoryReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
	// helmClientReceiver is a receiver function to generate a helmclient dynamically.
	HelmClientReceiver HelmClientFactory
}

// +kubebuilder:rbac:groups=registries.mittwald.de,resources=instancechartrepositories,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=registries.mittwald.de,resources=instancechartrepositories/status,verbs=get;update;patch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *InstanceChartRepositoryReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	reqLogger := r.Log.WithValues("instancechartrepository", req.NamespacedName)

	reqLogger.Info("Reconciling InstanceChartRepository")

	// Fetch the InstanceChartRepository instance
	instance := &v1alpha2.InstanceChartRepository{}

	patch := client.MergeFrom(instance.DeepCopy())

	err := r.Client.Get(ctx, req.NamespacedName, instance)
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

	if instance.Spec.SecretRef != nil {
		if err := r.reconcileInstanceChartRepositorySecret(ctx, instance); err != nil {
			return ctrl.Result{}, err
		}
	}

	entry, err := r.specToRepoEntry(ctx, instance)
	if err != nil {
		instance.Status.State = v1alpha2.RepoStateError
		return ctrl.Result{}, r.Client.Status().Patch(ctx, instance, patch)
	}

	helmClient, err := r.HelmClientReceiver(config.HelmClientRepoCachePath,
		config.HelmClientRepoConfPath, "")
	if err != nil {
		return ctrl.Result{}, err
	}

	err = helmClient.AddOrUpdateChartRepo(*entry)
	if err != nil {
		instance.Status.State = v1alpha2.RepoStateError
		return ctrl.Result{}, r.Client.Status().Patch(ctx, instance, patch)
	}

	instance.Status.State = v1alpha2.RepoStateReady

	return ctrl.Result{}, r.Client.Status().Patch(ctx, instance, patch)
}

func (r *InstanceChartRepositoryReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha2.InstanceChartRepository{}).
		Owns(&corev1.Secret{}).
		Owns(&corev1.Pod{}).
		WithOptions(controller.Options{
			MaxConcurrentReconciles: 1,
		}).
		Complete(r)
}

// reconcileInstanceChartRepositorySecret fetches the secret specified in an
// InstanceChartRepository's spec and sets an OwnerReference to the owned Object.
// Returns nil when the OwnerReference has been successfully set, or when no secret is specified.
func (r *InstanceChartRepositoryReconciler) reconcileInstanceChartRepositorySecret(ctx context.Context,
	i *v1alpha2.InstanceChartRepository) error {
	secret, err := r.getSecret(ctx, i)
	if err != nil {
		return err
	}

	return controllerutil.SetOwnerReference(i, secret, r.Scheme)
}

// specToRepoEntry constructs and returns a repository entry from an instancechartrepository CR object.
func (r *InstanceChartRepositoryReconciler) specToRepoEntry(ctx context.Context,
	cr *v1alpha2.InstanceChartRepository) (*repo.Entry, error) {
	if cr == nil {
		return nil, errors.New("no instance chart repo provided")
	}

	entry := repo.Entry{
		Name: cr.Name,
		URL:  cr.Spec.URL,
	}

	if cr.Spec.Name != "" {
		entry.Name = cr.Spec.Name
	}

	if cr.Spec.SecretRef == nil {
		return &entry, nil
	}

	secret, err := r.getSecret(ctx, cr)
	if err != nil {
		return nil, err
	}

	if secret == nil {
		return &entry, nil
	}

	entry.Username = string(secret.Data["username"])
	entry.Password = string(secret.Data["password"])
	entry.CertFile = string(secret.Data["certFile"])
	entry.KeyFile = string(secret.Data["keyFile"])
	entry.CAFile = string(secret.Data["caFile"])
	entry.PassCredentialsAll = true

	return &entry, nil
}

// getSecret gets and returns the kubernetes secret that is held in an instancechartrepositories spec.
func (r *InstanceChartRepositoryReconciler) getSecret(ctx context.Context,
	cr *v1alpha2.InstanceChartRepository) (*corev1.Secret, error) {
	var secret corev1.Secret

	exists, err := helper.ObjExists(ctx, r.Client, cr.Spec.SecretRef.Name, cr.Namespace, &secret)
	if err != nil {
		return nil, err
	}

	if !exists {
		return nil, fmt.Errorf("secret %s not found, namespace: %s", cr.Spec.SecretRef.Name, cr.Namespace)
	}

	return &secret, nil
}
