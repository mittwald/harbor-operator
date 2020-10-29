module github.com/mittwald/harbor-operator/api

go 1.15

require (
	github.com/mittwald/go-helm-client v0.4.1
	k8s.io/api v0.19.3
	k8s.io/apimachinery v0.19.3
	sigs.k8s.io/controller-runtime v0.6.3
)

replace k8s.io/kubectl => k8s.io/kubectl v0.19.3
