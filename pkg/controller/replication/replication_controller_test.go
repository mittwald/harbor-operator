package replication

import (
	"context"
	registriesv1alpha1 "github.com/mittwald/harbor-operator/pkg/apis/registries/v1alpha1"
	testingregistriesv1alpha1 "github.com/mittwald/harbor-operator/pkg/testing/registriesv1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"testing"
	"time"
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

// TestReplicationController_Transition_Creating
// Test reconciliation with a valid instance and replication object. The replication's status is expected to change
func TestReplicationController_Transition_Creating(t *testing.T) {
	ns := "test-namespace"

	instance := testingregistriesv1alpha1.CreateInstance("test-instance", ns)
	instance.Status.Phase.Name = registriesv1alpha1.InstanceStatusPhaseReady

	instanceSecret := testingregistriesv1alpha1.CreateSecret(instance.Name+"-harbor-core", ns)

	rep := testingregistriesv1alpha1.CreateReplication("test-replication", ns, instance.Spec.Name)
	rep.Status.Phase = registriesv1alpha1.ReplicationStatusPhaseUnknown

	r := buildReconcileWithFakeClientWithMocks([]runtime.Object{&rep, &instance, &instanceSecret})

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

	if !res.Requeue {
		t.Error("reconciliation was not requeued")
	}

	replication := &registriesv1alpha1.Replication{}

	err = r.client.Get(context.TODO(), req.NamespacedName, replication)
	if err != nil {
		t.Errorf("could not get replication: %v", err)
	}

	// Reconcile again
	res, err = r.Reconcile(req)
	if err != nil {
		t.Fatalf("reconcile returned error: (%v)", err)
	}

	if !res.Requeue {
		t.Error("reconciliation was not requeued")
	}

	err = r.client.Get(context.TODO(), req.NamespacedName, replication)
	if err != nil {
		t.Errorf("could not get replication: %v", err)
	}

	if replication.Status.Phase != registriesv1alpha1.ReplicationStatusPhaseCreating {
		t.Error("replication status did not change as expected")
	}
}

func TestReplicationController_Instance_Phase(t *testing.T) {
	rep := registriesv1alpha1.Replication{}

	// Test reconciliation with a non existent instance object which is expected to be requeued
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

		if res.RequeueAfter != 30*time.Second {
			t.Error("reconciliation did not requeue as expected")
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
		if res.RequeueAfter != 120*time.Second {
			t.Error("reconciliation did not requeue as expected")
		}
	})
}

// TestReplicationController_Registry_Type
// Test reconciliation with either only 'source registry' set in the replication spec,
// or with only 'destination registry' set in the replication spec
// Also tests ToHarborRegistry() for convenience
func TestReplicationController_Registry_Type(t *testing.T) {
	regName := "test-registry"
	repName := "test-replication"
	instanceName := "test-instance"
	ns := "test-namespace"

	instance := testingregistriesv1alpha1.CreateInstance(instanceName, ns)
	instance.Status.Phase.Name = registriesv1alpha1.InstanceStatusPhaseReady

	instanceSecret := testingregistriesv1alpha1.CreateSecret(instance.Name+"-harbor-core", ns)

	reg := testingregistriesv1alpha1.CreateRegistry(regName, ns, instance.Spec.Name)
	reg.Status.Phase = registriesv1alpha1.RegistryStatusPhaseReady

	t.Run("SourceRegistry", func(t *testing.T) {
		rep := testingregistriesv1alpha1.CreateReplication(repName, ns, instance.Spec.Name)
		rep.Status.Phase = registriesv1alpha1.ReplicationStatusPhaseCreating
		rep.Spec.SrcRegistry = &corev1.LocalObjectReference{
			Name:            regName,
		}

		r := buildReconcileWithFakeClientWithMocks([]runtime.Object{&rep, &instance, &instanceSecret, &reg})

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
		if !res.Requeue {
			t.Error("reconciliation did not requeue as expected")
		}
	})

	t.Run("DestinationRegistry", func(t *testing.T) {
		rep := testingregistriesv1alpha1.CreateReplication(repName, ns, instance.Spec.Name)
		rep.Status.Phase = registriesv1alpha1.ReplicationStatusPhaseCreating
		rep.Spec.DestRegistry = &corev1.LocalObjectReference{
			Name:            regName,
		}

		r := buildReconcileWithFakeClientWithMocks([]runtime.Object{&rep, &instance, &instanceSecret, &reg})

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
		if !res.Requeue {
			t.Error("reconciliation did not requeue as expected")
		}
	})
}

// TestReplicationController_Add_Finalizer
// Test adding the finalizer
func TestReplicationController_Add_Finalizer(t *testing.T) {
	ns := "test-namespace"

	// Create mock instance + secret
	instance := testingregistriesv1alpha1.CreateInstance("test-instance", ns)
	instance.Status.Phase.Name = registriesv1alpha1.InstanceStatusPhaseReady

	instanceSecret := testingregistriesv1alpha1.CreateSecret(instance.Name+"-harbor-core", ns)

	rep := testingregistriesv1alpha1.CreateReplication("test-replication", ns, instance.Spec.Name)
	rep.Status.Phase = registriesv1alpha1.ReplicationStatusPhaseReady

	r := buildReconcileWithFakeClientWithMocks([]runtime.Object{&rep, &instance, &instanceSecret})

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

	if !res.Requeue {
		t.Error("reconciliation was not requeued")
	}

	replication := &registriesv1alpha1.Replication{}
	err = r.client.Get(context.TODO(), req.NamespacedName, replication)

	if err != nil {
		t.Error("could not get replication")
	}

	if replication.Finalizers == nil || len(replication.Finalizers) == 0 {
		t.Error("finalizer has not been added")
	}

	if replication.Finalizers[0] != FinalizerName {
		t.Error("finalizer does not contain the expected value")
	}
}

// TestReplicationController_Existing_Finalizer
// Test the finalizer for existence
func TestReplicationController_Existing_Finalizer(t *testing.T) {
	ns := "test-namespace"

	// Create mock instance + secret
	instance := testingregistriesv1alpha1.CreateInstance("test-instance", ns)
	instance.Status.Phase.Name = registriesv1alpha1.InstanceStatusPhaseReady

	instanceSecret := testingregistriesv1alpha1.CreateSecret(instance.Name+"-harbor-core", ns)

	rep := testingregistriesv1alpha1.CreateReplication("test-replication", ns, instance.Spec.Name)
	rep.Status.Phase = registriesv1alpha1.ReplicationStatusPhaseReady
	rep.Finalizers = []string{"test"}

	r := buildReconcileWithFakeClientWithMocks([]runtime.Object{&rep, &instance, &instanceSecret})

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

	if !res.Requeue {
		t.Error("reconciliation was not requeued")
	}

	replication := &registriesv1alpha1.Replication{}
	err = r.client.Get(context.TODO(), req.NamespacedName, replication)

	if err != nil {
		t.Error("could not get replication")
	}

	if replication.Finalizers == nil || len(replication.Finalizers) == 0 {
		t.Error("finalizer has not been added")
	}
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
