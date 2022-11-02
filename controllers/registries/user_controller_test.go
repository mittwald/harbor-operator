package registries_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/mittwald/harbor-operator/apis/registries/v1alpha2"
	registriestesting "github.com/mittwald/harbor-operator/controllers/registries/testing"
)

var _ = Describe("UserController", func() {
	BeforeEach(func() {
		name = testUserName
		namespace = testNamespaceName
		request = ctrl.Request{
			NamespacedName: client.ObjectKey{
				Name:      name,
				Namespace: namespace,
			},
		}
	})
	Describe("Create, Get and Delete", func() {
		var user *v1alpha2.User
		Context("User", func() {
			BeforeEach(func() {
				user = registriestesting.CreateUser(name, namespace, "")
				Ω(k8sClient.Create(ctx, user)).Should(Succeed())
				Ω(k8sClient.Get(ctx, client.ObjectKey{
					Name:      name,
					Namespace: namespace,
				},
					user)).Should(Succeed())
			})
			AfterEach(func() {
				Ω(k8sClient.Delete(ctx, user)).Should(Succeed())
			})
			It("Should not be nil", func() {
				Ω(user).ToNot(BeNil())
			})
		})
	})
})
