## Example Resources
This page shows example usage of the following custom resources:

- [Instances](#Instances)

- [InstanceChartRepos](#InstanceChartRepos)
    - [Secrets](#InstanceChartRepo-Secrets)

- [Repositories](#Repositories)

- [Registries](#Registries)

- [Users](#Users)
    - [Secrets](#User-Secrets)

- [Replications](#Replications)
    - [Source Registries](#Source-Registries)
    - [Destination Registries](#Destination-Registries)


### Instances
The`Instance`-resource utilizes the `InstanceChartRepo`-resource for helm deployments

The former describes the helm deployment of a desired harbor instance:

[./instance.yaml](./instance.yaml)
```yaml
apiVersion: registries.mittwald.de/v1alpha1
kind: Instance
metadata:
  name: test-harbor
  namespace: harbor-operator
spec:
  name: test-harbor
  version: v1.3.1
  type: manual
  instanceURL: https://core.harbor.domain/
  garbageCollection:
    schedule:
      type: Hourly
      cron: "0 0 * * * "
  helmChart:
      release: test-harbor
      chart: harbor/harbor
      version: v1.3.1
      namespace: harbor-operator
      valuesYaml: |
          expose:
            type: ingress
            tls:
              enabled: true
            ingress:
              hosts:
                core: core.harbor.domain
            persistence:
              enabled: true
            harborAdminPassword: ""
      [...]
```

Note: Specyfing an empty string for the `harborAdminPassword`-key in `spec.helmChart.valuesYaml` will trigger password generation by the harbor instance itself.
The admin password will then be saved under the key `HARBOR_ADMIN_PASSWORD` in a secret named `HELM_RELEASE_NAME`-`harbor-core`.

[Harbor Garbage Collection](https://goharbor.io/docs/1.10/administration/garbage-collection/) can be configured via `spec.garbageCollection`.
Valid values for `.spec.garbageCollection.schedule.type` are `Hourly`, `Daily`, `Weekly`, `Custom`, `Manual`, and `None` (each starting with a capital letter).

The `None`-value of the schedule type effectively deactivates the garbage collection.
 
### InstanceChartRepos
An `InstanceChartRepo` object references the actual chart repository to be installed.

By utilizing a custom [helm client](https://github.com/mittwald/go-helm-client), the accompanying controller automatically adds/updates the specified chart in it's local cache (similarly to `helm repo add`).

An example `InstanceChartRepo`, using the official [goharbor/harbor-helm](https://github.com/goharbor/harbor-helm) chart might look like this:

[./instancechartrepo.yaml](./instancechartrepo.yaml)
```yaml
apiVersion: registries.mittwald.de/v1alpha1
kind: InstanceChartRepo
metadata:
  name: harbor # use this name as prefix for your chart (myrepo/mychart)
  namespace: harbor-operator
spec:
  url: https://helm.goharbor.io
```

If you need credentials accessing the desired helm repository, you can use kubernetes secrets and reference it with `spec.secretRef.name: <my-secret-name>`

#### InstanceChartRepo Secrets
`instancechartrepo` secrets are saved as kubernetes secrets:

[./instancechartrepo_secret.yaml](./instancechartrepo_secret.yaml)
```yaml
apiVersion: v1
data:
  username: Zm9vCg==
  password: YmFyCg==
kind: Secret
metadata:
  name: harbor-instancechartrepo-secret
  namespace: harbor-operator
```

### Repositories
Repositories (or *Harbor Projects*) hold the information of a Harbor project, mirroring values from it's spec to on to a Harbor instance via the [goharbor-client](https://github.com/mittwald/goharbor-client) library.

A harbor project is "hollow", in the sense of being the authority that holds repository and helm chart information over its lifecycle.

The essential values for a repository are its `.spec.name` and `.spec.parentInstance`. The latter is a reference to the name of the harbor instance.

Notice that project members are also supported - you can specify these under `.spec.memberRequests`.

[./repository.yaml](./repository.yaml)
```yaml
apiVersion: registries.mittwald.de/v1alpha1
kind: Repository
metadata:
  name: repository-1
  namespace: harbor-operator
spec:
  memberRequests:
  - roleID: 1
    memberUser:
     username: "harbor-user"
  name: repository-1
  parentInstance:
    name: test-harbor
  toggleable: false
  metadata:
    enableContentTrust:     false
    autoScan:               false
    severity:               "none"
    reuseSysSVEWhitelist:   false
    public:                 false
    preventVul:             false
```

### Registries
Registries (or *registry endpoints*) are user-defined registry endpoints, for example a custom `docker-registry` or another `harbor` instance.

This example shows a registry endpoint targeted at [Docker Hub](https://hub.docker.com/):

The available registry types specified via `.spec.type` are pre-defined by the [goharbor-client library](https://github.com/mittwald/goharbor-client/blob/master/registry_types.go#L10).

[./registry.yaml](./registry.yaml)
```yaml
apiVersion: registries.mittwald.de/v1alpha1
kind: Registry
metadata:
  name: test-registry
  namespace: harbor-operator
spec:
  id: 1
  name: test-registry
  parentInstance:
    name: test-harbor
  type: docker-hub
  url: https://hub.docker.com
  insecure: false
```

### Replications
Replications (or *replication policies*) enable a harbor registry to replicate resources between each other, or between a harbor instance and an external registry.

Using a [**source registry**](#Source-Registries) via `.spec.src_registry` equals using the **'Pull-based'** option for **'Replication Mode'** in the harbor web UI.

Using a [**destination registry**](#Destination-Registries) via `.spec.dest_registry` equals using the **'Push-based'** option for **'Replication Mode'** in the harbor web UI.

#### Source Registries
Specifying a source registry will trigger harbor to pull the specified resource from the remote registry to the local registry.

Filters and triggers are *optional* fields.

The commented filters would make harbor filter for `alpine:latest`.

[./replication_src.yaml](./replication_src.yaml)
```yaml
apiVersion: registries.mittwald.de/v1alpha1
kind: Replication
metadata:
  name: test-replication-src
  namespace: harbor-operator
spec:
  id: 1
  name: test-replication-src
  parentInstance:
    name: test-harbor
  deletion: false
  override: true
  enabled: true
  src_registry:
    id: 1
    name: test-registry
    parentInstance:
      name: test-harbor
    type: docker-hub
    url: https://hub.docker.com
    insecure: false
#  filters:
#    - type: name
#      value: alpine
#    - type: tag
#      value: latest
#  trigger:
#    type: manual
#    trigger_settings:
#      cron: ""
```

#### Destination Registries

Specifying a destination registry will trigger harbor to **push** the specified resource to the remote registry.

Filters and triggers are *optional* fields.

The commented filters would make harbor filter for `alpine:latest`. 

[./replication_dst.yaml](./replication_dst.yaml)
```yaml
apiVersion: registries.mittwald.de/v1alpha1
kind: Replication
metadata:
  name: test-replication-dst
  namespace: harbor-operator
spec:
  id: 2
  name: test-replication-dst
  parentInstance:
    name: test-harbor
  deletion: false
  override: true
  enabled: true
  dest_registry:
    id: 1
    name: test-registry
    parentInstance:
      name: test-harbor
    type: docker-hub
    url: https://hub.docker.com
    insecure: false
#  filters:
#    - type: name
#      value: alpine
#    - type: tag
#      value: latest
#  trigger:
#    type: manual
#    trigger_settings:
#      cron: ""
```

### Users

Users can access individual harbor projects through project memberships (defined in the desired repository spec). 
The admin role grants full admin access over a harbor instance, toggleable via `.spec.adminRole`

[./user.yaml](./user.yaml)
```yaml
apiVersion: registries.mittwald.de/v1alpha1
kind: User
metadata:
  name: harbor-user
  namespace: harbor-operator
spec:
  name: harbor-user
  parentInstance:
    name: test-harbor
  realname: harboruser
  email: test@example.com
  userSecretRef:
    name: harbor-user
  # available Role IDs:
  # projectAdmin
  # developer
  # guest
  # master
  roleID: projectAdmin
  adminRole: true
```

#### User Secrets
A user CR must contain the name of a kubernetes secret (*LocalObjectReference*) specfied via `.spec.userSecretRef.name`.

**Note**: If a pre-existing (or manually created) secret is specified, it is not included for deletion through reconciliation.

In case the secret with the specified name cannot be found, a new secret with the specified name `.spec.userSecretRef.name` will be created instead.
The users password will then be randomly generated.

**Passwords must be longer than 8 characters, containing at least 1 uppercase letter, 1 lowercase letter and 1 number.**

[./user_secret.yaml](./user_secret.yaml)
```yaml
apiVersion: v1
data:
  password: ...
  username: ...
kind: Secret
metadata:
  name: test-harbor-test-user
  namespace: harbor-operator
type: Opaque
```
