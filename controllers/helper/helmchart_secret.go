package helper

import (
	"context"
	"fmt"

	"github.com/mittwald/harbor-operator/api/v1alpha1"

	"github.com/imdario/mergo"
	helmclient "github.com/mittwald/go-helm-client"
	"sigs.k8s.io/yaml"

	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func InstanceToChartSpec(ctx context.Context, c client.Client, instance *v1alpha1.Instance) (*helmclient.ChartSpec, error) {
	err := enrichChartWithSecretValues(ctx, c, instance)
	if err != nil {
		return nil, err
	}

	return &instance.Spec.HelmChart.ChartSpec, nil
}

func enrichChartWithSecretValues(ctx context.Context, c client.Client, instance *v1alpha1.Instance) error {
	if instance.Spec.HelmChart.SecretValues == nil {
		return nil
	}

	secret, err := getValuesSecret(ctx, c, instance)
	if err != nil {
		return err
	}

	spec := instance.Spec.HelmChart

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

func getValuesSecret(ctx context.Context, c client.Client, instance *v1alpha1.Instance) (*v1.Secret, error) {
	secName := instance.Spec.HelmChart.SecretValues.SecretRef.Name

	var secret v1.Secret

	existing, err := ObjExists(ctx, c,
		secName,
		instance.Namespace,
		&secret)
	if err != nil {
		return nil, err
	}

	if !existing {
		return nil, fmt.Errorf("secret %q does not exist", secName)
	}

	return &secret, nil
}
