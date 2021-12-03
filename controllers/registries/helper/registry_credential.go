package helper

import (
	"context"
	"fmt"

	"github.com/mittwald/goharbor-client/v5/apiv2/model"

	"github.com/mittwald/harbor-operator/apis/registries/v1alpha2"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func ToHarborRegistryCredential(ctx context.Context, c client.Client, namespace string, cred v1alpha2.RegistryCredential) (*model.RegistryCredential, error) {
	var secret corev1.Secret
	if err := c.Get(ctx, client.ObjectKey{Name: cred.SecretRef.Name, Namespace: namespace}, &secret); err != nil {
		return nil, err
	}

	accessKey, exists := secret.Data[cred.SecretKeyAccessKey]
	if !exists {
		return nil, fmt.Errorf("key %q not found in secret %s/%s", cred.SecretKeyAccessKey, namespace, cred.SecretRef.Name)
	}

	accessSecret, exists := secret.Data[cred.SecretKeyAccessSecret]
	if !exists {
		return nil, fmt.Errorf("key %q not found in secret %s/%s", cred.SecretKeyAccessSecret, namespace, cred.SecretRef.Name)
	}

	return &model.RegistryCredential{
		AccessKey:    string(accessKey),
		AccessSecret: string(accessSecret),
		Type:         cred.Type,
	}, nil
}
