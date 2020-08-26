package repository

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
func buildReconcileWithFakeClientWithMocks(objs []runtime.Object) *ReconcileRepository {
	s := scheme.Scheme

	s.AddKnownTypes(
		registriesv1alpha1.SchemeGroupVersion,
		&registriesv1alpha1.Repository{},
		&registriesv1alpha1.Instance{},
		&registriesv1alpha1.User{},
	)

	// create a fake client to mock API calls with the mock objects
	cl := fake.NewFakeClientWithScheme(s, objs...)

	// create a ReconcileRepository object with the scheme and fake client
	return &ReconcileRepository{client: cl, scheme: s}
}

// TestRepositoryController_NonExistent_Instance
// Test reconciliation with a non existent instance object
func TestRepositoryController_NonExistent_Instance(t *testing.T) {
	repo := registriesv1alpha1.Repository{}

	r := buildReconcileWithFakeClientWithMocks([]runtime.Object{&repo})

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
		t.Error("reconciliation should not be re queued")
	}
}

func TestRepositoryController_Unready_Instance(t *testing.T) {
	repo := registriesv1alpha1.Repository{}

	i := registriesv1alpha1.Instance{}
	i.Status.Phase.Name = registriesv1alpha1.InstanceStatusPhaseError

	r := buildReconcileWithFakeClientWithMocks([]runtime.Object{&repo, &i})

	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      repo.Name,
			Namespace: repo.Namespace,
		},
	}

	res, err := r.Reconcile(req)
	if err == nil {
		t.Error("reconciliation did not return error as expected")
	}

	if !(res.RequeueAfter == 30*time.Second) {
		t.Error("reconciliation did not requeue as expected")
	}
}

// TestRepositoryController_Existing_Instance
// Test reconciliation with an existing instance that misses it's core secret
func TestRepositoryController_Existing_Instance(t *testing.T) {
	ns := "test-namespace"

	// Create mock instance
	instance := testingregistriesv1alpha1.CreateInstance("test-instance", ns)
	instance.Status.Phase.Name = registriesv1alpha1.InstanceStatusPhaseInstalled

	repo := registriesv1alpha1.Repository{}

	r := buildReconcileWithFakeClientWithMocks([]runtime.Object{&repo, &instance})

	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      repo.Name,
			Namespace: repo.Namespace,
		},
	}

	res, err := r.Reconcile(req)
	if err == nil {
		t.Fatalf("reconcile did not return an error")
	}

	if !res.Requeue {
		t.Error("reconciliation did not requeue")
	}
}

// TestRepositoryController_Repository_Deletion
func TestRepositoryController_Repository_Deletion(t *testing.T) {
	ns := "test-namespace"

	instance := testingregistriesv1alpha1.CreateInstance("test-instance", ns)
	instance.Status.Phase.Name = registriesv1alpha1.InstanceStatusPhaseInstalled

	user := testingregistriesv1alpha1.CreateUser("test-user", ns)
	user.Spec.ParentInstance.Name = instance.Spec.Name

	repo := testingregistriesv1alpha1.CreateRepository("test-repository", ns, instance.Spec.Name)
	repo.Status.Phase = registriesv1alpha1.RepositoryStatusPhaseReady

	r := buildReconcileWithFakeClientWithMocks([]runtime.Object{&repo, &instance, &user})

	err := r.client.Delete(context.TODO(), &repo)
	if err != nil {
		t.Error("could not delete repository")
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

	repository := &registriesv1alpha1.Repository{}
	err = r.client.Get(context.TODO(), req.NamespacedName, repository)

	if err == nil {
		t.Error("repository was not deleted")
	}
}
