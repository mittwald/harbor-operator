package registries_test

import (
	"github.com/mittwald/harbor-operator/apis/registries/v1alpha2"
	registriestesting "github.com/mittwald/harbor-operator/controllers/registries/testing"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

var _ = Describe("UserController", func() {
	BeforeEach(func() {
		name = testUserName
		namespace = testNamespaceName
		request = ctrl.Request{
			NamespacedName: types.NamespacedName{
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
				Ω(k8sClient.Get(ctx, types.NamespacedName{
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
