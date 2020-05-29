package registry

import (
	"context"
	"testing"
	"time"

	registriesv1alpha1 "github.com/mittwald/harbor-operator/pkg/apis/registries/v1alpha1"
	testingregistriesv1alpha1 "github.com/mittwald/harbor-operator/pkg/testing/registriesv1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// buildReconcileWithFakeClientWithMocks
// returns a reconcile with fake client, schemes and mock objects
// reference: https://github.com/aerogear/mobile-security-service-operator/blob/e74272a6c7addebdc77b18eeffb5e888b35f4dfd/pkg/controller/mobilesecurityservice/fakeclient_test.go#L14
func buildReconcileWithFakeClientWithMocks(objs []runtime.Object) *ReconcileRegistry {
	s := scheme.Scheme

	s.AddKnownTypes(
		registriesv1alpha1.SchemeGroupVersion,
		&registriesv1alpha1.Registry{},
		&registriesv1alpha1.Instance{},
	)

	// create a fake client to mock API calls with the mock objects
	cl := fake.NewFakeClientWithScheme(s, objs...)

	// create a ReconcileRegistry object with the scheme and fake client
	return &ReconcileRegistry{client: cl, scheme: s}
}

// TestRegistryController_Instance_Phase
// Test reconciliation with a non existent instance object which is expected to be requeued
// + Test reconciliation with an Instance with error status in loop.
func TestRegistryController_Instance_Phase(t *testing.T) {
	reg := registriesv1alpha1.Registry{}

	// Test reconciliation with a non existent instance object
	// Expect: Result without requeue + no error.
	t.Run("NonExistentInstance", func(t *testing.T) {
		r := buildReconcileWithFakeClientWithMocks([]runtime.Object{&reg})
		req := reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      reg.Name,
				Namespace: reg.Namespace,
			},
		}

		res, err := r.Reconcile(req)
		if err != nil {
			t.Fatalf("reconcile returned error: (%v)", err)
		}

		if res.Requeue {
			t.Error("reconciliation should not be re queued")
		}
	})

	i := registriesv1alpha1.Instance{}
	i.Status.Phase.Name = registriesv1alpha1.InstanceStatusPhaseError

	// Test with an Instance in error status in loop.
	// Expect: Result without requeue + no error.
	t.Run("UnreadyInstance", func(t *testing.T) {
		r := buildReconcileWithFakeClientWithMocks([]runtime.Object{&reg, &i})
		req := reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      reg.Name,
				Namespace: reg.Namespace,
			},
		}
		res, err := r.Reconcile(req)

		if err == nil {
			t.Error("reconciliation did not return error as expected")
		}
		if res.RequeueAfter != 30*time.Second {
			t.Error("reconciliation did not requeue as expected")
		}
	})
}

// TestRegistryController_Registry_Deletion
func TestRegistryController_Registry_Deletion(t *testing.T) {
	ns := "test-namespace"

	instance := testingregistriesv1alpha1.CreateInstance("test-instance", ns)
	instance.Status.Phase.Name = registriesv1alpha1.InstanceStatusPhaseReady

	repo := testingregistriesv1alpha1.CreateRegistry("test-registry", ns, instance.Spec.Name)
	repo.Status.Phase = registriesv1alpha1.RegistryStatusPhaseReady

	r := buildReconcileWithFakeClientWithMocks([]runtime.Object{&repo, &instance})

	err := r.client.Delete(context.TODO(), &repo)
	if err != nil {
		t.Error("could not delete registry")
	}
	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      repo.Name,
			Namespace: repo.Namespace,
		},
	}

	res, err := r.Reconcile(req)
	if err != nil {
		t.Fatalf("reconcile returned error: (%v)", err)
	}
	if res.Requeue {
		t.Error("reconciliation was erroneously requeued")
	}

	registry := &registriesv1alpha1.Registry{}
	err = r.client.Get(context.TODO(), req.NamespacedName, registry)

	if err == nil {
		t.Error("registry was not deleted")
	}
}
