package registries_test

import (
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions"
	"sigs.k8s.io/controller-runtime/pkg/client"

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

			err = envtest.WaitForCRDs(cfg, []client.Object{
				&apiextensions.CustomResourceDefinition{
					Spec: apiextensions.CustomResourceDefinitionSpec{
						Group:    "registries.mittwald.de",
						Names:    apiextensions.CustomResourceDefinitionNames{Plural: "instancechartrepositories"},
						Versions: []apiextensions.CustomResourceDefinitionVersion{{Name: "v1alpha2"}},
					},
				},
				&apiextensions.CustomResourceDefinition{
					Spec: apiextensions.CustomResourceDefinitionSpec{
						Group:    "registries.mittwald.de",
						Names:    apiextensions.CustomResourceDefinitionNames{Plural: "instances"},
						Versions: []apiextensions.CustomResourceDefinitionVersion{{Name: "v1alpha2"}},
					},
				},
				&apiextensions.CustomResourceDefinition{
					Spec: apiextensions.CustomResourceDefinitionSpec{
						Group:    "registries.mittwald.de",
						Names:    apiextensions.CustomResourceDefinitionNames{Plural: "projects"},
						Versions: []apiextensions.CustomResourceDefinitionVersion{{Name: "v1alpha2"}},
					},
				},
				&apiextensions.CustomResourceDefinition{
					Spec: apiextensions.CustomResourceDefinitionSpec{
						Group:    "registries.mittwald.de",
						Names:    apiextensions.CustomResourceDefinitionNames{Plural: "registries"},
						Versions: []apiextensions.CustomResourceDefinitionVersion{{Name: "v1alpha2"}},
					},
				},
				&apiextensions.CustomResourceDefinition{
					Spec: apiextensions.CustomResourceDefinitionSpec{
						Group:    "registries.mittwald.de",
						Names:    apiextensions.CustomResourceDefinitionNames{Plural: "replications"},
						Versions: []apiextensions.CustomResourceDefinitionVersion{{Name: "v1alpha2"}},
					},
				},
				&apiextensions.CustomResourceDefinition{
					Spec: apiextensions.CustomResourceDefinitionSpec{
						Group:    "registries.mittwald.de",
						Names:    apiextensions.CustomResourceDefinitionNames{Plural: "users"},
						Versions: []apiextensions.CustomResourceDefinitionVersion{{Name: "v1alpha2"}},
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
