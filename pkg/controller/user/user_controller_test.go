package user

import (
	"context"
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

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
func buildReconcileWithFakeClientWithMocks(objs []runtime.Object) *ReconcileUser {
	s := scheme.Scheme

	s.AddKnownTypes(
		registriesv1alpha1.SchemeGroupVersion,
		&registriesv1alpha1.Repository{},
		&registriesv1alpha1.Instance{},
		&registriesv1alpha1.User{},
	)

	// create a fake client to mock API calls with the mock objects
	cl := fake.NewFakeClientWithScheme(s, objs...)

	// create a ReconcileUser object with the scheme and fake client
	return &ReconcileUser{client: cl, scheme: s}
}

func TestUserController_Empty_User_Spec(t *testing.T) {
	ns := "test-namespace"

	instance := testingregistriesv1alpha1.CreateInstance("test-instance", ns)
	instance.Status.Phase.Name = registriesv1alpha1.InstanceStatusPhaseReady

	instanceSecret := testingregistriesv1alpha1.CreateSecret(instance.Name+"-harbor-core", ns)

	u := registriesv1alpha1.User{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-user",
			Namespace: ns,
		},
	}

	r := buildReconcileWithFakeClientWithMocks([]runtime.Object{&u, &instance, &instanceSecret})

	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      u.Name,
			Namespace: u.Namespace,
		},
	}

	res, err := r.Reconcile(req)
	if err != nil {
		t.Fatalf("reconcile returned error: %v", err)
	}

	if !res.Requeue {
		t.Error("reconciliation did not requeue")
	}
}

func TestUserController_Instance_Phase(t *testing.T) {
	u := registriesv1alpha1.User{}
	ns := "test-namespace"

	// Test reconciliation with a non existent instance object which is expected to be requeued
	// Expect: Result without requeue + no error.
	t.Run("NonExistentInstance", func(t *testing.T) {
		r := buildReconcileWithFakeClientWithMocks([]runtime.Object{&u})

		req := reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      u.Name,
				Namespace: ns,
			},
		}

		res, err := r.Reconcile(req)
		if err != nil {
			t.Fatalf("reconcile returned error: (%v)", err)
		}

		if res.Requeue {
			t.Error("reconciliation was requeued")
		}
	})

	i := registriesv1alpha1.Instance{}
	i.Status.Phase.Name = registriesv1alpha1.InstanceStatusPhaseError

	// Test with an Instance in error status in loop.
	// Expect: Result without requeue + no error.
	t.Run("UnreadyInstance", func(t *testing.T) {
		r := buildReconcileWithFakeClientWithMocks([]runtime.Object{&u, &i})

		req := reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      u.Name,
				Namespace: u.Namespace,
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

	t.Run("ExistingInstance", func(t *testing.T) {

		instance := testingregistriesv1alpha1.CreateInstance("test-instance", ns)
		instance.Status.Phase.Name = registriesv1alpha1.InstanceStatusPhaseReady

		u := registriesv1alpha1.User{}

		r := buildReconcileWithFakeClientWithMocks([]runtime.Object{&u, &instance})

		req := reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      u.Name,
				Namespace: u.Namespace,
			},
		}

		res, err := r.Reconcile(req)
		if err == nil {
			t.Fatalf("reconcile did not return an error")
		}

		if !res.Requeue {
			t.Error("reconciliation did not requeue")
		}

	})
}

// TestUserController_User_Deletion
func TestUserController_User_Deletion(t *testing.T) {
	ns := "test-namespace"

	instance := testingregistriesv1alpha1.CreateInstance("test-instance", ns)
	instance.Status.Phase.Name = registriesv1alpha1.InstanceStatusPhaseReady

	u := testingregistriesv1alpha1.CreateUser("test-user", ns)
	u.Spec.ParentInstance.Name = instance.Spec.Name

	r := buildReconcileWithFakeClientWithMocks([]runtime.Object{&u, &instance})

	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      u.Name,
			Namespace: u.Namespace,
		},
	}

	err := r.client.Delete(context.TODO(), &u)
	if err != nil {
		t.Error("could not delete user")
	}

	res, err := r.Reconcile(req)
	if err != nil {
		t.Fatalf("reconcile returned error: (%v)", err)
	}
	if res.Requeue {
		t.Error("reconciliation was erroneously requeued")
	}

	user := &registriesv1alpha1.User{}
	err = r.client.Get(context.TODO(), req.NamespacedName, user)

	if err == nil {
		t.Error("user was not deleted")
	}
}
