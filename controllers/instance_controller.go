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
	"reflect"
	"time"

	helmclient "github.com/mittwald/go-helm-client"
	"github.com/mittwald/harbor-operator/controllers/config"
	"github.com/mittwald/harbor-operator/controllers/helper"
	"github.com/mittwald/harbor-operator/controllers/internal"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

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

// InstanceReconciler reconciles a Instance object
type InstanceReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
	// helmClientReceiver is a receiver function to generate a helmclient dynamically.
	HelmClientReceiver HelmClientFactory
}

func (r *InstanceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// Create a new controller
	c, err := controller.New("harbor-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource Instance
	err = c.Watch(&source.Kind{Type: &registriesv1alpha1.Instance{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	return nil
}

// blank assignment to verify that InstanceReconciler implements reconcile.Reconciler.
var _ reconcile.Reconciler = &InstanceReconciler{}

// Reconcile reads that state of the cluster for a Instance object and makes changes based on the state read
// and what is in the Instance.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.

// +kubebuilder:rbac:groups=registries.mittwald.de,resources=instances,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=registries.mittwald.de,resources=instances/status,verbs=get;update;patch
func (r *InstanceReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	reqLogger := r.Log.WithValues("instance", req.NamespacedName)
	reqLogger.Info("Reconciling Instance")

	ctx := context.Background()

	// Fetch the Instance
	harbor := &registriesv1alpha1.Instance{}
	if err := r.Client.Get(ctx, req.NamespacedName, harbor); err != nil {
		if k8serrors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return ctrl.Result{}, err
	}

	reqLogger = reqLogger.WithValues("instanceName", harbor.Spec.Name)
	originalInstance := harbor.DeepCopy()

	if harbor.DeletionTimestamp != nil &&
		harbor.Status.Phase.Name != registriesv1alpha1.InstanceStatusPhaseTerminating {
		now := metav1.Now()
		harbor.Status.Phase = registriesv1alpha1.InstanceStatusPhase{
			Name:           registriesv1alpha1.InstanceStatusPhaseTerminating,
			Message:        "Deleted",
			LastTransition: &now,
		}

		return r.updateInstanceCR(ctx, originalInstance, harbor)
	}

	switch harbor.Status.Phase.Name {
	default:
		return ctrl.Result{}, nil

	case "":
		harbor.Status.Phase.Name = registriesv1alpha1.InstanceStatusPhaseInstalling

	case registriesv1alpha1.InstanceStatusPhaseInstalling:
		reqLogger.Info("Installing Helm chart")

		err := r.updateHelmRepos()
		if err != nil {
			return ctrl.Result{}, err
		}

		chartSpec, err := helper.InstanceToChartSpec(ctx, r.Client, harbor)
		if err != nil {
			return ctrl.Result{}, err
		}

		helper.PushFinalizer(harbor, internal.FinalizerName)

		err = r.installOrUpgradeHelmChart(ctx, chartSpec)
		if err != nil {
			return ctrl.Result{RequeueAfter: 60 * time.Second}, err
		}

		harbor.Status.Phase.Name = registriesv1alpha1.InstanceStatusPhaseInstalled
		harbor.Status.Version = harbor.Spec.HelmChart.Version

		// Creating a spec hash of the chart spec pre-installation
		// ensures that it is set in "InstanceStatusPhaseInstalled", preventing the controller
		// to jump right back into "InstanceStatusPhaseInstalling"
		if specHash, err := helper.CreateSpecHash(chartSpec); err != nil {
			return ctrl.Result{}, err
		} else if harbor.Status.SpecHash == "" {
			harbor.Status.SpecHash = specHash

			return r.updateInstanceCR(ctx, originalInstance, harbor)
		}

	case registriesv1alpha1.InstanceStatusPhaseInstalled:
		if harbor.Spec.GarbageCollection != nil {
			if err := r.reconcileGarbageCollection(ctx, harbor); err != nil {
				return ctrl.Result{RequeueAfter: 60 * time.Second}, err
			}
		}

		chartSpec, err := helper.InstanceToChartSpec(ctx, r.Client, harbor)
		if err != nil {
			return ctrl.Result{}, err
		}

		specHash, err := helper.CreateSpecHash(chartSpec)
		if err != nil {
			return ctrl.Result{}, err
		}

		if harbor.Status.SpecHash != specHash {
			harbor.Status.Phase.Name = registriesv1alpha1.InstanceStatusPhaseInstalling
			harbor.Status.SpecHash = specHash

			return r.updateInstanceCR(ctx, originalInstance, harbor)
		}

	case registriesv1alpha1.InstanceStatusPhaseTerminating:
		err := r.reconcileTerminatingInstance(ctx, reqLogger, harbor)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	return r.updateInstanceCR(ctx, originalInstance, harbor)
}

// reconcileTerminatingInstance triggers a helm uninstall for the created release.
func (r *InstanceReconciler) reconcileTerminatingInstance(ctx context.Context, log logr.Logger,
	harbor *registriesv1alpha1.Instance) error {
	if harbor == nil {
		return errors.New("no harbor instance provided")
	}

	chartSpec, err := helper.InstanceToChartSpec(ctx, r.Client, harbor)
	if err != nil {
		return err
	}

	log.Info("deleting helm release", "release", chartSpec.ReleaseName)

	err = r.uninstallHelmRelease(chartSpec)
	if err != nil {
		return err
	}

	log.Info("pulling finalizers")
	helper.PullFinalizer(harbor, internal.FinalizerName)

	return nil
}

// updateInstanceCR compares the new CR status and finalizers with the pre-existing ones and updates them accordingly.
func (r *InstanceReconciler) updateInstanceCR(ctx context.Context, originalInstance,
	instance *registriesv1alpha1.Instance) (ctrl.Result, error) {
	if originalInstance == nil || instance == nil {
		return ctrl.Result{}, fmt.Errorf("cannot update instance '%s' because the original instance is nil",
			instance.Spec.Name)
	}

	// Update Status
	if !reflect.DeepEqual(originalInstance.Status, instance.Status) {
		originalInstance.Status = instance.Status
		if err := r.Client.Status().Update(ctx, originalInstance); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Update Finalizers
	if !reflect.DeepEqual(originalInstance.Finalizers, instance.Finalizers) {
		originalInstance.Finalizers = instance.Finalizers
	}

	if err := r.Client.Update(ctx, originalInstance); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{Requeue: true}, nil
}

// updateHelmRepos updates helm chart repositories.
func (r *InstanceReconciler) updateHelmRepos() error {
	helmClient, err := r.HelmClientReceiver(config.Config.HelmClientRepositoryCachePath,
		config.Config.HelmClientRepositoryConfigPath, "")
	if err != nil {
		return err
	}

	return helmClient.UpdateChartRepos()
}

// installOrUpgradeHelmChart installs and upgrades a helm chart.
func (r *InstanceReconciler) installOrUpgradeHelmChart(ctx context.Context, helmChart *helmclient.ChartSpec) error {
	helmClient, err := r.HelmClientReceiver(config.Config.HelmClientRepositoryCachePath,
		config.Config.HelmClientRepositoryConfigPath, helmChart.Namespace)
	if err != nil {
		return err
	}

	return helmClient.InstallOrUpgradeChart(ctx, helmChart)
}

// uninstallHelmRelease uninstalls a helm release.
func (r *InstanceReconciler) uninstallHelmRelease(helmChart *helmclient.ChartSpec) error {
	helmClient, err := r.HelmClientReceiver(config.Config.HelmClientRepositoryCachePath,
		config.Config.HelmClientRepositoryConfigPath, helmChart.Namespace)
	if err != nil {
		return err
	}

	return helmClient.UninstallRelease(helmChart)
}
