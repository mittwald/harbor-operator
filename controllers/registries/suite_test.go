/*


Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package registries_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/mittwald/harbor-operator/apis/registries/v1alpha2"

	ctrl "sigs.k8s.io/controller-runtime"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/envtest/printer"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	registriesv1alpha2 "github.com/mittwald/harbor-operator/apis/registries/v1alpha2"
	// +kubebuilder:scaffold:imports
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var (
	k8sClient                       client.Client
	testEnv                         *envtest.Environment
	cfg                             *rest.Config
	name                            string
	namespace                       string
	request                         ctrl.Request
	testNamespace                   = &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: testNamespaceName}}
	testInstanceChartRepositoryName = "test-instancechartrepo"
	testInstanceName                = "test-instance"
	testProjectName                 = "test-project"
	testRegistryName                = "test-registry"
	testUserName                    = "test-user"
	testReplicationName             = "test-replication"
	testNamespaceName               = "test-namespace"
	ctx                             = context.TODO()
)

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecsWithDefaultAndCustomReporters(t,
		"Controller Suite",
		[]Reporter{printer.NewlineReporter{}})
}

var _ = BeforeSuite(func(done Done) {
	logf.SetLogger(zap.LoggerTo(GinkgoWriter, true))

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths: []string{filepath.Join("..", "..", "config", "crd", "bases")},
	}

	var err error
	cfg, err = testEnv.Start()
	Ω(err).ToNot(HaveOccurred())
	Ω(cfg).ToNot(BeNil())

	// Add the apiextensions API to operate on CRDs
	err = apiextensions.AddToScheme(scheme.Scheme)
	Ω(err).ToNot(HaveOccurred())

	err = v1alpha2.AddToScheme(scheme.Scheme)
	Ω(err).ToNot(HaveOccurred())

	err = registriesv1alpha2.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	// +kubebuilder:scaffold:scheme
	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Ω(err).ToNot(HaveOccurred())
	Ω(k8sClient).ToNot(BeNil())

	// Create namespace for tests
	err = k8sClient.Create(ctx, testNamespace)
	Ω(err).ToNot(HaveOccurred())

	close(done)
}, 60)

var _ = AfterSuite(func() {
	// Clean up CRDs
	Context("Deleting CRDs", func() {
		err := envtest.UninstallCRDs(cfg, envtest.CRDInstallOptions{
			Paths:        []string{filepath.Join("..", "..", "config", "crd", "bases")},
			MaxTime:      50 * time.Millisecond,
			PollInterval: 15 * time.Millisecond,
		})
		Ω(err).ToNot(HaveOccurred())
	})

	By("tearing down the test environment")
	err := testEnv.Stop()
	Ω(err).ToNot(HaveOccurred())
})
