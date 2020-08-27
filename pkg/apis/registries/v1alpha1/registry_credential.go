package v1alpha1

import (
	"context"
	"fmt"

	modelv1 "github.com/mittwald/goharbor-client/model/v1_10_0"
	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (creds *RegistryCredential) ToHarborRegistryCredential(ctx context.Context, c client.Client, namespace string) (*modelv1.RegistryCredential, error) {
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

	return &modelv1.RegistryCredential{
		AccessKey:    string(accessKey),
		AccessSecret: string(accessSecret),
		Type:         creds.Type,
	}, nil
}
