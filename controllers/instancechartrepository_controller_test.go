package controllers_test

import (
	registriesv1alpha1 "github.com/mittwald/harbor-operator/api/v1alpha1"
	registriesv1alpha1test "github.com/mittwald/harbor-operator/controllers/testing/registriesv1alpha1"
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
			}}
	})
	Describe("Create, Get and Delete", func() {
		var (
			instanceChartRepository *registriesv1alpha1.InstanceChartRepository
		)
		Context("InstanceChartRepository", func() {
			BeforeEach(func() {
				instanceChartRepository = registriesv1alpha1test.CreateInstanceChartRepository(name, namespace)
				Ω(k8sClient.Create(ctx, instanceChartRepository)).Should(Succeed())
				Ω(k8sClient.Get(ctx, types.NamespacedName{
					Name:      name,
					Namespace: namespace},
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
