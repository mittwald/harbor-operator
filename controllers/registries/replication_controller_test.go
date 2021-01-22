package registries_test

import (
	"github.com/mittwald/harbor-operator/apis/registries/v1alpha2"
	registriestesting "github.com/mittwald/harbor-operator/controllers/registries/testing"
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
		var replication *v1alpha2.Replication
		Context("Replication", func() {
			BeforeEach(func() {
				replication = registriestesting.CreateReplication(name, namespace, "")
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
