package registries_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/mittwald/harbor-operator/apis/registries/v1alpha2"
	registriestesting "github.com/mittwald/harbor-operator/controllers/registries/testing"
)

var _ = Describe("InstanceController", func() {
	BeforeEach(func() {
		name = testInstanceName
		namespace = testNamespaceName
		request = ctrl.Request{
			NamespacedName: client.ObjectKey{
				Name:      name,
				Namespace: namespace,
			},
		}
	})
	Describe("Create, Get and Delete", func() {
		var instance *v1alpha2.Instance
		Context("Instance", func() {
			BeforeEach(func() {
				instance = registriestesting.CreateInstance(name, namespace)
				Ω(k8sClient.Create(ctx, instance)).Should(Succeed())
				Ω(k8sClient.Get(ctx, client.ObjectKey{
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
