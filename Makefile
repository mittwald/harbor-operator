# Current Operator version
VERSION ?= 0.0.1
# Default bundle image tag
BUNDLE_IMG ?= harbor-operator-bundle:$(VERSION)
# Options for 'bundle-build'
ifneq ($(origin CHANNELS), undefined)
BUNDLE_CHANNELS := --channels=$(CHANNELS)
endif
ifneq ($(origin DEFAULT_CHANNEL), undefined)
BUNDLE_DEFAULT_CHANNEL := --default-channel=$(DEFAULT_CHANNEL)
endif
BUNDLE_METADATA_OPTS ?= $(BUNDLE_CHANNELS) $(BUNDLE_DEFAULT_CHANNEL)

# Image URL to use all building/pushing image targets
IMG ?= quay.io/mittwald/harbor-operator:latest
# Produce CRDs that work back to Kubernetes 1.11 (no version conversion)
CRD_OPTIONS ?= "crd:trivialVersions=true,preserveUnknownFields=false"

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

all: manager

# Run tests
ENVTEST_ASSETS_DIR = $(shell pwd)/testbin
test: generate fmt vet manifests
	mkdir -p $(ENVTEST_ASSETS_DIR)
	test -f $(ENVTEST_ASSETS_DIR)/setup-envtest.sh || curl -sSLo $(ENVTEST_ASSETS_DIR)/setup-envtest.sh https://raw.githubusercontent.com/kubernetes-sigs/controller-runtime/v0.8.1/hack/setup-envtest.sh
	source $(ENVTEST_ASSETS_DIR)/setup-envtest.sh; fetch_envtest_tools $(ENVTEST_ASSETS_DIR); setup_envtest_env $(ENVTEST_ASSETS_DIR); go test ./... -coverprofile cover.out

# Build manager binary
manager: generate fmt vet
	go build -o bin/manager main.go

# Run against the configured Kubernetes cluster in ~/.kube/config
run: generate fmt vet manifests
	go run ./main.go

debug: generate fmt vet manifests manager
	dlv --listen=:2345 --headless=true --api-version=2 exec bin/manager --

# Generate manifests e.g. CRD, RBAC etc.
manifests: controller-gen
	go mod download && go mod tidy
	$(CONTROLLER_GEN) $(CRD_OPTIONS) rbac:roleName=manager-role webhook paths="./..." output:crd:artifacts:config=config/crd/bases
	# build helm-chart
	echo "# AUTOGENERATED BY 'make manifests' - DO NOT EDIT!" | tee \
		./deploy/helm-chart/harbor-operator/templates/role.yaml \
		./deploy/helm-chart/harbor-operator/templates/role_binding.yaml
	sed 's/manager-role/{{ include "harbor-operator.name" . }}/g' ./config/rbac/role.yaml >> ./deploy/helm-chart/harbor-operator/templates/role.yaml
	sed 's/manager-rolebinding/{{ include "harbor-operator.name" . }}/g; s/manager-role/{{ include "harbor-operator.name" . }}/g; s/default/{{ include "harbor-operator.name" . }}/g; s/system/{{ .Release.Namespace }}/g' \
		./config/rbac/role_binding.yaml >> ./deploy/helm-chart/harbor-operator/templates/role_binding.yaml

# Run go fmt against code
fmt:
	go fmt $$(go list ./...)

# Run go vet against code
vet:
	go vet $$(go list ./...)

# Generate code
generate: controller-gen mock-runtime-client
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./apis/registries/..."
	rm -r ./pkg/apis/v* && cp -rf ./apis/registries/* ./pkg/apis/
	cd pkg/apis && go mod tidy

# Build the docker image
docker-build:
	docker build . -t ${IMG}

# Push the docker image
docker-push:
	docker push ${IMG}

# Download controller-gen locally if necessary
CONTROLLER_GEN = $(shell pwd)/bin/controller-gen
controller-gen:
	$(call go-get-tool,$(CONTROLLER_GEN),sigs.k8s.io/controller-tools/cmd/controller-gen@v0.4.1)

# Download kustomize locally if necessary
KUSTOMIZE = $(which kustomize)
kustomize:
	$(call go-get-tool,$(KUSTOMIZE),sigs.k8s.io/kustomize/kustomize/v3@v3.8.7)


# go-get-tool will 'go get' any package $2 and install it to $1.
PROJECT_DIR := $(shell dirname $(abspath $(lastword $(MAKEFILE_LIST))))
define go-get-tool
@[ -f $(1) ] || { \
set -e ;\
TMP_DIR=$$(mktemp -d) ;\
cd $$TMP_DIR ;\
go mod init tmp ;\
echo "Downloading $(2)" ;\
GOBIN=$(PROJECT_DIR)/bin go get $(2) ;\
rm -rf $$TMP_DIR ;\
}
endef

# Generate bundle manifests and metadata, then validate generated files.
.PHONY: bundle
bundle: manifests
	operator-sdk generate kustomize manifests -q
	cd config/manager && $(KUSTOMIZE) edit set image controller=$(IMG)
	$(KUSTOMIZE) build config/manifests | operator-sdk generate bundle -q --overwrite --version $(VERSION) $(BUNDLE_METADATA_OPTS)
	operator-sdk bundle validate ./bundle

# Build the bundle image.
.PHONY: bundle-build
bundle-build:
	docker build -f bundle.Dockerfile -t $(BUNDLE_IMG) .

# Mock generation for the controller runtime client used by some unit tests
CONTROLLER_RUNTIME_VERSION=$(shell echo `find . -maxdepth 1 -name 'go.mod' | xargs awk '$$1 == "sigs.k8s.io/controller-runtime"{print $$2}'`)

MOCKERY_VERSION=v2.6.0

mock-runtime-client:
	@echo installing mockery $(MOCKERY_VERSION)
	go get github.com/vektra/mockery/v2/.../@$(MOCKERY_VERSION)
	@echo generating mocked k8s runtime client via
	@echo sigs.k8s.io/controller-runtime@$(CONTROLLER_RUNTIME_VERSION)/pkg/client.Client
	$(eval TMP := $(shell mktemp -d))
	git clone https://github.com/kubernetes-sigs/controller-runtime -q --branch $(CONTROLLER_RUNTIME_VERSION) $(TMP)
	cd $(TMP) && mockery --dir pkg/client/ --name Client --structname MockClient \
	--filename=runtime_client_mock.go --output "$(PWD)/controllers/registries/internal/mocks"
	rm -rf $(TMP)