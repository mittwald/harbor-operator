package testing

import (
	helmclient "github.com/mittwald/go-helm-client"
	"github.com/mittwald/harbor-operator/apis/registries/v1alpha2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CreateInstance returns an instance object with sample values.
func CreateInstance(name, namespace string) *v1alpha2.Instance {
	i := v1alpha2.Instance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: v1alpha2.InstanceSpec{
			Name:        name,
			Type:        "manual",
			InstanceURL: "https://core.harbor.domain",
			GarbageCollection: &v1alpha2.GarbageCollection{
				Cron:         "0 * * * *",
				ScheduleType: "Hourly",
			},
			HelmChart: &v1alpha2.InstanceHelmChartSpec{
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
