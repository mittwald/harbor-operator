package registries_test

import (
	"path/filepath"
	"time"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
)

var _ = Describe("CRDInstallation", func() {
	Describe("Install & Delete CRDs", func() {
		It("should install & delete CRDs in cluster context", func() {
			crds, err := envtest.InstallCRDs(cfg, envtest.CRDInstallOptions{
				Paths:              []string{filepath.Join("..", "..", "config", "crd", "bases")},
				ErrorIfPathMissing: true,
			})
			Ω(err).ToNot(HaveOccurred())
			Ω(crds).ToNot(BeNil())

			err = envtest.WaitForCRDs(cfg, []*apiextensionsv1.CustomResourceDefinition{
				{
					Spec: apiextensionsv1.CustomResourceDefinitionSpec{
						Group:    "registries.mittwald.de",
						Names:    apiextensionsv1.CustomResourceDefinitionNames{Plural: "instancechartrepositories"},
						Versions: []apiextensionsv1.CustomResourceDefinitionVersion{{Name: "v1alpha2"}},
					},
				},
				{
					Spec: apiextensionsv1.CustomResourceDefinitionSpec{
						Group:    "registries.mittwald.de",
						Names:    apiextensionsv1.CustomResourceDefinitionNames{Plural: "instances"},
						Versions: []apiextensionsv1.CustomResourceDefinitionVersion{{Name: "v1alpha2"}},
					},
				},
				{
					Spec: apiextensionsv1.CustomResourceDefinitionSpec{
						Group:    "registries.mittwald.de",
						Names:    apiextensionsv1.CustomResourceDefinitionNames{Plural: "projects"},
						Versions: []apiextensionsv1.CustomResourceDefinitionVersion{{Name: "v1alpha2"}},
					},
				},
				{
					Spec: apiextensionsv1.CustomResourceDefinitionSpec{
						Group:    "registries.mittwald.de",
						Names:    apiextensionsv1.CustomResourceDefinitionNames{Plural: "registries"},
						Versions: []apiextensionsv1.CustomResourceDefinitionVersion{{Name: "v1alpha2"}},
					},
				},
				{
					Spec: apiextensionsv1.CustomResourceDefinitionSpec{
						Group:    "registries.mittwald.de",
						Names:    apiextensionsv1.CustomResourceDefinitionNames{Plural: "replications"},
						Versions: []apiextensionsv1.CustomResourceDefinitionVersion{{Name: "v1alpha2"}},
					},
				},
				{
					Spec: apiextensionsv1.CustomResourceDefinitionSpec{
						Group:    "registries.mittwald.de",
						Names:    apiextensionsv1.CustomResourceDefinitionNames{Plural: "users"},
						Versions: []apiextensionsv1.CustomResourceDefinitionVersion{{Name: "v1alpha2"}},
					},
				},
			}, envtest.CRDInstallOptions{
				MaxTime:      5 * time.Second,
				PollInterval: 1 * time.Second,
			})
			Ω(err).ToNot(HaveOccurred())
		})
	})
})
