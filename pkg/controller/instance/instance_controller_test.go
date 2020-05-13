package instance

import (
	"context"
	"testing"
	"time"

	registriesv1alpha1 "github.com/mittwald/harbor-operator/pkg/apis/registries/v1alpha1"
	testingregistriesv1alpha1 "github.com/mittwald/harbor-operator/pkg/testing/registriesv1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
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
			Name:      "test-instance",
			Namespace: "test-namespace",
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
	name := "test-instance"
	namespace := "test-namespace"
	i := testingregistriesv1alpha1.CreateInstance(name, namespace)

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
	ns := "test-namespace"

	i := testingregistriesv1alpha1.CreateInstance("test-instance", ns)

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

	res, err = r.Reconcile(req)
	if err != nil {
		t.Fatalf("reconcile returned error: (%v)", err)
	}

	instance = &registriesv1alpha1.Instance{}

	err = r.client.Get(context.TODO(), req.NamespacedName, instance)
	if err != nil {
		t.Errorf("could not get instance: %v", err)
	}

	if !res.Requeue {
		t.Error("reconciliation was not requeued")
	}

	if instance.Status.Phase.Name != registriesv1alpha1.InstanceStatusPhaseInstalling {
		t.Errorf("instance status unexpected, status: %s, expected: %s", instance.Status.Phase.Name, registriesv1alpha1.InstanceStatusPhaseInstalling)
	}
}

// TestInstanceController_Instance_Deletion
// Test the deletion of an instance object
func TestInstanceController_Instance_Deletion(t *testing.T) {
	ns := "test-namespace"

	i := testingregistriesv1alpha1.CreateInstance("test-instance", ns)

	instanceSecret := testingregistriesv1alpha1.CreateSecret(i.Name+"-harbor-core", ns)
	i.Status.Phase.Name = registriesv1alpha1.InstanceStatusPhaseReady

	r := buildReconcileWithFakeClientWithMocks([]runtime.Object{&i, &instanceSecret})

	i.DeletionTimestamp = &metav1.Time{Time: time.Now()}

	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      i.Name,
			Namespace: i.Namespace,
		},
	}

	err := r.client.Delete(context.TODO(), &i)
	if err != nil {
		t.Error("could not delete instance")
	}

	res, err := r.Reconcile(req)
	if err != nil {
		t.Fatalf("reconcile returned error: (%v)", err)
	}

	if res.Requeue {
		t.Error("reconciliation was erroneously requeued")
	}

	instance := &registriesv1alpha1.Instance{}
	err = r.client.Get(context.TODO(), req.NamespacedName, instance)
	if err == nil {
		t.Error("instance was not deleted")
	}
}

// TestInstanceController_Add_Finalizer
// Test adding the finalizer
func TestInstanceController_Add_Finalizer(t *testing.T) {
	ns := "test-namespace"

	// Create mock instance + secret
	i := testingregistriesv1alpha1.CreateInstance("test-instance", ns)
	i.Status.Phase.Name = registriesv1alpha1.InstanceStatusPhaseReady

	instanceSecret := testingregistriesv1alpha1.CreateSecret(i.Name+"-harbor-core", ns)

	r := buildReconcileWithFakeClientWithMocks([]runtime.Object{&i, &instanceSecret})

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
	ns := "test-namespace"

	// Create mock instance + secret
	i := testingregistriesv1alpha1.CreateInstance("test-instance", ns)
	i.Status.Phase.Name = registriesv1alpha1.InstanceStatusPhaseReady

	instanceSecret := testingregistriesv1alpha1.CreateSecret(i.Name+"-harbor-core", ns)

	r := buildReconcileWithFakeClientWithMocks([]runtime.Object{&i, &instanceSecret})

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
