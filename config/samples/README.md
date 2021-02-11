# Example Resources
The resources contained in this directory help you to get started using harbor-operator.

This page covers example usage of all resources currently supported by this operator:

[Instances](#Instances) (Harbor Helm installations)

[InstanceChartRepositories](#InstanceChartRepositories) (Helm chart reference used for instance installations)
    
   - [Secrets](#InstanceChartRepository-Secrets) (_Optional_ secret values for the above)

[Projects](#Projects)

[Registries](#Registries)

[Replications](#Replications)
    
   - [Source Registries](#Source-Registries)

   - [Destination Registries](#Destination-Registries)

[Users](#Users)
   
   - [User Secrets](#User-Secrets)

### Instances
The `Instance`-resource specifies the desired Harbor helm installation:

[registries_v1alpha2_instance.yaml](./registries_v1alpha2_instance.yaml)
```yaml
apiVersion: registries.mittwald.de/v1alpha2
kind: Instance
metadata:
  name: test-harbor
  namespace: harbor-operator
spec:
  name: test-harbor
  type: manual
  instanceURL: http://core.harbor.domain:30002
  garbageCollection:
    cron: "0 * * * *"
    scheduleType: "Hourly"
  helmChart:
      release: test-harbor
      chart: harbor/harbor
      version: v1.5.3 # equalling to Harbor OSS version v2.1.3
      # see https://github.com/goharbor/harbor-helm/releases for a full list of supported versions
      namespace: harbor-operator
      valuesYaml: | # helm chart values
        expose:
          type: nodePort
          tls:
            enabled: false
            secretName: ""
            notarySecretName: ""
            commonName: ""
          ingress:
        [...]
```

The operator utilizes the [InstanceChartRepository](#InstanceChartRepositories)-resource for helm installations.
The helm chart version can be specified via `.spec.helmChart.version`.

Note: Specifying an empty string for the `harborAdminPassword`-key in `spec.helmChart.valuesYaml` will trigger
 password generation by the Harbor instance itself.
The admin password will be saved under the key `HARBOR_ADMIN_PASSWORD` in a secret named `HELM_RELEASE_NAME
`-`harbor-core`.

[Harbor Garbage Collection](https://goharbor.io/docs/1.10/administration/garbage-collection/) can be configured via `spec.garbageCollection`.
Valid values for `.scheduleType` are `Hourly`, `Daily`, `Weekly`, `Custom`, `Manual`, and `None` (each starting with
 a capital letter).
The `.cron` parameter is a cron expression:

```yaml
  garbageCollection:
    cron: "0 * * * *"
    scheduleType: "Hourly"
```

A `None`-value of the schedule type effectively deactivates the garbage collection.
 
### InstanceChartRepositories
An `InstanceChartRepository` is a reference to a helm chart repository which contains a `goharbor` helm chart.

By utilizing a custom [helm client](https://github.com/mittwald/go-helm-client), 
the accompanying controller automatically adds / updates the specified 
chart to its local cache (similarly to `helm repo add` / `helm repo update`).

An example `InstanceChartRepository`, using the official [goharbor/harbor-helm](https://github.com/goharbor/harbor-helm) chart might look like this:

[registries_v1alpha2_instancechartrepository.yaml](./registries_v1alpha2_instancechartrepository.yaml)
```yaml
apiVersion: registries.mittwald.de/v1alpha2
kind: InstanceChartRepository
metadata:
  name: harbor
  namespace: harbor-operator
spec:
  url: https://helm.goharbor.io
```

> When using `kubectl get`, the following fields are exposed through the CRs status fields:
> ```shell script
> kubectl get instancechartrepos.registries.mittwald.de 
> NAME     URL                        STATUS
> harbor   https://helm.goharbor.io   Ready
> ```

If you need credentials accessing the desired helm repository (e.g. when hosting your own), you can use a kubernetes secret and reference it with `spec.secretRef.name: <my-secret-name>`

#### InstanceChartRepository Secrets
An `instancechartrepository`'s secret is a kubernetes secret:

[instancechartrepository-secret.yaml](instancechartrepository-secret.yaml)
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

### Projects
A `Project` holds the desired specification of a Harbor project, mirroring values from its spec on to a Harbor instance via the [mittwald/goharbor-client](https://github.com/mittwald/goharbor-client) library.

A harbor project is "hollow", in the sense of being the authority that holds repository and helm chart information over its lifecycle.

The essential values for a repository are its `.spec.name` and `.spec.parentInstance`. The latter is a reference to the name of the harbor instance.

Notice that the operator supports project members, too - you can specify these under `.spec.memberRequests`.

[registries_v1alpha2_project.yaml](./registries_v1alpha2_project.yaml)
```yaml
apiVersion: registries.mittwald.de/v1alpha2
kind: Project
metadata:
  name: repository-1
  namespace: harbor-operator
spec:
  memberRequests:
  - role: ProjectAdmin # one of "ProjectAdmin", "Developer", "Guest" or "Master"
    user:
     name: "harbor-user" # reference to a user object
  storageLimit: -1 # storage quota in GB
  name: harbor-project
  parentInstance:
    name: test-harbor
# All project metadata fields but 'public' are optional
  metadata:
    public:                 false
#    enableContentTrust:     false
#    autoScan:               false
#    severity:               "none"
#    reuseSysCVEAllowlist:   false
#    preventVul:             false
```

> When using `kubectl get`, the following fields are exposed through the CRs status fields:
> ```shell script
> kubectl get projects.registries.mittwald.de
> NAME             STATUS   ID    PUBLIC
> harbor-project   Ready    1     false
> ```

### Registries
A `Registry` is a registry endpoint, for example a custom `docker-registry
`, `docker-hub` or another `harbor` instance.

> Before creating a [Replication](#Replications) to a remote [destination registry](#Destination-Registries), a corresponding `Registry` **must** exist.

> Available registry types (configurable via `.spec.type`) are:
> `harbor`, `docker-hub`, `docker-registry`, `huawei-SWR`, `google-gcr`, `aws-ecr`,
> `azure-acr`, `ali-acr`, `jfrog-artifactory`, `quay`, `gitlab`, `helm-hub`.

This example shows a `Registry` endpoint targeted at [Docker Hub](https://hub.docker.com/):

[registries_v1alpha2_registry-dockerhub.yaml](./registries_v1alpha2_registry-dockerhub.yaml)
```yaml
apiVersion: registries.mittwald.de/v1alpha2
kind: Registry
metadata:
  name: test-registry-dockerhub
  namespace: harbor-operator
spec:
  name: test-registry-dockerhub
  parentInstance:
   name: test-harbor
  type: docker-hub
  url: https://hub.docker.com
  insecure: false
```

This example shows a `Registry` with its endpoint set to a local [registry](https://hub.docker.com/_/registry): 

[registries_v1alpha2_registry-local.yaml](./registries_v1alpha2_registry-local.yaml)
```yaml
apiVersion: registries.mittwald.de/v1alpha2
kind: Registry
metadata:
  name: test-registry-local
  namespace: harbor-operator
spec:
  name: test-registry-local
  parentInstance:
    name: test-harbor
  type: docker-registry
  url: "http://registry-docker-registry:5000/"
  insecure: true
```

When using `kubectl get`, the following fields are exposed through the CRs status fields:

```shell script
kubectl get registries.registries.mittwald.de
NAME                     STATUS   ID
test-registry-dockerhub  Ready    1
test-registry-local      Ready    2
```

### Replications
A `Replication` (or *replication policy*) enables Harbor to replicate resources
**to** or **from** its own [Registries](#Registries).

Using a [**source registry**](#Source-Registries) via `.spec.srcRegistry` equals using the **'Pull-based'** option for **'Replication Mode'** in the harbor web UI.

Using a [**destination registry**](#Destination-Registries) via `.spec.destRegistry` equals using the **'Push-based'** option for **'Replication Mode'** in the harbor web UI.

When using `kubectl get`, the following fields are exposed through the CRs status fields:

```shell script
kubectl get replications.registries.mittwald.de
NAME                   STATUS   ID    ENABLED   SOURCE                DESTINATION
test-replication-dst   Ready    1     true      harbor                docker-hub
test-replication-src   Ready    2     true      docker-hub            harbor
```

#### Source Registries
Specifying a source registry will trigger harbor to _pull_ the specified resource from the remote registry to the
 local registry.

Filters and triggers are *optional* fields.

[registries_v1alpha2_replication_src.yaml](./registries_v1alpha2_replication_src.yaml)
```yaml
apiVersion: registries.mittwald.de/v1alpha2
kind: Replication
metadata:
  name: test-replication-src
  namespace: harbor-operator
spec:
  name: test-replication-src
  parentInstance:
    name: test-harbor
  replicateDeletion: false # do not replicate deletion operations
  override: true # override the resources on the destination registry
  enabled: true
  triggerAfterCreation: true # trigger this replication after its creation
  srcRegistry: # it is optional to create the srcRegistry via a registry custom resource
    name: test-registry-local
#  filters:
#    - type: name
#      value: alpine
#    - type: tag
#      value: latest
#  trigger:
#    type: manual
#    triggerSettings:
#      cron: ""
```

The commented `filters` section in this example will make harbor filter the provided registry for an image named `alpine:latest`:

#### Destination Registries

Specifying a destination registry will trigger harbor to **push** the specified resource to the remote registry.

Filters and triggers are *optional* fields.

[registries_v1alpha2_replication_dst.yaml](./registries_v1alpha2_replication_dst.yaml)
```yaml
apiVersion: registries.mittwald.de/v1alpha2
kind: Replication
metadata:
  name: test-replication-dst
  namespace: harbor-operator
spec:
  name: test-replication-dst
  parentInstance:
    name: test-harbor
  replicateDeletion: false  # do not replicate deletion operations
  override: true # override the resources on the destination registry
  enabled: true
  triggerAfterCreation: true
  destRegistry: # you have to create the destRegistry via a registry custom resource first
    name: test-registry-local
#  filters:
#    - type: name
#      value: alpine
#    - type: tag
#      value: latest
#  trigger:
#    type: manual
#    triggerSettings:
#      cron: ""
```

### Users

A `User` can access individual harbor projects through project memberships (defined in the desired [repository](#Repositories) spec). 
The admin role grants full admin access over a harbor instance to the `User`, toggleable via `.spec.adminRole`.

If `.spec.userSecretRef` specifies a non-existing secret, the strength for a generated secret password value can
 be defined via `.spec.passwordStrength`.

[registries_v1alpha2_user.yaml](./registries_v1alpha2_user.yaml)
```yaml
apiVersion: registries.mittwald.de/v1alpha2
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
  sysAdmin: true
  passwordStrength: 16
```

#### User Secrets
A user CR **must** contain the name of a kubernetes secret (*LocalObjectReference*) specfied via `.spec.userSecretRef
.name`. 
However, it is _optional_ to create this secret beforehand.

> **Note**: When specifying a pre-existing (or manually created) secret, **it is included for deletion** through reconciliation when the user gets deleted.

> **Passwords must be between 8 and 128 characters long, containing at least 1 uppercase letter, 1 lowercase letter and 1 number.**
> Generated passwords default to a length of 8 characters.

[user-secret.yaml](./user-secret.yaml)
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
