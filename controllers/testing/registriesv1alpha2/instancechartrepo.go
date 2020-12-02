package registriesv1alpha2_test

import (
	registriesv1alpha2 "github.com/mittwald/harbor-operator/api/v1alpha2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CreateInstanceChartRepo returns an instancechartrepo object with sample values.
func CreateInstanceChartRepository(name, namespace string) *registriesv1alpha2.InstanceChartRepository {
	icr := registriesv1alpha2.InstanceChartRepository{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec:   registriesv1alpha2.InstanceChartRepositorySpec{},
		Status: registriesv1alpha2.InstanceChartRepositoryStatus{},
	}

	return &icr
}
