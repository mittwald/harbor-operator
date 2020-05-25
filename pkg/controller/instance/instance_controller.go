package instance

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/mittwald/harbor-operator/pkg/config"
	"github.com/mittwald/harbor-operator/pkg/controller/internal"

	"github.com/go-logr/logr"
	helmclient "github.com/mittwald/go-helm-client"
	registriesv1alpha1 "github.com/mittwald/harbor-operator/pkg/apis/registries/v1alpha1"
	"github.com/mittwald/harbor-operator/pkg/internal/helper"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
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

var log = logf.Log.WithName("controller_harbor")

// Add creates a new Instance Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	// This function is used to dynamically generate a helmclient
	// and is passed as a field value to the ReconcileInstance struct.
	f := func(repoCache, repoConfig, namespace string) (helmclient.Client, error) {
		opts := &helmclient.RestConfClientOptions{
			Options: &helmclient.Options{
				Namespace:        namespace,
				RepositoryCache:  repoCache,
				RepositoryConfig: repoConfig,
			},
			RestConfig: mgr.GetConfig(),
		}

		return helmclient.NewClientFromRestConf(opts)
	}

	return add(mgr, newReconciler(mgr, f))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager, f internal.HelmClientFactory) reconcile.Reconciler {
	return &ReconcileInstance{client: mgr.GetClient(), scheme: mgr.GetScheme(), helmClientReceiver: f}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
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

// blank assignment to verify that ReconcileInstance implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileInstance{}

// ReconcileInstance reconciles a Instance object
type ReconcileInstance struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver.
	client client.Client

	// The Scheme of this operator.
	scheme *runtime.Scheme

	// helmClientReceiver is a receiver function to generate a helmclient dynamically.
	helmClientReceiver internal.HelmClientFactory
}

// Reconcile reads that state of the cluster for a Instance object and makes changes based on the state read
// and what is in the Instance.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileInstance) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling Instance")

	ctx := context.Background()

	// Fetch the Instance
	harbor := &registriesv1alpha1.Instance{}
	if err := r.client.Get(ctx, request.NamespacedName, harbor); err != nil {
		if k8serrors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}
	reqLogger = reqLogger.WithValues("instanceName", harbor.Spec.Name)

	originalInstance := harbor.DeepCopy()

	if harbor.DeletionTimestamp != nil {
		now := metav1.Now()
		harbor.Status.Phase = registriesv1alpha1.InstanceStatusPhase{
			Name:           registriesv1alpha1.InstanceStatusPhaseTerminating,
			Message:        "Deleted",
			LastTransition: &now}
		return r.patchInstance(ctx, originalInstance, harbor)
	}

	switch harbor.Status.Phase.Name {
	default:
		return reconcile.Result{}, nil

	case "":
		harbor.Status.Phase.Name = registriesv1alpha1.InstanceStatusPhaseInstalling

	case registriesv1alpha1.InstanceStatusPhaseInstalling:
		reqLogger.Info("installing helm-chart")

		err := r.updateHelmRepos()
		if err != nil {
			return reconcile.Result{}, err
		}

		chartSpec, err := harbor.ToChartSpec(ctx, r.client)
		if err != nil {
			return reconcile.Result{}, err
		}

		helper.PushFinalizer(harbor, FinalizerName)

		err = r.installOrUpgradeHelmChart(chartSpec)
		if err != nil {
			return reconcile.Result{}, err
		}

		harbor.Status.Phase.Name = registriesv1alpha1.InstanceStatusPhaseReady
		harbor.Status.Version = harbor.Spec.Version

	case registriesv1alpha1.InstanceStatusPhaseReady:
		if harbor.Spec.GarbageCollection != nil {
			if err := r.reconcileGarbageCollection(ctx, harbor); err != nil {
				return reconcile.Result{}, err
			}
		}

		chartSpec, err := harbor.ToChartSpec(ctx, r.client)
		if err != nil {
			return reconcile.Result{}, err
		}

		specHash, err := r.createSpecHash(chartSpec)
		if err != nil {
			return reconcile.Result{}, err
		}

		if harbor.Status.SpecHash != specHash {
			harbor.Status.Phase.Name = registriesv1alpha1.InstanceStatusPhaseInstalling
			harbor.Status.SpecHash = specHash
			return r.patchInstance(ctx, originalInstance, harbor)
		}

	case registriesv1alpha1.InstanceStatusPhaseTerminating:
		err := r.reconcileTerminatingInstance(ctx, reqLogger, harbor)
		if err != nil {
			return reconcile.Result{}, err
		}
	}

	return r.patchInstance(ctx, originalInstance, harbor)
}

// reconcileTerminatingInstance triggers a helm uninstall for the created release
func (r *ReconcileInstance) reconcileTerminatingInstance(ctx context.Context, log logr.Logger, harbor *registriesv1alpha1.Instance) error {
	if harbor == nil {
		return errors.New("no harbor instance provided")
	}

	chartSpec, err := harbor.ToChartSpec(ctx, r.client)
	if err != nil {
		return err
	}

	log.Info("deleting helm release", "release", chartSpec.ReleaseName)

	err = r.uninstallHelmRelease(chartSpec)
	if err != nil {
		return err
	}

	log.Info("pulling finalizers")
	helper.PullFinalizer(harbor, FinalizerName)

	return nil
}

// createSpecHash returns a hash string constructed with the helm chart spec
func (r *ReconcileInstance) createSpecHash(spec *helmclient.ChartSpec) (string, error) {
	hashSrc, err := json.Marshal(spec)
	if err != nil {
		return "", err
	}

	toHash := []interface{}{hashSrc}
	hash, err := helper.GenerateHashFromInterfaces(toHash)
	if err != nil {
		return "", err
	}

	return hash.String(), nil
}

// patchInstance compares the new CR status and finalizers with the pre-existing ones and updates them accordingly
func (r *ReconcileInstance) patchInstance(ctx context.Context, originalInstance, instance *registriesv1alpha1.Instance) (reconcile.Result, error) {
	// Update Status
	if !reflect.DeepEqual(originalInstance.Status, instance.Status) {
		originalInstance.Status = instance.Status
		if err := r.client.Status().Update(ctx, originalInstance); err != nil {
			return reconcile.Result{}, err
		}
	}

	// Update Finalizers
	if !reflect.DeepEqual(originalInstance.Finalizers, instance.Finalizers) {
		originalInstance.Finalizers = instance.Finalizers
	}

	if err := r.client.Update(ctx, originalInstance); err != nil {
		return reconcile.Result{}, err
	}

	return reconcile.Result{Requeue: true}, nil
}

// updateHelmRepos updates helm chart repositories
func (r *ReconcileInstance) updateHelmRepos() error {
	helmClient, err := r.helmClientReceiver(config.Config.HelmClientRepositoryCachePath,
		config.Config.HelmClientRepositoryConfigPath, "")

	if err != nil {
		return err
	}

	return helmClient.UpdateChartRepos()
}

// installOrUpgradeHelmChart installs and upgrades a helm chart
func (r *ReconcileInstance) installOrUpgradeHelmChart(helmChart *helmclient.ChartSpec) error {
	helmClient, err := r.helmClientReceiver(config.Config.HelmClientRepositoryCachePath,
		config.Config.HelmClientRepositoryConfigPath, helmChart.Namespace)

	if err != nil {
		return err
	}

	return helmClient.InstallOrUpgradeChart(helmChart)
}

// uninstallHelmRelease uninstalls a helm release
func (r *ReconcileInstance) uninstallHelmRelease(helmChart *helmclient.ChartSpec) error {
	helmClient, err := r.helmClientReceiver(config.Config.HelmClientRepositoryCachePath,
		config.Config.HelmClientRepositoryConfigPath, helmChart.Namespace)

	if err != nil {
		return err
	}

	return helmClient.UninstallRelease(helmChart)
}
