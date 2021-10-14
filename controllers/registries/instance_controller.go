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
	"time"

	helmclient "github.com/mittwald/go-helm-client"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/mittwald/harbor-operator/apis/registries/v1alpha2"
	"github.com/mittwald/harbor-operator/controllers/registries/config"
	"github.com/mittwald/harbor-operator/controllers/registries/helper"
	"github.com/mittwald/harbor-operator/controllers/registries/internal"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
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
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha2.Instance{}).
		Complete(r)
}

// +kubebuilder:rbac:groups=registries.mittwald.de,resources=instances,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=registries.mittwald.de,resources=instances/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="",resources=persistentvolumeclaims,verbs=get;list;watch;create;update
// +kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;update;delete
// +kubebuilder:rbac:groups="apps",resources=deployments;statefulsets;replicasets,verbs=get;list;watch;create;update;delete;patch
// +kubebuilder:rbac:groups="networking.k8s.io",resources=ingresses,verbs=get;list;watch;create;update;delete;patch

// Reconcile reads that state of the cluster for a Instance object and makes changes based on the state read
// and what is in the Instance.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *InstanceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	reqLogger := r.Log.WithValues("instance", req.NamespacedName)
	reqLogger.Info("Reconciling Instance")

	// Fetch the Instance
	harbor := &v1alpha2.Instance{}
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
	patch := client.MergeFrom(harbor.DeepCopy())

	if harbor.DeletionTimestamp != nil &&
		harbor.Status.Phase.Name != v1alpha2.InstanceStatusPhaseTerminating {
		now := metav1.Now()
		harbor.Status.Phase = v1alpha2.InstanceStatusPhase{
			Name:           v1alpha2.InstanceStatusPhaseTerminating,
			Message:        "Deleted",
			LastTransition: &now,
		}
		harbor.Status.SpecHash = ""

		return ctrl.Result{}, r.Client.Status().Patch(ctx, harbor, patch)
	}

	switch harbor.Status.Phase.Name {
	default:
		return ctrl.Result{}, nil

	case "":
		harbor.Status.Phase.Name = v1alpha2.InstanceStatusPhaseInstalling
		harbor.Status.Phase.Message = "project is about to be created"
		harbor.Status.SpecHash = ""

	case v1alpha2.InstanceStatusPhaseInstalling:
		reqLogger.Info("Installing Helm chart")

		err := r.updateHelmRepos()
		if err != nil {
			return ctrl.Result{}, err
		}

		chartSpec, err := helper.InstanceToChartSpec(ctx, r.Client, harbor)
		if err != nil {
			return ctrl.Result{}, err
		}

		err = r.installOrUpgradeHelmChart(ctx, chartSpec)
		if err != nil {
			return ctrl.Result{RequeueAfter: 60 * time.Second}, err
		}

		harbor.Status.Phase.Name = v1alpha2.InstanceStatusPhaseInstalled
		harbor.Status.Phase.Message = "harbor was successfully installed"

		// Creating a spec hash of the chart spec pre-installation
		// ensures that it is set in "InstanceStatusPhaseInstalled", preventing the controller
		// to jump right back into "InstanceStatusPhaseInstalling"
		if specHash, err := helper.CreateSpecHash(chartSpec); err != nil {
			return ctrl.Result{}, err
		} else if harbor.Status.SpecHash == "" {
			harbor.Status.SpecHash = specHash

			return ctrl.Result{}, r.Client.Status().Patch(ctx, harbor, patch)
		}

	case v1alpha2.InstanceStatusPhaseInstalled:
		if harbor.Spec.GarbageCollection != nil {
			if err := r.reconcileGarbageCollection(ctx, harbor); err != nil {
				return ctrl.Result{RequeueAfter: 60 * time.Second}, err
			}
		}

		controllerutil.AddFinalizer(harbor, internal.FinalizerName)
		err := r.Client.Patch(ctx, harbor, patch)
		if err != nil {
			return ctrl.Result{}, err
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
			harbor.Status.Phase.Name = v1alpha2.InstanceStatusPhaseInstalling
			harbor.Status.SpecHash = specHash

			return ctrl.Result{}, r.Client.Status().Patch(ctx, harbor, patch)
		}

	case v1alpha2.InstanceStatusPhaseTerminating:
		err := r.reconcileTerminatingInstance(ctx, reqLogger, harbor, patch)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, r.Client.Status().Patch(ctx, harbor, patch)
}

// reconcileTerminatingInstance triggers a helm uninstall for the created release.
func (r *InstanceReconciler) reconcileTerminatingInstance(ctx context.Context, log logr.Logger,
	harbor *v1alpha2.Instance, patch client.Patch) error {
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

	log.Info("pulling finalizer")
	controllerutil.RemoveFinalizer(harbor, internal.FinalizerName)

	return r.Client.Patch(ctx, harbor, patch)
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
// If the install/upgrade operation fails however, a rollback to the latest release will be performed.
func (r *InstanceReconciler) installOrUpgradeHelmChart(ctx context.Context, helmChart *helmclient.ChartSpec) error {
	helmClient, err := r.HelmClientReceiver(config.Config.HelmClientRepositoryCachePath,
		config.Config.HelmClientRepositoryConfigPath, helmChart.Namespace)
	if err != nil {
		return err
	}

	helmChart.Timeout = 5 * time.Minute

	upgradedRelease, upgradeErr := helmClient.InstallOrUpgradeChart(ctx, helmChart)
	if upgradeErr != nil {
		if upgradedRelease != nil {
			rollbackErr := helmClient.RollbackRelease(helmChart, 0)
			if rollbackErr != nil {
				return fmt.Errorf("rollback failed: (%s), upgrade failed: %w", rollbackErr, upgradeErr)
			}
		}
		return upgradeErr
	}

	return nil
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
