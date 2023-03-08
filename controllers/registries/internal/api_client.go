package internal

import (
	"context"
	"strings"
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

	// Append "-harbor" to the expected secret name if the release name doesn't already contain it [^1].
	// [1]: https://github.com/goharbor/harbor-helm/commit/db7b7c17c20c6046c031abacc9dda6aef57e90a6#diff-87d68c754766af8e2e930e653be7e4b75fa0c8bdb187cb1bec293f265d9159ff
	secretName := func() string {
		if strings.Contains(harbor.Spec.HelmChart.ReleaseName, "harbor") {
			return harbor.Spec.HelmChart.ReleaseName + "-core"
		}
		return harbor.Spec.HelmChart.ReleaseName + "-harbor-core"
	}

	err := cl.Get(ctx, client.ObjectKey{
		Name:      secretName(),
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
