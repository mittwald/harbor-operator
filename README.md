# Harbor Operator

A Kubernetes operator for managing [Goharbor](https://github.com/goharbor/harbor) instances

[![GitHub license](https://img.shields.io/github/license/mittwald/harbor-operator.svg?style=flat-square)](https://github.com/mittwald/harbor-operator/blob/master/LICENSE)
[![Go Doc](https://img.shields.io/badge/godoc-reference-blue.svg?style=flat-square)](https://pkg.go.dev/github.com/mittwald/harbor-operator)
[![Release](https://img.shields.io/github/release/mittwald/harbor-operator.svg?style=flat-square)](https://github.com/mittwald/harbor-operator/releases/latest)

[![Go Report Card](https://goreportcard.com/badge/github.com/mittwald/harbor-operator?style=flat-square)](https://goreportcard.com/badge/github.com/mittwald/harbor-operator)
![Go](https://github.com/mittwald/harbor-operator/workflows/Go/badge.svg?branch=master)
[![Maintainability](https://api.codeclimate.com/v1/badges/6208714b76fca48ea633/maintainability)](https://codeclimate.com/github/mittwald/harbor-operator/maintainability)
[![Test Coverage](https://api.codeclimate.com/v1/badges/6208714b76fca48ea633/test_coverage)](https://codeclimate.com/github/mittwald/harbor-operator/test_coverage)

##### This project is still under development and not stable yet - breaking changes may happen at any time and without notice
## Features
**Easy Harbor deployment & scaling**:
- Every Harbor instance is bound only to the deployed Custom Resource.
The operator utilizes a [helm client](https://github.com/mittwald/go-helm-client) library for the management of these instances

**Custom chart repositories**:
- If you need to install a customized or private Harbor helm chart, the
 `InstanceChartRepository` resource allows you to do so. The official Harbor Helm chart can be found [here](https://github.com/goharbor/harbor-helm)

**Harbor resource reconciliation**:
- This operator automatically manages Harbor components by utilizing
 a custom [harbor client](https:/github.com/mittwald/goharbor-client).

### CRDs
registries.mittwald.de/v1alpha2:
- [InstanceChartRepositories](./config/samples/README.md#InstanceChartRepositories)
- [Instances](./config/samples/README.md#Instances)
- [Projects](./config/samples/README.md#Projects)
- [Registries](./config/samples/README.md#Registries)
- [Replications](./config/samples/README.md#Replications)
- [Users](./config/samples/README.md#Users)

To get an overview of the individual resources that come with this operator,
take a look at the [samples directory](./config/samples).

## Installation
### Helm
The helm chart of this operator can be found under [./deploy/helm-chart/harbor-operator](./deploy/helm-chart/harbor-operator).

Alternatively, you can use the the [Mittwald Kubernetes Helm Charts](https://github.com/mittwald/helm-charts) repository:
```shell script
helm repo add mittwald https://helm.mittwald.de
helm repo update
helm install harbor-operator mittwald/harbor-operator --namespace my-namespace
```

## Documentation
For more specific documentation, please refer to the [godoc](https://pkg.go.dev/github.com/mittwald/harbor-operator) of this repository.

#### Web UI
For a trouble-free experience with created instances, a valid TLS certificate is required.

However, local installations can be accessed via `http://`.

**Automatic certificate creation** can be configured via the `Instance` resource:

 `.spec.helmChart.valuesYaml.expose.ingress.annotations`.

Example annotation value using cert-manager as the cluster-issuer:

`cert-manager.io/cluster-issuer: "letsencrypt-issuer"`

### Local Development
To start the operator locally, run:
```shell script
make run
```

To start a debug session using [delve](https://github.com/go-delve/delve), run:
```shell script
make debug
```
This will start a debugging server with the listen address `localhost:2345`.

When making changes to API definitions (located in [./api/v1alpha1](api/v1alpha2)),
make sure to re-generate manifests via:
```shell script
make manifests
```

#### Testing
To test the operator, simply run:
```shell script
make test
```

This will spin up a local [envtest](https://sdk.operatorframework.io/docs/building-operators/golang/references/envtest-setup)
environment and execute the provided tests.

Alternatively, you can run tests by [ginkgo](http://onsi.github.io/ginkgo/#getting-ginkgo) via:
``` shell script
ginkgo test ./...
```
Or via the go test suite:
``` shell script
go test -v ./...
```

_Some_ unit tests require a [mocked controller-runtime client](./controllers/internal/mocks/runtime_client_mock.go).
This mock is generated using: `make mock-runtime-client`.

#### Deploying example resources
Note: When using the provided examples and running the operator locally, an entry to your `/etc/hosts` is
 needed:
```shell script
127.0.0.1 core.harbor.domain
```

Example resources can be deployed using the files provided in the [samples directory](./config/samples).

To start testing, simply apply these after starting the operator:

```
k create -f config/samples/
```

After a successful installation, the Harbor portal may be accessed either by `localhost:30002` or `core.harbor.domain:30002`.
