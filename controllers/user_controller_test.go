package controllers_test

import (
	registriesv1alpha2 "github.com/mittwald/harbor-operator/api/v1alpha2"
	registriesv1alpha2test "github.com/mittwald/harbor-operator/controllers/testing/registriesv1alpha2"
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
		var user *registriesv1alpha2.User
		Context("User", func() {
			BeforeEach(func() {
				user = registriesv1alpha2test.CreateUser(name, namespace, "")
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
