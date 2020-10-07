package registriesv1alpha1_test

import (
	registriesv1alpha1 "github.com/mittwald/harbor-operator/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CreateInstanceChartRepo returns an instancechartrepo object with sample values.
func CreateInstanceChartRepository(name, namespace string) *registriesv1alpha1.InstanceChartRepository {
	icr := registriesv1alpha1.InstanceChartRepository{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec:   registriesv1alpha1.InstanceChartRepositorySpec{},
		Status: registriesv1alpha1.InstanceChartRepositoryStatus{},
	}

	return &icr
}
