package registries_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/mittwald/harbor-operator/apis/registries/v1alpha2"
	registriestesting "github.com/mittwald/harbor-operator/controllers/registries/testing"
)

var _ = Describe("InstanceChartRepositoryController", func() {
	BeforeEach(func() {
		name = testInstanceChartRepositoryName
		namespace = testNamespaceName
		request = ctrl.Request{
			NamespacedName: client.ObjectKey{
				Name:      name,
				Namespace: namespace,
			},
		}
	})
	Describe("Create, Get and Delete", func() {
		var instanceChartRepository *v1alpha2.InstanceChartRepository
		Context("InstanceChartRepository", func() {
			BeforeEach(func() {
				instanceChartRepository = registriestesting.CreateInstanceChartRepository(name, namespace)
				Ω(k8sClient.Create(ctx, instanceChartRepository)).Should(Succeed())
				Ω(k8sClient.Get(ctx, client.ObjectKey{
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
