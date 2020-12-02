package controllers_test

import (
	registriesv1alpha2 "github.com/mittwald/harbor-operator/api/v1alpha2"
	registriesv1alpha2test "github.com/mittwald/harbor-operator/controllers/testing/registriesv1alpha2"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

var _ = Describe("InstanceChartRepositoryController", func() {
	BeforeEach(func() {
		name = testInstanceChartRepositoryName
		namespace = testNamespaceName
		request = ctrl.Request{
			NamespacedName: types.NamespacedName{
				Name:      name,
				Namespace: namespace,
			},
		}
	})
	Describe("Create, Get and Delete", func() {
		var instanceChartRepository *registriesv1alpha2.InstanceChartRepository
		Context("InstanceChartRepository", func() {
			BeforeEach(func() {
				instanceChartRepository = registriesv1alpha2test.CreateInstanceChartRepository(name, namespace)
				Ω(k8sClient.Create(ctx, instanceChartRepository)).Should(Succeed())
				Ω(k8sClient.Get(ctx, types.NamespacedName{
					Name:      name,
					Namespace: namespace,
				},
					instanceChartRepository)).Should(Succeed())
			})
			AfterEach(func() {
				Ω(k8sClient.Delete(ctx, instanceChartRepository)).Should(Succeed())
			})
			It("Should not be nil", func() {
				Ω(instanceChartRepository).ToNot(BeNil())
			})
		})
	})
})
