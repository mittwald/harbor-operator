package helper

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func PushFinalizer(o metav1.Object, finalizer string) {
	hasFinalizer := false
	finalizers := o.GetFinalizers()

	for _, f := range finalizers {
		if f == finalizer {
			hasFinalizer = true
			break
		}
	}

	if !hasFinalizer {
		newFinalizers := append(finalizers, finalizer)
		o.SetFinalizers(newFinalizers)
	}
}

func PullFinalizer(o metav1.Object, finalizer string) {
	finalizers := o.GetFinalizers()
	newFinalizers := make([]string, 0, len(finalizers))

	if len(finalizers) == 0 {
		return
	}

	for _, f := range finalizers {
		if f != finalizer {
			newFinalizers = append(newFinalizers, f)
		}
	}

	o.SetFinalizers(newFinalizers)
}
