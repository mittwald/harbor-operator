# Harbor Operator

A Kubernetes operator for managing [Goharbor](https://github.com/goharbor/harbor) instances

[![GitHub license](https://img.shields.io/github/license/mittwald/harbor-operator.svg)](https://github.com/mittwald/harbor-operator/blob/master/LICENSE)
[![GoDoc](https://godoc.org/github.com/mittwald/harbor-operator?status.svg)](https://pkg.go.dev/github.com/mittwald/harbor-operator)
![Go](https://github.com/mittwald/harbor-operator/workflows/Go/badge.svg?branch=master)
[![Maintainability](https://api.codeclimate.com/v1/badges/6208714b76fca48ea633/maintainability)](https://codeclimate.com/github/mittwald/harbor-operator/maintainability)
[![Test Coverage](https://api.codeclimate.com/v1/badges/6208714b76fca48ea633/test_coverage)](https://codeclimate.com/github/mittwald/harbor-operator/test_coverage)
##### This project is still under development and not stable yet - breaking changes may happen at any time and without notice
## Features
**Easy Harbor deployment & scaling**: Every Harbor instance is bound only to the deployed Custom Resource.
The operator utilizes a [helm client](https://github.com/mittwald/go-helm-client) library for the management of these instances

**Custom chart repositories**: If you need to install a customized or private Harbor helm chart, the `instancechartrepo` resource allows you to do so. The official Helm chart can be found [here](https://github.com/goharbor/harbor-helm)

**Harbor resource reconciliation**: This operator automatically manages the following Harbor components by utilizing a custom [harbor client library](https://github.com/mittwald/goharbor-client):

- users
- repositories
- replications
- registries

**Helm Chart**: [Installation Guide](#Helm)

### CRDs
- registriesv1alpha1:
    - instances.registries.mittwald.de
    - instancechartrepos.registries.mittwald.de
    - repository.registries.mittwald.de
    - users.registries.mittwald.de
    - replications.registries.mittwald.de
    - registries.registries.mittwald.de
    
To get an overview of the individual resources that come this operator, take a look at the [examples directory](./examples).

## Installation
### Helm
The helm chart of this operator can be found under [./deploy/helm-chart/harbor-operator](./deploy/helm-chart/harbor-operator).

Alternatively, you can use the the [Mittwald Kubernetes Helm Charts](https://github.com/mittwald/helm-charts) repository:
```bash
helm repo add mittwald https://helm.mittwald.de
helm repo update
helm install harbor-operator mittwald/harbor-operator --namespace my-namespace
```

## Documentation
For more specific documentation, please refer to the [godoc](https://pkg.go.dev/github.com/mittwald/harbor-operator) of this repository

#### Web UI
For a trouble-free experience with created instances, a valid TLS certificate is required.

For automatic certificate creation, you can set the desired cluster certificate issuer via the instance spec:
 
`.spec.helmChart.valuesYaml.expose.ingress.annotations`

Example annotation, using cert-manager as the cluster-issuer: 

`cert-manager.io/cluster-issuer: "letsencrypt-issuer"`