package instancechartrepo

import (
	"context"
	"fmt"
	"github.com/mittwald/go-helm-client"
	registriesv1alpha1 "github.com/mittwald/harbor-operator/pkg/apis/registries/v1alpha1"
	"github.com/mittwald/harbor-operator/pkg/config"
	"github.com/mittwald/harbor-operator/pkg/internal/helper"
	"helm.sh/helm/v3/pkg/repo"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller_instancechartrepo")

// Add creates a new InstanceChartRepo Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileInstanceChartRepo{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("instancechartrepo-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource InstanceChartRepo
	err = c.Watch(&source.Kind{Type: &registriesv1alpha1.InstanceChartRepo{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// Watch for changes to secondary resource Pods and requeue the owner InstanceChartRepo
	err = c.Watch(&source.Kind{Type: &corev1.Pod{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &registriesv1alpha1.InstanceChartRepo{},
	})
	if err != nil {
		return err
	}

	return nil
}

// blank assignment to verify that ReconcileInstanceChartRepo implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileInstanceChartRepo{}

// ReconcileInstanceChartRepo reconciles a InstanceChartRepo object
type ReconcileInstanceChartRepo struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

func (r *ReconcileInstanceChartRepo) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling InstanceChartRepo")

	ctx := context.Background()

	// Fetch the InstanceChartRepo instance
	instance := &registriesv1alpha1.InstanceChartRepo{}
	err := r.client.Get(context.TODO(), request.NamespacedName, instance)
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

	entry, err := r.specToRepoEntry(ctx, instance)
	if err != nil {
		return r.setErrStatus(instance, err)
	}

	helmClient, err := helmclient.New(&helmclient.Options{
		RepositoryCache:  config.Config.HelmClientRepositoryCachePath,
		RepositoryConfig: config.Config.HelmClientRepositoryConfigPath,
	})
	if err != nil {
		return reconcile.Result{}, err
	}

	err = helmClient.AddOrUpdateChartRepo(*entry)
	if err != nil {
		return r.setErrStatus(instance, err)
	}

	instance.Status.State = registriesv1alpha1.RepoStateReady
	err = r.client.Status().Update(context.TODO(), instance)
	return reconcile.Result{}, err
}

// setErrStatus sets the error status of an instancechartrepo objec
func (r *ReconcileInstanceChartRepo) setErrStatus(cr *registriesv1alpha1.InstanceChartRepo, err error) (reconcile.Result, error) {
	cr.Status.State = registriesv1alpha1.RepoStateError
	updateErr := r.client.Status().Update(context.TODO(), cr)
	if updateErr != nil {
		return reconcile.Result{}, err
	}
	return reconcile.Result{}, err
}

// specToRepoEntry constructs and returns a repository entry from an instancechartrepo CR object
func (r *ReconcileInstanceChartRepo) specToRepoEntry(ctx context.Context, cr *registriesv1alpha1.InstanceChartRepo) (*repo.Entry, error) {
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

// getSecret gets and returns a kubernetes secret
func (r *ReconcileInstanceChartRepo) getSecret(ctx context.Context, cr *registriesv1alpha1.InstanceChartRepo) (*corev1.Secret, error) {
	var secret corev1.Secret
	existing, err := helper.ObjExists(ctx, r.client, cr.Spec.SecretRef.Name, cr.Namespace, &secret)
	if err != nil {
		return nil, err
	}

	if !existing {
		return nil, fmt.Errorf("secret %q not found", cr.Spec.SecretRef.Name)
	}

	return &secret, nil
}
