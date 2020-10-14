package registriesv1alpha1_test

import (
	helmclient "github.com/mittwald/go-helm-client"
	registriesv1alpha1 "github.com/mittwald/harbor-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CreateInstance returns an instance object with sample values.
func CreateInstance(name, namespace string) *registriesv1alpha1.Instance {
	i := registriesv1alpha1.Instance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: registriesv1alpha1.InstanceSpec{
			Name:        name,
			Version:     "v1.0.0",
			Type:        "manual",
			InstanceURL: "https://core.harbor.domain",
			GarbageCollection: &registriesv1alpha1.GarbageCollection{
				Cron:         "0 * * * *",
				ScheduleType: "Hourly",
			},
			Options: &registriesv1alpha1.InstanceDeployOptions{
				KubeconfigSecret: &registriesv1alpha1.KubeconfigSecret{
					SecretRef: &corev1.LocalObjectReference{},
				}},
			HelmChart: &registriesv1alpha1.InstanceHelmChartSpec{
				ChartSpec: helmclient.ChartSpec{
					ReleaseName: name,
					ChartName:   "harbor/harbor",
					Namespace:   namespace,
					ValuesYaml:  "",
					Version:     "v1.0.0",
				},
				SecretValues: nil,
			},
		},
	}

	return &i
}

// CreateSecret returns an instance secret with sample values.
func CreateSecret(name, namespace string) corev1.Secret {
	sec := corev1.Secret{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Data: map[string][]byte{
			"HARBOR_ADMIN_PASSWORD": []byte("test"),
		},
	}

	return sec
}
