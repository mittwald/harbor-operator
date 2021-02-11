module github.com/mittwald/harbor-operator

go 1.15

require (
	github.com/go-logr/logr v0.3.0
	github.com/imdario/mergo v0.3.11
	github.com/jinzhu/copier v0.0.0-20190924061706-b57f9002281a
	github.com/mittwald/go-helm-client v0.4.2
	github.com/mittwald/goharbor-client/v3 v3.2.0
	github.com/onsi/ginkgo v1.14.2
	github.com/onsi/gomega v1.10.2
	github.com/spf13/pflag v1.0.5
	github.com/spf13/viper v1.7.0
	github.com/stretchr/testify v1.6.1
	github.com/vektra/mockery/v2 v2.6.0 // indirect
	helm.sh/helm/v3 v3.5.1
	k8s.io/api v0.20.2
	k8s.io/apiextensions-apiserver v0.20.1
	k8s.io/apimachinery v0.20.2
	k8s.io/client-go v0.20.2
	sigs.k8s.io/controller-runtime v0.8.1
	sigs.k8s.io/yaml v1.2.0
)
