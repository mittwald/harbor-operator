package controllers_test

import (
	registriesv1alpha2 "github.com/mittwald/harbor-operator/api/v1alpha2"
	registriesv1alpha2test "github.com/mittwald/harbor-operator/controllers/testing/registriesv1alpha2"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

var _ = Describe("ReplicationController", func() {
	BeforeEach(func() {
		name = testReplicationName
		namespace = testNamespaceName
		request = ctrl.Request{
			NamespacedName: types.NamespacedName{
				Name:      name,
				Namespace: namespace,
			},
		}
	})
	Describe("Create, Get and Delete", func() {
		var replication *registriesv1alpha2.Replication
		Context("Replication", func() {
			BeforeEach(func() {
				replication = registriesv1alpha2test.CreateReplication(name, namespace, "")
				Ω(k8sClient.Create(ctx, replication)).Should(Succeed())
				Ω(k8sClient.Get(ctx, types.NamespacedName{
					Name:      name,
					Namespace: namespace,
				},
					replication)).Should(Succeed())
			})
			AfterEach(func() {
				Ω(k8sClient.Delete(ctx, replication)).Should(Succeed())
			})
			It("Should not be nil", func() {
				Ω(replication).ToNot(BeNil())
			})
		})
	})
})
