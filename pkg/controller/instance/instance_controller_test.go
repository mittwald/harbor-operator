package instance

import (
	"context"
	"testing"
	"time"

	"github.com/mittwald/harbor-operator/pkg/internal/helper"

	"github.com/golang/mock/gomock"
	helmclient "github.com/mittwald/go-helm-client"
	helmclientmock "github.com/mittwald/go-helm-client/mock"
	registriesv1alpha1 "github.com/mittwald/harbor-operator/pkg/apis/registries/v1alpha1"
	testingregistriesv1alpha1 "github.com/mittwald/harbor-operator/pkg/testing/registriesv1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	ns   string = "test-namespace"
	name string = "test-instance"
)

// buildReconcileWithFakeClientWithMocks
// returns a reconcile with fake client, schemes and mock objects
// reference: https://github.com/aerogear/mobile-security-service-operator/blob/e74272a6c7addebdc77b18eeffb5e888b35f4dfd/pkg/controller/mobilesecurityservice/fakeclient_test.go#L14
func buildReconcileWithFakeClientWithMocks(objs []runtime.Object) *ReconcileInstance {
	s := scheme.Scheme

	s.AddKnownTypes(
		registriesv1alpha1.SchemeGroupVersion,
		&registriesv1alpha1.Instance{},
	)

	// create a fake client to mock API calls with the mock objects
	cl := fake.NewFakeClientWithScheme(s, objs...)

	// create a ReconcileInstance object with the scheme and fake client
	return &ReconcileInstance{client: cl, scheme: s}
}

// TestInstanceController_Empty_Instance_Spec
// Test reconciliation with an empty instance object which is expected to be requeued
func TestInstanceController_Empty_Instance_Spec(t *testing.T) {
	i := registriesv1alpha1.Instance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
		},
	}

	r := buildReconcileWithFakeClientWithMocks([]runtime.Object{&i})

	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      i.Name,
			Namespace: i.Namespace,
		},
	}

	res, err := r.Reconcile(req)
	if err != nil {
		t.Fatalf("reconcile returned error: (%v)", err)
	}

	if !res.Requeue {
		t.Error("reconciliation was not requeued")
	}
}

// TestInstanceController_Empty_Instance_Spec
// Test reconciliation with a mocked instance object which is expected to be requeued
func TestInstanceController_With_Instance_Spec(t *testing.T) {
	i := testingregistriesv1alpha1.CreateInstance(name, ns)

	r := buildReconcileWithFakeClientWithMocks([]runtime.Object{&i})

	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      i.Name,
			Namespace: i.Namespace,
		},
	}

	res, err := r.Reconcile(req)
	if err != nil {
		t.Fatalf("reconcile returned error: (%v)", err)
	}

	if !res.Requeue {
		t.Error("reconciliation was not requeued")
	}
}

// TestInstanceController_Transition_Installing
// Test reconciliation with valid instance object which is expected not to be requeued
func TestInstanceController_Transition_Installing(t *testing.T) {
	i := testingregistriesv1alpha1.CreateInstance(name, ns)

	instanceSecret := testingregistriesv1alpha1.CreateSecret(i.Name+"-harbor-core", ns)
	i.Spec.HelmChart.ValuesYaml = `
	harborAdminPassword: {}
	proxy: {}
	nginx: 
	portal: {}
	core: {}
	jobservice: {}
	registry: {}
	  controller: {}
	  middleware: {}
	chartmuseum:
	  image: {}
	clair:
	  clair: {}
	  adapter: {}
	trivy:
	  image: {}
	notary:
	  server: {}
	  signer: {}
	database:
	  internal: {}
	  external: {}
	redis:
	  internal: {}
	  external: {}`

	r := buildReconcileWithFakeClientWithMocks([]runtime.Object{&i, &instanceSecret})

	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      i.Name,
			Namespace: i.Namespace,
		},
	}

	if i.Status.Phase.Name != "" {
		t.Error("instance status was not empty before reconciliation")
	}

	res, err := r.Reconcile(req)
	if err != nil {
		t.Fatalf("reconcile returned error: (%v)", err)
	}

	instance := &registriesv1alpha1.Instance{}

	err = r.client.Get(context.TODO(), req.NamespacedName, instance)
	if err != nil {
		t.Errorf("could not get instance: %v", err)
	}

	if !res.Requeue {
		t.Error("reconciliation was not requeued")
	}

	if instance.Status.Phase.Name != registriesv1alpha1.InstanceStatusPhaseInstalling {
		t.Errorf("instance status unexpected, status: %s, expected: %s", instance.Status.Phase.Name,
			registriesv1alpha1.InstanceStatusPhaseInstalling)
	}
}

// TestInstanceController_Instance_Installation
// Test if an instance object triggers the installation of an helm chart.
func TestInstanceController_Instance_Installation(t *testing.T) {
	i := testingregistriesv1alpha1.CreateInstance("harbor", "foobar")
	i.Status.Phase.Name = registriesv1alpha1.InstanceStatusPhaseInstalling

	chartSecret := testingregistriesv1alpha1.CreateSecret(i.Name+"-harbor-core", "foobar")
	r := buildReconcileWithFakeClientWithMocks([]runtime.Object{&i, &chartSecret})

	ctx := context.TODO()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockClient := helmclientmock.NewMockClient(ctrl)
	gomock.InOrder(
		mockClient.EXPECT().UpdateChartRepos(),
		mockClient.EXPECT().InstallOrUpgradeChart(ctx, &helmclient.ChartSpec{
			ReleaseName: i.Spec.HelmChart.ReleaseName,
			ChartName:   i.Spec.HelmChart.ChartName,
			Namespace:   i.Spec.HelmChart.Namespace,
			ValuesYaml:  i.Spec.HelmChart.ValuesYaml,
			Version:     i.Spec.HelmChart.Version,
		}),
	)

	r.helmClientReceiver = func(repoCache, repoConfig, namespace string) (helmclient.Client, error) {
		return helmclient.Client(mockClient), nil
	}

	req := reconcile.Request{NamespacedName: types.NamespacedName{Name: i.Name, Namespace: i.Namespace}}

	res, err := r.Reconcile(req)
	if err != nil {
		t.Fatalf("reconcile returned error: %v", err)
	}

	if !res.Requeue {
		t.Errorf("object got requeued, but should have not")
	}

	fetched := &registriesv1alpha1.Instance{}
	err = r.client.Get(context.TODO(), req.NamespacedName, fetched)

	if err != nil {
		t.Fatalf("could not get instance: %v", err)
	}

	if fetched.Status.SpecHash == "" {
		t.Errorf("spec hash has not been set on the instance")
	}

	if fetched.Status.Phase.Name != registriesv1alpha1.InstanceStatusPhaseInstalled {
		t.Errorf("wrong phase of received instance, want: %s, got: %s",
			registriesv1alpha1.InstanceStatusPhaseInstalled, i.Status.Phase.Name)
	}

	if fetched.Status.Version != i.Spec.Version {
		t.Errorf("wrong status.version of received instance, want: %s, got: %s",
			i.Spec.Version, fetched.Status.Version)
	}
}

// TestInstanceController_Instance_Deletion
// Test the deletion of an instance object.
// Expect the helm chart to be deleted and no finalizers on the object anymore,
// so Kubernetes would be able to delete the object.
func TestInstanceController_Instance_Deletion(t *testing.T) {
	i := testingregistriesv1alpha1.CreateInstance("harbor", "foobar")
	i.Status.Phase.Name = registriesv1alpha1.InstanceStatusPhaseTerminating
	i.SetFinalizers([]string{"harbor-operator.registries.mittwald.de"})

	r := buildReconcileWithFakeClientWithMocks([]runtime.Object{&i})

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockClient := helmclientmock.NewMockClient(ctrl)
	mockClient.EXPECT().UninstallRelease(&helmclient.ChartSpec{
		ReleaseName: i.Spec.HelmChart.ReleaseName,
		ChartName:   i.Spec.HelmChart.ChartName,
		Namespace:   i.Spec.HelmChart.Namespace,
		ValuesYaml:  i.Spec.HelmChart.ValuesYaml,
		Version:     i.Spec.HelmChart.Version,
	})

	r.helmClientReceiver = func(repoCache, repoConfig, namespace string) (helmclient.Client, error) {
		return helmclient.Client(mockClient), nil
	}

	req := reconcile.Request{NamespacedName: types.NamespacedName{Name: i.Name, Namespace: i.Namespace}}

	res, err := r.Reconcile(req)
	if err != nil {
		t.Fatalf("reconcile returned error: %v", err)
	}

	if !res.Requeue {
		t.Error("object should have been requeued")
	}

	fetched := &registriesv1alpha1.Instance{}

	err = r.client.Get(context.TODO(), req.NamespacedName, fetched)
	if err != nil {
		t.Fatalf("could not get instance: %v", err)
	}

	if len(fetched.GetFinalizers()) != 0 {
		t.Errorf("Unexpected length of finalizers, expected: %d, got: %d",
			0, len(fetched.GetFinalizers()))
	}
}

// TestInstanceController_Instance_Ready_Deletion
func TestInstanceController_Instance_Ready_Deletion(t *testing.T) {
	i := testingregistriesv1alpha1.CreateInstance("harbor", "foobar")
	i.Status.Phase.Name = registriesv1alpha1.InstanceStatusPhaseInstalled
	chartSecret := testingregistriesv1alpha1.CreateSecret(i.Name+"-harbor-core", "foobar")

	i.DeletionTimestamp = &metav1.Time{Time: time.Now()}

	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Namespace: i.Namespace,
			Name:      i.Name,
		},
	}

	r := buildReconcileWithFakeClientWithMocks([]runtime.Object{&i, &chartSecret})

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockClient := helmclientmock.NewMockClient(ctrl)
	mockClient.EXPECT().UninstallRelease(&helmclient.ChartSpec{
		ReleaseName: i.Spec.HelmChart.ReleaseName,
		ChartName:   i.Spec.HelmChart.ChartName,
		Namespace:   i.Spec.HelmChart.Namespace,
		ValuesYaml:  i.Spec.HelmChart.ValuesYaml,
		Version:     i.Spec.HelmChart.Version,
	})

	r.helmClientReceiver = func(repoCache, repoConfig, namespace string) (helmclient.Client, error) {
		return helmclient.Client(mockClient), nil
	}

	res, err := r.Reconcile(req)
	if err != nil {
		t.Fatalf("reconcile returned error: (%v)", err)
	}

	if !res.Requeue {
		t.Error("reconciliation was not requeued")
	}

	fetched := &registriesv1alpha1.Instance{}

	err = r.client.Get(context.TODO(), req.NamespacedName, fetched)
	if err != nil {
		t.Fatalf("could not get instance %s", err)
	}

	if fetched.Status.Phase.Name != registriesv1alpha1.InstanceStatusPhaseTerminating {
		t.Errorf("instance status unexpected: %s, expected: %s", fetched.Status.Phase.Name,
			registriesv1alpha1.InstanceStatusPhaseTerminating)
	}

	// Reconcile again here to run the steps in the `InstanceStatusPhaseTerminating` phase.
	// The prior reconcile-step only sets the phase to `InstanceStatusPhaseTerminating` but returns after that.
	res, err = r.Reconcile(req)
	if err != nil {
		t.Fatalf("reconcile returned error: (%v)", err)
	}

	if !res.Requeue {
		t.Error("reconciliation was not requeued")
	}
}

// TestInstanceController_Instance_Ready_Ensure_Chart_Spec tests the integrity of the helm chart spec
// by comparing a manually generated specHash to the specHash created by reconciliation
func TestInstanceController_Instance_Ready_Ensure_Chart_Spec(t *testing.T) {
	i := testingregistriesv1alpha1.CreateInstance("harbor", "foobar")
	i.Status.Phase.Name = registriesv1alpha1.InstanceStatusPhaseInstalled
	i.Spec.GarbageCollection = nil
	chartSecret := testingregistriesv1alpha1.CreateSecret(i.Name+"-harbor-core", "foobar")

	// Generate and add the spec hash of the helm chart to the instance object before reconciliation
	// (this let's us skip the reconciliation loop of "InstanceStatusPhaseInstalling")
	specHash, _ := helper.CreateSpecHash(&i.Spec.HelmChart.ChartSpec)
	i.Status.SpecHash = specHash

	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Namespace: i.Namespace,
			Name:      i.Name,
		},
	}

	r := buildReconcileWithFakeClientWithMocks([]runtime.Object{&i, &chartSecret})

	res, err := r.Reconcile(req)
	if err != nil {
		t.Fatalf("reconcile returned error: (%v)", err)
	}

	if !res.Requeue {
		t.Error("reconciliation was not requeued")
	}

	fetched := &registriesv1alpha1.Instance{}

	err = r.client.Get(context.TODO(), req.NamespacedName, fetched)
	if err != nil {
		t.Fatalf("could not get instance %s", err)
	}

	res, err = r.Reconcile(req)
	if err != nil {
		t.Fatalf("reconcile returned error: (%v)", err)
	}

	if !res.Requeue {
		t.Error("reconciliation was not requeued")
	}

	fetched = &registriesv1alpha1.Instance{}

	err = r.client.Get(context.TODO(), req.NamespacedName, fetched)
	if err != nil {
		t.Fatalf("could not get instance %s", err)
	}

	if fetched.Status.SpecHash != specHash {
		t.Errorf("unexpected value of specHash: %s, expected: %s", fetched.Status.SpecHash, specHash)
	}
}

// TestInstanceController_Add_Finalizer
// Test adding the finalizer
func TestInstanceController_Add_Finalizer(t *testing.T) {
	// Create mock instance + secret
	i := testingregistriesv1alpha1.CreateInstance(name, ns)
	i.Status.Phase.Name = registriesv1alpha1.InstanceStatusPhaseInstalling
	i.Spec.GarbageCollection = nil

	instanceSecret := testingregistriesv1alpha1.CreateSecret(i.Name+"-harbor-core", ns)

	r := buildReconcileWithFakeClientWithMocks([]runtime.Object{&i, &instanceSecret})

	ctx := context.TODO()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockClient := helmclientmock.NewMockClient(ctrl)
	gomock.InOrder(
		mockClient.EXPECT().UpdateChartRepos(),
		mockClient.EXPECT().InstallOrUpgradeChart(ctx, &helmclient.ChartSpec{
			ReleaseName: i.Spec.HelmChart.ReleaseName,
			ChartName:   i.Spec.HelmChart.ChartName,
			Namespace:   i.Spec.HelmChart.Namespace,
			ValuesYaml:  i.Spec.HelmChart.ValuesYaml,
			Version:     i.Spec.HelmChart.Version,
		}),
	)

	r.helmClientReceiver = func(repoCache, repoConfig, namespace string) (helmclient.Client, error) {
		return helmclient.Client(mockClient), nil
	}

	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      i.Name,
			Namespace: i.Namespace,
		},
	}

	res, err := r.Reconcile(req)
	if err != nil {
		t.Fatalf("reconcile returned error: (%v)", err)
	}

	if !res.Requeue {
		t.Error("reconciliation was not requeued")
	}

	instance := &registriesv1alpha1.Instance{}

	err = r.client.Get(context.TODO(), req.NamespacedName, instance)
	if err != nil {
		t.Error("could not get instance")
	}

	if instance.Finalizers == nil || len(instance.Finalizers) == 0 {
		t.Error("finalizer has not been added")
	}

	if instance.Finalizers[0] != FinalizerName {
		t.Error("finalizer does not contain the expected value")
	}
}

// TestInstanceController_Existing_Finalizer
// Test the finalizer for existence
func TestInstanceController_Existing_Finalizer(t *testing.T) {
	// Create mock instance + secret
	i := testingregistriesv1alpha1.CreateInstance(name, ns)
	i.Status.Phase.Name = registriesv1alpha1.InstanceStatusPhaseInstalling
	i.Spec.GarbageCollection = nil

	instanceSecret := testingregistriesv1alpha1.CreateSecret(i.Name+"-harbor-core", ns)

	r := buildReconcileWithFakeClientWithMocks([]runtime.Object{&i, &instanceSecret})

	ctx := context.TODO()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockClient := helmclientmock.NewMockClient(ctrl)
	gomock.InOrder(
		mockClient.EXPECT().UpdateChartRepos(),
		mockClient.EXPECT().InstallOrUpgradeChart(ctx, &helmclient.ChartSpec{
			ReleaseName: i.Spec.HelmChart.ReleaseName,
			ChartName:   i.Spec.HelmChart.ChartName,
			Namespace:   i.Spec.HelmChart.Namespace,
			ValuesYaml:  i.Spec.HelmChart.ValuesYaml,
			Version:     i.Spec.HelmChart.Version,
		}),
	)

	r.helmClientReceiver = func(repoCache, repoConfig, namespace string) (helmclient.Client, error) {
		return helmclient.Client(mockClient), nil
	}

	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      i.Name,
			Namespace: i.Namespace,
		},
	}

	res, err := r.Reconcile(req)
	if err != nil {
		t.Fatalf("reconcile returned error: (%v)", err)
	}

	if !res.Requeue {
		t.Error("reconciliation was not requeued")
	}

	repository := &registriesv1alpha1.Instance{}
	err = r.client.Get(context.TODO(), req.NamespacedName, repository)

	if err != nil {
		t.Error("could not get instance")
	}

	if repository.Finalizers == nil || len(repository.Finalizers) == 0 {
		t.Error("finalizer has not been added")
	}
}
