package internal

import (
	"context"

	h "github.com/mittwald/goharbor-client"
	registriesv1alpha1 "github.com/mittwald/harbor-operator/pkg/apis/registries/v1alpha1"
	"github.com/mittwald/harbor-operator/pkg/internal/helper"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// BuildClient builds a harbor client to interact with the API
// using the default (admin) credentials of an existing harbor instance.
func BuildClient(ctx context.Context, client client.Client,
	harbor *registriesv1alpha1.Instance) (*h.RESTClient, error) {
	sec := &corev1.Secret{}

	err := client.Get(ctx, types.NamespacedName{
		Name:      harbor.Name + "-harbor-core",
		Namespace: harbor.Namespace,
	}, sec)
	if err != nil {
		return nil, err
	}

	corePassword, err := helper.GetValueFromSecret(sec, "HARBOR_ADMIN_PASSWORD")
	if err != nil {
		return nil, err
	}

	return h.NewRESTClientForHost(harbor.Spec.InstanceHost+"/api", "admin", corePassword)
}
