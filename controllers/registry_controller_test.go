package controllers_test

import (
	registriesv1alpha2 "github.com/mittwald/harbor-operator/api/v1alpha2"
	registriesv1alpha2test "github.com/mittwald/harbor-operator/controllers/testing/registriesv1alpha2"
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
		var registry *registriesv1alpha2.Registry
		Context("Registry", func() {
			BeforeEach(func() {
				registry = registriesv1alpha2test.CreateRegistry(name, namespace, "")
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
