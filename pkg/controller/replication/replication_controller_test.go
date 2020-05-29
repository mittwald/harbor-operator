package replication

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
func buildReconcileWithFakeClientWithMocks(objs []runtime.Object) *ReconcileReplication {
	s := scheme.Scheme

	s.AddKnownTypes(
		registriesv1alpha1.SchemeGroupVersion,
		&registriesv1alpha1.Replication{},
		&registriesv1alpha1.Instance{},
		&registriesv1alpha1.Registry{},
	)

	// create a fake client to mock API calls with the mock objects
	cl := fake.NewFakeClientWithScheme(s, objs...)

	// create a ReconcileReplication object with the scheme and fake client
	return &ReconcileReplication{client: cl, scheme: s}
}

func TestReplicationController_Instance_Phase(t *testing.T) {
	rep := registriesv1alpha1.Replication{}

	// Test reconciliation with a non existent instance object
	// Expect: Result without requeue + no error.
	t.Run("NonExistentInstance", func(t *testing.T) {
		r := buildReconcileWithFakeClientWithMocks([]runtime.Object{&rep})
		req := reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      rep.Name,
				Namespace: rep.Namespace,
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
		r := buildReconcileWithFakeClientWithMocks([]runtime.Object{&rep, &i})
		req := reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      rep.Name,
				Namespace: rep.Namespace,
			},
		}
		res, err := r.Reconcile(req)

		if err == nil {
			t.Error("reconciliation did not return error as expected")
		}
		if res.RequeueAfter != 30 * time.Second {
			t.Error("reconciliation did not requeue as expected")
		}
	})
}

// TestReplicationController_Replication_Deletion
func TestReplicationController_Replication_Deletion(t *testing.T) {
	ns := "test-namespace"

	instance := testingregistriesv1alpha1.CreateInstance("test-instance", ns)
	instance.Status.Phase.Name = registriesv1alpha1.InstanceStatusPhaseReady

	repo := testingregistriesv1alpha1.CreateReplication("test-replication", ns, instance.Spec.Name)
	repo.Status.Phase = registriesv1alpha1.ReplicationStatusPhaseReady

	r := buildReconcileWithFakeClientWithMocks([]runtime.Object{&repo, &instance})

	err := r.client.Delete(context.TODO(), &repo)
	if err != nil {
		t.Error("could not delete replication")
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

	replication := &registriesv1alpha1.Replication{}
	err = r.client.Get(context.TODO(), req.NamespacedName, replication)

	if err == nil {
		t.Error("replication was not deleted")
	}
}
