package controllers_test

import (
	"context"
	registriesv1alpha1 "github.com/mittwald/harbor-operator/api/v1alpha1"
	registriesv1alpha1_test "github.com/mittwald/harbor-operator/controllers/testing/registriesv1alpha1"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	testInstanceName          = "test-instance"
	testInstanceChartRepoName = "test-instancechartrepo"
	testNamespace             = "test-namespace"
)

var (
	name      string
	namespace string
	request   ctrl.Request
)

var _ = Describe("InstanceController", func() {
	ctx := context.TODO()

	BeforeEach(func() {
		name = testInstanceChartRepoName
		namespace = testNamespace
		request = ctrl.Request{
			NamespacedName: types.NamespacedName{
				Name:      testInstanceChartRepoName,
				Namespace: testNamespace,
			},
		}
	})
	Describe("Create and Get", func() {
		var (
			instanceChartRepository *registriesv1alpha1.InstanceChartRepository
		)
		Context("using single InstanceChartRepository resource", func() {
			BeforeEach(func() {
				instanceChartRepository = registriesv1alpha1_test.CreateInstanceChartRepository(testInstanceChartRepoName, testNamespace)
				k8sClient.Create(ctx, instanceChartRepository)
				k8sClient.Get(ctx, types.NamespacedName{Name: testInstanceName, Namespace: testNamespace}, instanceChartRepository)
				Ω(instanceChartRepository).ToNot(BeNil())
			})
			AfterEach(func() {
				k8sClient.Delete(ctx, instanceChartRepository)
			})
			It("should", func() {
				Ω(instanceChartRepository).ToNot(BeNil())
			})
		})
	})
})
