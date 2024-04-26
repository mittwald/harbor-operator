# Harbor Operator

A Kubernetes operator for automated management of [Goharbor](https://github.com/goharbor/harbor) instances

[![GitHub license](https://img.shields.io/github/license/mittwald/harbor-operator.svg?style=flat-square)](https://github.com/mittwald/harbor-operator/blob/master/LICENSE)
[![Go Doc](https://img.shields.io/badge/godoc-reference-blue.svg?style=flat-square)](https://pkg.go.dev/github.com/mittwald/harbor-operator)
[![Release](https://img.shields.io/github/release/mittwald/harbor-operator.svg?style=flat-square)](https://github.com/mittwald/harbor-operator/releases/latest)

[![Go Report Card](https://goreportcard.com/badge/github.com/mittwald/harbor-operator?style=flat-square)](https://goreportcard.com/badge/github.com/mittwald/harbor-operator)
![Go](https://github.com/mittwald/harbor-operator/workflows/Go/badge.svg?branch=master)

## Table of contents
- [Installation](#Installation)
- [Architecture](#Architecture)
- [CRDs](#CRDs)
- [Documentation](#Documentation)
  - [Local Development](#Local-Development)
  - [Testing](#Testing)
- [Example Deployment](#Example-Deployment)

### Installation

The helm chart of this operator can be found in this repository under [./deploy/chart](./deploy/chart)
Alternatively, you can use the [helm.mittwald.de](https://helm.mittwald.de) chart repository:

```shell script
helm repo add mittwald https://helm.mittwald.de
helm repo update
helm install harbor-operator mittwald/harbor-operator --namespace my-namespace
```

### Architecture

- The operator manages the deployment of [goharbor/harbor](https://github.com/goharbor/harbor) instances
  
- Many components / features of Harbor can be accessed by creating _Custom Resource Definitons_.
  Resource changes are reconciled in the main controller loop.
    > For a full list of Harbor's features, please refer to [goharbor/harbor#features](https://github.com/goharbor/harbor#features)
  
- The operator manages Harbor components by utilizing the [mittwald/goharbor-client](https:/github.com/mittwald/goharbor-client) API client

- Customized or private Harbor helm charts are supported via the `InstanceChartRepository` resource
  > The official Harbor Helm chart can be found [here](https://github.com/goharbor/harbor-helm)

```
 0
/|\ User
/ \

 |
 |      creates         ┌───────────────────────────────┐
 ├────────────────────▶ |    InstanceChartRepository    |
 |                      |       (Custom Resource)       |
 |                      └───────────────────────────────┘
 |                                             ▲
 |      creates         ┌───────────────────┐  |
 ├────────────────────▶ |      Instance     |  |
 |                      | (Custom Resource) |  |
 |                      └───────────────────┘  | watches
 |                                    ▲        |
 |                                    |        |
 |                            watches |        |
 |                                    |        |           creates & updates
 |                                  ┌─┴────────┴──────┐      (via Instance)      
 |                                  │ Harbor Operator ├──────────────────────────┐
 |                                  └─────────┬─────┬─┘                          |
 |                                            ╎     |                            |
 |                                    watches ╎     |                            |
 |                                            ╎     |                            |
 |      creates         ┌─────────────────┐   ╎     |         ┌─────────┐  ┌─────┴──────┐
 ├────────────────────▶ |     Project     ├ - ┼ - - └─────── ▶| Harbor  ├──┤   Harbor   |
 |                      |(Custom Resource)|   ╎      perform  |   API   |  |Helm Release|
 |                      └─────────────────┘   ╎      CRUD     └─────────┘  └────────────┘
 |                              ▲             ╎      via the CRs on the left
 |                              |             ╎
 |           has access through |             ╎
 |               membership     |             ╎
 |                              |             ╎
 |      creates         ┌───────┴─────────┐   ╎
 ├────────────────────▶ |      User       ├ - ┤
 |                      |(Custom Resource)|   ╎
 |                      └─────────────────┘   ╎
 |      creates         ┌─────────────────┐   ╎
 ├────────────────────▶ |    Registry     ├ - ┤
 |                      |(Custom Resource)|   ╎
 |                      └─────────────────┘   ╎
 |                              ▲             ╎
 |                              |             ╎
 |                  is owned by |             ╎
 |                              |             ╎
 |      creates         ┌───────┴─────────┐   ╎
 └────────────────────▶ |    Replication  ├ - ┘
                        |(Custom Resource)|
                        └─────────────────┘
```

### CRDs

The following _Custom Resource Definitions_ can be used to create / configure Harbor components:

- [InstanceChartRepositories](./config/samples/README.md#InstanceChartRepositories)
- [Instances](./config/samples/README.md#Instances)
- [Projects](./config/samples/README.md#Projects)
- [Registries](./config/samples/README.md#Registries)
- [Replications](./config/samples/README.md#Replications)
- [Users](./config/samples/README.md#Users)

To get an overview of the individual resources that come with this operator,
take a look at the [samples directory](./config/samples).

## Documentation
For more specific documentation, please refer to the [godoc](https://pkg.go.dev/github.com/mittwald/harbor-operator) of this repository.

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

Local installations can be accessed via `http://`

When making changes to API definitions (located in [./apis/registries/v1alpha2](/apis/registries/v1alpha2)),
make sure to re-generate manifests via:
```shell script
make manifests
```

### Testing
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

_Some_ unit tests require a [mocked controller-runtime client](controllers/registries/internal/mocks/runtime_client_mock.go).
This mock is generated using: `make mock-runtime-client`.

### Example Deployment
> Note: If you want to test a local setup using an URL, you will need to append it to your `/etc/hosts`:
> ```shell script
> 127.0.0.1 core.harbor.domain
> ```

Example resources can be deployed using the files provided in the [samples directory](./config/samples).

To start testing, simply apply these after the operator has started:

```
kubectl create -f config/samples/
```

After a successful installation, the Harbor portal may be accessed either by `localhost:30002` or `core.harbor.domain:30002`.
