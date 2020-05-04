package registriesv1alpha1

import (
	registriesv1alpha1 "github.com/mittwald/harbor-operator/pkg/apis/registries/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func CreateInstanceChartRepo(name, namespace string) registriesv1alpha1.InstanceChartRepo {
	icr := registriesv1alpha1.InstanceChartRepo{
		TypeMeta:   metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{},
		Spec:       registriesv1alpha1.InstanceChartRepoSpec{},
		Status:     registriesv1alpha1.InstanceChartRepoStatus{},
	}

	return icr
}
