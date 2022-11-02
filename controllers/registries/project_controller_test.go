package registries_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/mittwald/harbor-operator/apis/registries/v1alpha2"
	registriestesting "github.com/mittwald/harbor-operator/controllers/registries/testing"
)

var _ = Describe("ProjectController", func() {
	BeforeEach(func() {
		name = testProjectName
		namespace = testNamespaceName
		request = ctrl.Request{
			NamespacedName: client.ObjectKey{
				Name:      name,
				Namespace: namespace,
			},
		}
	})
	Describe("Create, Get and Delete", func() {
		var project *v1alpha2.Project
		Context("Project", func() {
			BeforeEach(func() {
				project = registriestesting.CreateProject(name, namespace, "")
				Ω(k8sClient.Create(ctx, project)).Should(Succeed())
				Ω(k8sClient.Get(ctx, client.ObjectKey{
					Name:      name,
					Namespace: namespace,
				},
					project)).Should(Succeed())
			})
			AfterEach(func() {
				Ω(k8sClient.Delete(ctx, project)).Should(Succeed())
			})
			It("Should not be nil", func() {
				Ω(project).ToNot(BeNil())
			})
		})
	})
})
