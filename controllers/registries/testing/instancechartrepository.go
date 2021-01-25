package testing

import (
	"github.com/mittwald/harbor-operator/apis/registries/v1alpha2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CreateInstanceChartRepo returns an instancechartrepo object with sample values.
func CreateInstanceChartRepository(name, namespace string) *v1alpha2.InstanceChartRepository {
	icr := v1alpha2.InstanceChartRepository{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec:   v1alpha2.InstanceChartRepositorySpec{},
		Status: v1alpha2.InstanceChartRepositoryStatus{},
	}

	return &icr
}
