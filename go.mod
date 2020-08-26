module github.com/mittwald/harbor-operator

go 1.15

require (
	github.com/elazarl/goproxy v0.0.0-20200315184450-1f3cb6622dad // indirect
	github.com/fatih/color v1.9.0 // indirect
	github.com/go-logr/logr v0.1.0
	github.com/go-openapi/jsonreference v0.19.4 // indirect
	github.com/go-openapi/runtime v0.19.20 // indirect
	github.com/golang/mock v1.4.3
	github.com/imdario/mergo v0.3.8
	github.com/jinzhu/copier v0.0.0-20190924061706-b57f9002281a
	github.com/mattn/go-colorable v0.1.6 // indirect
	github.com/mittwald/go-helm-client v0.3.0
	github.com/mittwald/goharbor-client v1.0.4
	github.com/operator-framework/operator-sdk v0.18.2
	github.com/spf13/pflag v1.0.5
	github.com/spf13/viper v1.4.0
	github.com/stretchr/testify v1.6.1
	go.mongodb.org/mongo-driver v1.3.5 // indirect
	golang.org/x/net v0.0.0-20200707034311-ab3426394381 // indirect
	golang.org/x/sys v0.0.0-20200420163511-1957bb5e6d1f // indirect
	gopkg.in/yaml.v2 v2.3.0
	helm.sh/helm/v3 v3.2.4
	k8s.io/api v0.18.2
	k8s.io/apimachinery v0.18.2
	k8s.io/client-go v12.0.0+incompatible
	sigs.k8s.io/controller-runtime v0.6.0
)

replace (
	github.com/Azure/go-autorest => github.com/Azure/go-autorest v13.3.2+incompatible // Required by OLM
	k8s.io/client-go => k8s.io/client-go v0.18.2 // Required by prometheus-operator
)

replace github.com/docker/docker => github.com/moby/moby v0.7.3-0.20190826074503-38ab9da00309 // Required by Helm
