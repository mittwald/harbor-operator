# Harbor Operator

A Kubernetes operator for managing [Goharbor](https://github.com/goharbor/harbor) instances

## Features

**Easy deployment & scaling**: Every Harbor instance is bound only to the deployed Custom Resource.
The operator utilizes a [helm client](https://github.com/mittwald/go-helm-client) library for the management of these instances

**Custom chart repositories**: If you need to install a customized or private [harbor helm chart](https://github.com/goharbor/harbor-helm), the `instancechartrepo` resource allows you to do so

**Harbor resource reconciliation**: This operator automatically manages the following goharbor components by utilizing a custom [harbor client library](https://github.com/mittwald/goharbor-client):

- users
- repositories
- replications
- registries

**Helm Chart**: A Helm chart of this operator can be found under [./deploy/chart](./deploy/chart)

## CRDs
- registriesv1alpha1:
    - instances.registries.mittwald.de
    - instancechartrepos.registries.mittwald.de
    - repository.registries.mittwald.de
    - users.registries.mittwald.de
    - replications.registries.mittwald.de
    - registries.registries.mittwald.de
    
### Installation
To get an overview of the individual resources that come this operator, take a look at the [examples directory](./examples).
 
#### Web UI
For a trouble-free experience, a valid TLS certificate is required.

For automatic certificate creation, you can set the desired cluster certificate issuer via the instance spec:
 
`.spec.helmChart.valuesYaml.expose.ingress.annotations`

An example value for this annotation, using cert-manager as the cluster-issuer: `cert-manager.io/cluster-issuer: "letsencrypt-issuer"`