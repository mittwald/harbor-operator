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
	"sigs.k8s.io/controller-runtime/pkg/controller"

	"github.com/go-logr/logr"
	"github.com/mittwald/harbor-operator/controllers/config"
	"github.com/mittwald/harbor-operator/controllers/helper"
	"helm.sh/helm/v3/pkg/repo"
	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	registriesv1alpha2 "github.com/mittwald/harbor-operator/api/v1alpha2"
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
func (r *InstanceChartRepositoryReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	reqLogger := r.Log.WithValues("instancechartrepository", req.NamespacedName)

	reqLogger.Info("Reconciling InstanceChartRepository")

	ctx := context.Background()

	// Fetch the InstanceChartRepository instance
	instance := &registriesv1alpha2.InstanceChartRepository{}

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

	if err := r.reconcileInstanceChartRepositorySecret(ctx, instance); err != nil {
		return ctrl.Result{}, err
	}

	entry, err := r.specToRepoEntry(ctx, instance)
	if err != nil {
		return r.setErrStatus(ctx, instance, err)
	}

	helmClient, err := r.HelmClientReceiver(config.HelmClientRepoCachePath,
		config.HelmClientRepoConfPath, "")
	if err != nil {
		return ctrl.Result{}, err
	}

	err = helmClient.AddOrUpdateChartRepo(*entry)
	if err != nil {
		return r.setErrStatus(ctx, instance, err)
	}

	instance.Status.State = registriesv1alpha2.RepoStateReady
	if err = r.Client.Status().Update(ctx, instance); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *InstanceChartRepositoryReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&registriesv1alpha2.InstanceChartRepository{}).
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
func (r *InstanceChartRepositoryReconciler) reconcileInstanceChartRepositorySecret(ctx context.Context, i *registriesv1alpha2.InstanceChartRepository) error {
	if i.Spec.SecretRef != nil {
		secret, err := r.getSecret(ctx, i)
		if err != nil {
			return err
		}

		if len(secret.OwnerReferences) == 0 {
			err = ctrl.SetControllerReference(i, secret, r.Scheme)
			if err != nil {
				return err
			}
			if err = r.Client.Update(ctx, secret); err != nil {
				return err
			}
		}
	}
	return nil
}

// setErrStatus sets the error status of an InstanceChartRepository object.
func (r *InstanceChartRepositoryReconciler) setErrStatus(ctx context.Context,
	cr *registriesv1alpha2.InstanceChartRepository, err error) (ctrl.Result, error) {
	if cr == nil {
		return ctrl.Result{}, errors.New("no instance chart repo provided")
	}

	cr.Status.State = registriesv1alpha2.RepoStateError

	updateErr := r.Status().Update(ctx, cr)
	if updateErr != nil {
		return ctrl.Result{}, updateErr
	}

	return ctrl.Result{}, err
}

// specToRepoEntry constructs and returns a repository entry from an instancechartrepo CR object.
func (r *InstanceChartRepositoryReconciler) specToRepoEntry(ctx context.Context,
	cr *registriesv1alpha2.InstanceChartRepository) (*repo.Entry, error) {
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

	return &entry, nil
}

// getSecret gets and returns a kubernetes secret.
func (r *InstanceChartRepositoryReconciler) getSecret(ctx context.Context,
	cr *registriesv1alpha2.InstanceChartRepository) (*corev1.Secret, error) {
	var secret corev1.Secret

	existing, err := helper.ObjExists(ctx, r, cr.Spec.SecretRef.Name, cr.Namespace, &secret)
	if err != nil {
		return nil, err
	}

	if !existing {
		return nil, fmt.Errorf("secret %s not found, namespace: %s", cr.Spec.SecretRef.Name, cr.Namespace)
	}

	return &secret, nil
}
