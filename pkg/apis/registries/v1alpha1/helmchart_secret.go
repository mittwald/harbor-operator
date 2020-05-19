package v1alpha1

import (
	"context"
	"fmt"

	"github.com/imdario/mergo"
	"github.com/mittwald/go-helm-client"
	"github.com/mittwald/harbor-operator/pkg/internal/helper"
	"gopkg.in/yaml.v2"

	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (i *Instance) ToChartSpec(ctx context.Context, c client.Client) (*helmclient.ChartSpec, error) {
	err := i.enrichChartWithSecretValues(ctx, c)
	if err != nil {
		return nil, err
	}

	return &i.Spec.HelmChart.ChartSpec, nil
}

func (i *Instance) enrichChartWithSecretValues(ctx context.Context, c client.Client) error {
	if i.Spec.HelmChart.SecretValues == nil {
		return nil
	}

	secret, err := i.getValuesSecret(ctx, c)
	if err != nil {
		return err
	}

	spec := i.Spec.HelmChart

	secretValuesYaml, ok := secret.Data[spec.SecretValues.Key]
	if !ok {
		return fmt.Errorf(
			"secret %q does not have the key %q",
			spec.SecretValues.SecretRef.Name,
			spec.SecretValues.Key,
		)
	}

	var secretValuesMap map[string]interface{}
	err = yaml.Unmarshal(secretValuesYaml, &secretValuesMap)
	if err != nil {
		return err
	}

	valuesMap, err := spec.ChartSpec.GetValuesMap()
	if err != nil {
		return err
	}

	err = mergo.Merge(&valuesMap, secretValuesMap, mergo.WithOverride)
	if err != nil {
		return err
	}

	newValuesYaml, err := yaml.Marshal(&valuesMap)
	if err != nil {
		return err
	}

	spec.ChartSpec.ValuesYaml = string(newValuesYaml)

	return nil
}

func (i *Instance) getValuesSecret(ctx context.Context, c client.Client) (*v1.Secret, error) {
	secName := i.Spec.HelmChart.SecretValues.SecretRef.Name

	var secret v1.Secret
	existing, err := helper.ObjExists(ctx, c,
		secName,
		i.Namespace,
		&secret)
	if err != nil {
		return nil, err
	}
	if !existing {
		return nil, fmt.Errorf("secret %q does not exist", secName)
	}

	return &secret, nil
}
