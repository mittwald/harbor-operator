package controllers_test

import (
	registriesv1alpha2 "github.com/mittwald/harbor-operator/api/v1alpha2"
	registriesv1alpha2test "github.com/mittwald/harbor-operator/controllers/testing/registriesv1alpha2"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

var _ = Describe("InstanceController", func() {
	BeforeEach(func() {
		name = testInstanceName
		namespace = testNamespaceName
		request = ctrl.Request{
			NamespacedName: types.NamespacedName{
				Name:      name,
				Namespace: namespace,
			},
		}
	})
	Describe("Create, Get and Delete", func() {
		var instance *registriesv1alpha2.Instance
		Context("Instance", func() {
			BeforeEach(func() {
				instance = registriesv1alpha2test.CreateInstance(name, namespace)
				Ω(k8sClient.Create(ctx, instance)).Should(Succeed())
				Ω(k8sClient.Get(ctx, types.NamespacedName{
					Name:      name,
					Namespace: namespace,
				},
					instance)).Should(Succeed())
			})
			AfterEach(func() {
				Ω(k8sClient.Delete(ctx, instance)).Should(Succeed())
			})
			It("Should not be nil", func() {
				Ω(instance).ToNot(BeNil())
			})
		})
	})
})
