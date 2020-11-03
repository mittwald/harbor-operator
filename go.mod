module github.com/mittwald/harbor-operator

go 1.15

require (
	github.com/go-logr/logr v0.1.0
	github.com/imdario/mergo v0.3.9
	github.com/jinzhu/copier v0.0.0-20190924061706-b57f9002281a
	github.com/mittwald/go-helm-client v0.4.1
	github.com/mittwald/goharbor-client/v3 v3.0.4
	github.com/onsi/ginkgo v1.14.2
	github.com/onsi/gomega v1.10.2
	github.com/spf13/pflag v1.0.5
	github.com/spf13/viper v1.4.0
	github.com/stretchr/testify v1.6.1
	helm.sh/helm/v3 v3.3.2
	k8s.io/api v0.18.8
	k8s.io/apiextensions-apiserver v0.18.8
	k8s.io/apimachinery v0.18.8
	k8s.io/client-go v0.18.8
	sigs.k8s.io/controller-runtime v0.6.3
	sigs.k8s.io/yaml v1.2.0
)
