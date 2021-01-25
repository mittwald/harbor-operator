package registries_test

import (
	"github.com/mittwald/harbor-operator/apis/registries/v1alpha2"
	registriestesting "github.com/mittwald/harbor-operator/controllers/registries/testing"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

var _ = Describe("RegistryController", func() {
	BeforeEach(func() {
		name = testRegistryName
		namespace = testNamespaceName
		request = ctrl.Request{
			NamespacedName: types.NamespacedName{
				Name:      name,
				Namespace: namespace,
			},
		}
	})
	Describe("Create, Get and Delete", func() {
		var registry *v1alpha2.Registry
		Context("Registry", func() {
			BeforeEach(func() {
				registry = registriestesting.CreateRegistry(name, namespace, "")
				Ω(k8sClient.Create(ctx, registry)).Should(Succeed())
				Ω(k8sClient.Get(ctx, types.NamespacedName{
					Name:      name,
					Namespace: namespace,
				},
					registry)).Should(Succeed())
			})
			AfterEach(func() {
				Ω(k8sClient.Delete(ctx, registry)).Should(Succeed())
			})
			It("Should not be nil", func() {
				Ω(registry).ToNot(BeNil())
			})
		})
	})
})
