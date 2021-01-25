package helper

import (
	"context"
	"fmt"

	"github.com/mittwald/harbor-operator/apis/registries/v1alpha2"

	legacymodel "github.com/mittwald/goharbor-client/v3/apiv2/model/legacy"
	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func ToHarborRegistryCredential(ctx context.Context, c client.Client, namespace string, regcred v1alpha2.RegistryCredential) (*legacymodel.RegistryCredential, error) {
	var secret v1.Secret
	if err := c.Get(ctx, client.ObjectKey{Name: regcred.SecretRef.Name, Namespace: namespace}, &secret); err != nil {
		return nil, err
	}

	accessKey, exists := secret.Data[regcred.SecretKeyAccessKey]
	if !exists {
		return nil, fmt.Errorf("key %q not found in secret %s/%s", regcred.SecretKeyAccessKey, namespace, regcred.SecretRef.Name)
	}

	accessSecret, exists := secret.Data[regcred.SecretKeyAccessSecret]
	if !exists {
		return nil, fmt.Errorf("key %q not found in secret %s/%s", regcred.SecretKeyAccessSecret, namespace, regcred.SecretRef.Name)
	}

	return &legacymodel.RegistryCredential{
		AccessKey:    string(accessKey),
		AccessSecret: string(accessSecret),
		Type:         regcred.Type,
	}, nil
}
