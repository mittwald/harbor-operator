package instancechartrepo

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/golang/mock/gomock"
	helmclient "github.com/mittwald/go-helm-client"
	helmclientmock "github.com/mittwald/go-helm-client/mock"
	registriesv1alpha1 "github.com/mittwald/harbor-operator/pkg/apis/registries/v1alpha1"
	testingregistriesv1alpha1 "github.com/mittwald/harbor-operator/pkg/testing/registriesv1alpha1"
	"helm.sh/helm/v3/pkg/repo"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	testRepoName       = "testrepo"
	testNamespace      = "default"
	testSecretName     = "testsecret"
	testURL            = "http://foo.bar"
	testError          = "testerror"
	testSecretUsername = "testuser"
	testSecretPassword = "testpass"
	testSecretCertFile = "testcertfile"
	testSecretKeyFile  = "testkeyfile"
	testSecretCaFile   = "testcafile"
)

// newTestReconciler returns a reconcile with fake client, schemes and mock objects
func newTestReconciler(objs []runtime.Object) *ReconcileInstanceChartRepo {
	s := scheme.Scheme

	s.AddKnownTypes(
		registriesv1alpha1.SchemeGroupVersion,
		&registriesv1alpha1.Instance{},
		&registriesv1alpha1.InstanceChartRepo{},
	)

	// create a fake client to mock API calls with the mock objects
	cl := fake.NewFakeClientWithScheme(s, objs...)

	// create a ReconcileInstance object with the scheme and fake client
	return &ReconcileInstanceChartRepo{client: cl, scheme: s}
}

// TestReconcileInstanceChartRepo_getSecret
// tests if a secret object can be retrieved by a given object reference.
func TestReconcileInstanceChartRepo_getSecret(t *testing.T) {
	ctx := context.Background()
	cr := testingregistriesv1alpha1.CreateInstanceChartRepo(testRepoName, testNamespace)
	crSecret := testingregistriesv1alpha1.CreateSecret(testSecretName, testNamespace)
	cr.Spec.SecretRef = &corev1.LocalObjectReference{Name: testSecretName}
	r := newTestReconciler([]runtime.Object{&crSecret})

	secret, err := r.getSecret(ctx, &cr)
	if err != nil {
		t.Fatalf("got error while fetching secret: %v", err)
	}

	if secret == nil {
		t.Fatalf("secret is nil, expected a value")
	}
	if !reflect.DeepEqual(crSecret.ObjectMeta, secret.ObjectMeta) ||
		!reflect.DeepEqual(crSecret.Data, secret.Data) {
		t.Error("fetched secret is not equal to input secret")
	}
}

// TestReconcileInstanceChartRepo_getSecret_Missing
// tests if a missing secret returns a proper error.
func TestReconcileInstanceChartRepo_getSecret_Missing(t *testing.T) {
	r := newTestReconciler([]runtime.Object{})
	ctx := context.Background()
	cr := testingregistriesv1alpha1.CreateInstanceChartRepo(testRepoName, testNamespace)
	cr.Spec.SecretRef = &corev1.LocalObjectReference{Name: testSecretName}

	secret, err := r.getSecret(ctx, &cr)
	if err == nil {
		t.Fatalf("expected error, but got none")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("Expected keyword 'not found' in error message, got: %v", err)
	}
	if secret != nil {
		t.Errorf("secret should be nil")
	}

}

// TestReconcileInstanceChartRepo_specToRepoEntry
// tests if a helm repo spec can be generated out of a InstanceChartRepo object.
func TestReconcileInstanceChartRepo_specToRepoEntry(t *testing.T) {
	ctx := context.Background()
	cr := testingregistriesv1alpha1.CreateInstanceChartRepo(testRepoName, testNamespace)
	cr.Spec.SecretRef = &corev1.LocalObjectReference{Name: testSecretName}
	cr.Spec.URL = testURL
	crSecret := testingregistriesv1alpha1.CreateSecret(testSecretName, testNamespace)
	crSecret.Data["username"] = []byte(testSecretUsername)
	crSecret.Data["password"] = []byte(testSecretPassword)
	crSecret.Data["certFile"] = []byte(testSecretCertFile)
	crSecret.Data["keyFile"] = []byte(testSecretKeyFile)
	crSecret.Data["caFile"] = []byte(testSecretCaFile)

	r := newTestReconciler([]runtime.Object{&crSecret})

	repo, err := r.specToRepoEntry(ctx, &cr)
	if err != nil {
		t.Fatalf("got error: %v", err)
	}

	if repo == nil {
		t.Fatal("repo is nil")
	}

	if repo.Name != testRepoName {
		t.Errorf("unexpected name, expected: %s, got: %s", testRepoName, repo.Name)
	}
	if repo.URL != testURL {
		t.Errorf("unexpected url, expected: %s, got: %s", testURL, repo.URL)
	}
	if repo.Username != testSecretUsername {
		t.Errorf("unexpected url, expected: %s, got: %s", testSecretUsername, repo.Username)
	}
	if repo.Password != testSecretPassword {
		t.Errorf("unexpected password, expected: %s, got: %s", testSecretPassword, repo.Password)
	}
	if repo.CertFile != testSecretCertFile {
		t.Errorf("unexpected cert file, expected: %s, got: %s", testSecretCertFile, repo.CertFile)
	}
	if repo.KeyFile != testSecretKeyFile {
		t.Errorf("unexpected key file, expected: %s, got: %s", testSecretKeyFile, repo.KeyFile)
	}
	if repo.CAFile != testSecretCaFile {
		t.Errorf("unexpected ca file, expected: %s, got: %s", testSecretCaFile, repo.CAFile)
	}
}

// TestReconcileInstanceChartRepo_specToRepoEntry_MissingSecret
// tests if a proper error is returned while trying to generate a helm repo spec.
func TestReconcileInstanceChartRepo_specToRepoEntry_MissingSecret(t *testing.T) {
	ctx := context.Background()
	cr := testingregistriesv1alpha1.CreateInstanceChartRepo(testRepoName, testNamespace)
	cr.Spec.SecretRef = &corev1.LocalObjectReference{Name: testSecretName}
	cr.Spec.URL = testURL

	r := newTestReconciler([]runtime.Object{})

	repo, err := r.specToRepoEntry(ctx, &cr)
	if err == nil {
		t.Fatalf("expected error, but got none")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("Expected keyword 'not found' in error message, got: %v", err)
	}
	if repo != nil {
		t.Fatal("repo must be nil")
	}
}

// TestReconcileInstanceChartRepo_specToRepoEntry_NilCR
// tests if a nil InstanceChartRepo is properly handled by specToRepoEntry().
func TestReconcileInstanceChartRepo_specToRepoEntry_NilCR(t *testing.T) {
	ctx := context.Background()
	r := newTestReconciler([]runtime.Object{})
	repo, err := r.specToRepoEntry(ctx, nil)
	if err == nil {
		t.Fatalf("expected error, but got none")
	}
	if !strings.Contains(err.Error(), "no instance chart") {
		t.Errorf("Expected keyword 'not found' in error message, got: %v", err)
	}
	if repo != nil {
		t.Fatal("repo must be nil")
	}
}

// TestReconcileInstanceChartRepo_specToRepoEntry_NameOverwrite
// tests if a cr.Spec.Name overwrites the repochart.Name, when set.
func TestReconcileInstanceChartRepo_specToRepoEntry_NameOverwrite(t *testing.T) {
	overwriteName := "something-else"
	ctx := context.Background()
	cr := testingregistriesv1alpha1.CreateInstanceChartRepo(testRepoName, testNamespace)
	cr.Spec.SecretRef = &corev1.LocalObjectReference{Name: testSecretName}
	cr.Spec.Name = overwriteName
	cr.Spec.URL = testURL
	crSecret := testingregistriesv1alpha1.CreateSecret(testSecretName, testNamespace)

	r := newTestReconciler([]runtime.Object{&crSecret})

	repo, err := r.specToRepoEntry(ctx, &cr)
	if err != nil {
		t.Fatalf("got error: %v", err)
	}

	if repo == nil {
		t.Fatal("repo is nil")
	}

	if repo.Name != overwriteName {
		t.Errorf("unexpected name, expected: %s, got: %s", overwriteName, repo.Name)
	}
}

// TestReconcileInstanceChartRepo_specToRepoEntry_setErrStatus
// tests if status and error on a InstanceChartRepo is properly set.
func TestReconcileInstanceChartRepo_specToRepoEntry_setErrStatus(t *testing.T) {
	ctx := context.Background()
	cr := testingregistriesv1alpha1.CreateInstanceChartRepo(testRepoName, testNamespace)
	cr.Spec.SecretRef = &corev1.LocalObjectReference{Name: testSecretName}
	cr.Spec.URL = testURL
	err := errors.New(testError)

	r := newTestReconciler([]runtime.Object{&cr})
	res, rErr := r.setErrStatus(ctx, &cr, err)
	if err != rErr {
		t.Error("received error did not match input error")
	}
	if res.Requeue {
		t.Error("object got requeued, but must not")
	}

	fetched := &registriesv1alpha1.InstanceChartRepo{}
	err = r.client.Get(ctx, types.NamespacedName{Namespace: testNamespace, Name: testRepoName}, fetched)
	if err != nil {
		t.Fatalf("no instance object found in loop")
	}

	if fetched.Status.State != registriesv1alpha1.RepoStateError {
		t.Errorf("unexpected state, expected: %s, got: %s", registriesv1alpha1.RepoStateError, fetched.Status.State)
	}
}

// TestReconcileInstanceChartRepo_specToRepoEntry_setErrStatus_NotRegistered
// tests if setErrStatus() can properly handle a object, which is not in the reconcilation loop.
func TestReconcileInstanceChartRepo_specToRepoEntry_setErrStatus_NotRegistered(t *testing.T) {
	ctx := context.Background()
	cr := testingregistriesv1alpha1.CreateInstanceChartRepo(testRepoName, testNamespace)
	cr.Spec.SecretRef = &corev1.LocalObjectReference{Name: testSecretName}
	cr.Spec.URL = testURL
	err := errors.New(testError)

	r := newTestReconciler([]runtime.Object{})
	_, rErr := r.setErrStatus(ctx, &cr, err)
	if rErr == nil {
		t.Fatal("expected error, but got none")
	}
	if !strings.Contains(rErr.Error(), "not found") {
		t.Errorf("Expected keyword 'not found' in error message, got: %v", rErr)
	}
}

// TestReconcileInstanceChartRepo_Reconcile
// tests if a valid InstanceChartRepo in reconcilation loop is properly handled.
func TestReconcileInstanceChartRepo_Reconcile(t *testing.T) {
	cr := testingregistriesv1alpha1.CreateInstanceChartRepo(testRepoName, testNamespace)
	cr.Spec.SecretRef = &corev1.LocalObjectReference{Name: testSecretName}
	cr.Spec.URL = testURL
	crSecret := testingregistriesv1alpha1.CreateSecret(testSecretName, testNamespace)

	r := newTestReconciler([]runtime.Object{&cr, &crSecret})
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockClient := helmclientmock.NewMockClient(ctrl)
	mockClient.EXPECT().AddOrUpdateChartRepo(repo.Entry{
		Name: testRepoName,
		URL:  testURL,
	})
	r.helmClientReceiver = func(repoCache, repoConfig, namespace string) (helmclient.Client, error) {
		return helmclient.Client(mockClient), nil
	}

	req := reconcile.Request{NamespacedName: types.NamespacedName{
		Namespace: testNamespace,
		Name:      testRepoName,
	}}
	res, err := r.Reconcile(req)
	if err != nil {
		t.Fatalf("error while reconciling: %v", err)
	}
	if res.Requeue {
		t.Error("object got requeued")
	}

	ctx := context.Background()
	fetched := &registriesv1alpha1.InstanceChartRepo{}
	err = r.client.Get(ctx, types.NamespacedName{Namespace: testNamespace, Name: testRepoName}, fetched)
	if err != nil {
		t.Fatalf("no instance object found in loop")
	}

	if fetched.Status.State != registriesv1alpha1.RepoStateReady {
		t.Errorf("unexpected state, expected: %s, got: %s", registriesv1alpha1.RepoStateReady, fetched.Status.State)
	}
}

// TestReconcileInstanceChartRepo_Reconcile_MissingSecret
// tests if the reconciler is able to return a proper error, when a secret to a InstanceChartRepo is missing.
func TestReconcileInstanceChartRepo_Reconcile_MissingSecret(t *testing.T) {
	cr := testingregistriesv1alpha1.CreateInstanceChartRepo(testRepoName, testNamespace)
	cr.Spec.SecretRef = &corev1.LocalObjectReference{Name: testSecretName}
	cr.Spec.URL = testURL

	r := newTestReconciler([]runtime.Object{&cr})
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockClient := helmclientmock.NewMockClient(ctrl)
	r.helmClientReceiver = func(repoCache, repoConfig, namespace string) (helmclient.Client, error) {
		return helmclient.Client(mockClient), nil
	}

	req := reconcile.Request{NamespacedName: types.NamespacedName{
		Namespace: testNamespace,
		Name:      testRepoName,
	}}
	res, err := r.Reconcile(req)
	if err == nil {
		t.Fatalf("expected error, but got none")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("Expected keyword 'not found' in error message, got: %v", err)
	}
	if res.Requeue {
		t.Error("object got requeued")
	}

	ctx := context.Background()
	fetched := &registriesv1alpha1.InstanceChartRepo{}
	err = r.client.Get(ctx, types.NamespacedName{Namespace: testNamespace, Name: testRepoName}, fetched)
	if err != nil {
		t.Fatalf("no instance object found in loop")
	}

	if fetched.Status.State != registriesv1alpha1.RepoStateError {
		t.Errorf("unexpected state, expected: %s, got: %s", registriesv1alpha1.RepoStateError, fetched.Status.State)
	}
}
