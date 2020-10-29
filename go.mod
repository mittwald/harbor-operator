module github.com/mittwald/harbor-operator

go 1.15

require (
	github.com/go-logr/logr v0.2.1
	github.com/imdario/mergo v0.3.11
	github.com/jinzhu/copier v0.0.0-20190924061706-b57f9002281a
	github.com/mittwald/go-helm-client v0.4.1
	github.com/mittwald/goharbor-client v1.0.7 // indirect
	github.com/mittwald/goharbor-client/v3 v3.0.3
	github.com/mittwald/harbor-operator/api v0.0.0-00010101000000-000000000000
	github.com/nxadm/tail v1.4.5 // indirect
	github.com/onsi/ginkgo v1.14.2
	github.com/onsi/gomega v1.10.2
	github.com/spf13/pflag v1.0.5
	github.com/spf13/viper v1.4.0
	github.com/stretchr/testify v1.6.1
	helm.sh/helm/v3 v3.4.0
	k8s.io/api v0.19.3
	k8s.io/apiextensions-apiserver v0.19.2
	k8s.io/apimachinery v0.19.3
	k8s.io/client-go v0.19.3
	sigs.k8s.io/controller-runtime v0.6.3
	sigs.k8s.io/yaml v1.2.0
)

replace (
	github.com/go-logr/logr => github.com/go-logr/logr v0.2.1
	github.com/go-logr/zapr => github.com/go-logr/zapr v0.2.0
	github.com/mittwald/go-helm-client => ../go-helm-client
	github.com/mittwald/harbor-operator/api => ./api
	k8s.io/cli-runtime => k8s.io/cli-runtime v0.18.8
	k8s.io/client-go => k8s.io/client-go v0.19.3
	sigs.k8s.io/kustomize => sigs.k8s.io/kustomize v2.0.3+incompatible
)
