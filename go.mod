module github.com/mittwald/harbor-operator

go 1.16

require (
	github.com/go-logr/logr v0.4.0
	github.com/imdario/mergo v0.3.12
	github.com/jinzhu/copier v0.3.0
	github.com/mittwald/go-helm-client v0.5.0
	github.com/mittwald/goharbor-client/v3 v3.3.0
	github.com/onsi/ginkgo v1.16.2
	github.com/onsi/gomega v1.12.0
	github.com/spf13/pflag v1.0.5
	github.com/spf13/viper v1.7.1
	github.com/stretchr/testify v1.7.0
	helm.sh/helm/v3 v3.5.1
	k8s.io/api v0.20.4
	k8s.io/apiextensions-apiserver v0.20.2
	k8s.io/apimachinery v0.20.4
	k8s.io/client-go v0.20.4
	sigs.k8s.io/controller-runtime v0.8.3
	sigs.k8s.io/yaml v1.2.0
)
