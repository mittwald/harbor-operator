package instancechartrepo

import (
	registriesv1alpha1 "github.com/mittwald/harbor-operator/pkg/apis/registries/v1alpha1"
	testingregistriesv1alpha1 "github.com/mittwald/harbor-operator/pkg/testing/registriesv1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"testing"
)

// buildReconcileWithFakeClientWithMocks
// returns a reconcile with fake client, schemes and mock objects
// reference: https://github.com/aerogear/mobile-security-service-operator/blob/e74272a6c7addebdc77b18eeffb5e888b35f4dfd/pkg/controller/mobilesecurityservice/fakeclient_test.go#L14
func buildReconcileWithFakeClientWithMocks(objs []runtime.Object) *ReconcileInstanceChartRepo {
	s := scheme.Scheme

	s.AddKnownTypes(
		registriesv1alpha1.SchemeGroupVersion,
		&registriesv1alpha1.InstanceChartRepo{},
	)

	// create a fake client to mock API calls with the mock objects
	cl := fake.NewFakeClientWithScheme(s, objs...)

	// create a ReconcileInstanceChartRepo object with the scheme and fake client
	return &ReconcileInstanceChartRepo{client: cl, scheme: s}
}

// TestInstanceChartRepoController_Empty_SecretRef
// Test if the defined secret ref in spec is empty
func TestInstanceChartRepoController_Empty_SecretRef(t *testing.T) {
	icr := testingregistriesv1alpha1.CreateInstanceChartRepo("test-instancechartrepo", "test-namespace")

	r := buildReconcileWithFakeClientWithMocks([]runtime.Object{&icr})

	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      icr.Name,
			Namespace: icr.Namespace,
		},
	}

	_, err := r.Reconcile(req)
	if err != nil {
		if err.Error() != "no secret ref defined in spec" {
			t.Errorf("reconcile did not return expected error: (%v)", err)
		}
	}
}
