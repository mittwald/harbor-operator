package v1alpha1

import (
	"context"
	"fmt"

	legacymodel "github.com/mittwald/goharbor-client/v2/apiv2/model/legacy"
	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (creds *RegistryCredential) ToHarborRegistryCredential(ctx context.Context, c client.Client, namespace string) (*legacymodel.RegistryCredential, error) {
	var secret v1.Secret
	if err := c.Get(ctx, client.ObjectKey{Name: creds.SecretRef.Name, Namespace: namespace}, &secret); err != nil {
		return nil, err
	}

	accessKey, exists := secret.Data[creds.SecretKeyAccessKey]
	if !exists {
		return nil, fmt.Errorf("key %q not found in secret %s/%s", creds.SecretKeyAccessKey, namespace, creds.SecretRef.Name)
	}

	accessSecret, exists := secret.Data[creds.SecretKeyAccessSecret]
	if !exists {
		return nil, fmt.Errorf("key %q not found in secret %s/%s", creds.SecretKeyAccessSecret, namespace, creds.SecretRef.Name)
	}

	return &legacymodel.RegistryCredential{
		AccessKey:    string(accessKey),
		AccessSecret: string(accessSecret),
		Type:         creds.Type,
	}, nil
}
