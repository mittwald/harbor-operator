package internal

import (
	"context"
	"time"

	h "github.com/mittwald/goharbor-client/v5/apiv2"
	clientconfig "github.com/mittwald/goharbor-client/v5/apiv2/pkg/config"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/mittwald/harbor-operator/apis/registries/v1alpha2"
	"github.com/mittwald/harbor-operator/controllers/registries/helper"
)

// BuildClient builds a harbor client to interact with the API
// using the default (admin) credentials of an existing harbor instance.
func BuildClient(ctx context.Context, cl client.Client,
	harbor *v1alpha2.Instance) (*h.RESTClient, error) {
	sec := &corev1.Secret{}

	err := cl.Get(ctx, client.ObjectKey{
		Name:      harbor.Name + "-core",
		Namespace: harbor.Namespace,
	}, sec)
	if err != nil {
		return nil, err
	}

	corePassword, err := helper.GetValueFromSecret(sec, "HARBOR_ADMIN_PASSWORD")
	if err != nil {
		return nil, err
	}

	opts := clientconfig.Options{
		Timeout:  10 * time.Second,
		PageSize: 10,
	}

	return h.NewRESTClientForHost(harbor.Spec.InstanceURL+"/api", "admin", corePassword, &opts)
}
