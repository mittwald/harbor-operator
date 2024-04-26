ENVTEST_K8S_VERSION = 1.26.0
CONTROLLER_TOOLS_VERSION ?= v0.14.0

REVISION := $(shell git rev-parse --show-toplevel)
OPERATOR_NAME := $(shell basename $(REVISION))

# Image URL to use all building/pushing image targets
IMG ?= quay.io/mittwald/harbor-operator:latest

.PHONY: default
.DEFAULT: default
default: | generate go manifests controller-gen imports manager test docker-build

## Environment Variables

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

# Setting SHELL to bash allows bash commands to be executed by recipes.
# Options are set to exit when a recipe line exits non-zero or a piped command fails.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

GO_FILES := $(shell find . -type f -name '*.go' -not -name 'zz_generated.deepcopy.go')

UNAME := $(shell uname -s)

# Set sed command based on OS
ifeq ($(UNAME),Darwin)
SED=gsed
else
SED=sed
endif

## Location to install dependencies to
LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

# Tool Binaries
KUSTOMIZE ?= $(LOCALBIN)/kustomize
CONTROLLER_GEN ?= $(LOCALBIN)/controller-gen
ENVTEST ?= $(LOCALBIN)/setup-envtest

##@ Help

.SILENT: help
.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Docker

.PHONY: docker-build
.SILENT: docker-build
docker-build: test ## Build docker image with the manager.
	docker build -t ${IMG} .

.PHONY: docker-push
.SILENT: docker-push
docker-push: ## Push docker image with the manager.
	docker push ${IMG}

##@ Development

.PHONY: manager
.SILENT: manager
manager: generate fmt vet ## Build manager binary.
	@go build -o bin/manager main.go

##@ Code Generation

.PHONY: manifests
.SILENT: manifests
manifests: controller-gen helm-chart helm-template ## Generate manifests e.g. CRD, RBAC, as well as the Helm chart.
	@$(CONTROLLER_GEN) rbac:roleName=manager-role crd:generateEmbeddedObjectMeta=true webhook paths="./apis/..." output:crd:artifacts:config=config/crd/bases

.PHONY: generate
.SILENT: generate
generate: controller-gen ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
	echo "Running controller-gen.."
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."
	echo "Running go generate for main package.."
	go generate -tags ignore ./...
	[ -d ./webhook ] && \
	echo "Running go generate for webhook package.." && \
	cd ./webhook; go generate -tags ignore ./... || echo No webhook package found, skipping.. && \
	cd ../

##@ Go

.PHONY: go
.SILENT: go
go: fmt vet imports lint ## Prepare go files.
	go mod download && go mod tidy

.SILENT: fmt
.PHONY: fmt
fmt: ## Run go fmt & goimports against code.
	echo "Running go fmt ./..."
	@go fmt ./...

.SILENT: vet
.PHONY: vet
vet: ## Run go vet against code.
	echo "Running go vet ./..."
	@go vet ./...

.PHONY: imports
imports: ## Run goimports against code.
	@command -v goimports 1>&/dev/null && \
	goimports -w . || echo "goimports not installed, skipping.."

.PHONY: lint
lint: ## Run golangci-lint against code.
	@command -v yq 1>&/dev/null || \
	@command -v docker 1>&/dev/null && \
	echo "Running golangci-lint ($(shell cat `pwd`/.github/workflows/workflow.yml | yq '.env.GOLANGCI_LINT_VERSION'))..";
	@docker run --rm -v $(shell pwd):/app -w /app golangci/golangci-lint:$(shell cat `pwd`/.github/workflows/workflow.yml | yq '.env.GOLANGCI_LINT_VERSION') golangci-lint run --timeout=10m -v -E godox ./... || echo "yq or docker not installed, skipping.."

.PHONY: controller-gen
controller-gen: $(CONTROLLER_GEN) ## Download controller-gen locally if necessary. If wrong version is installed, it will be overwritten.
$(CONTROLLER_GEN): $(LOCALBIN)
	@test -s $(LOCALBIN)/controller-gen && $(LOCALBIN)/controller-gen --version | grep -q $(CONTROLLER_TOOLS_VERSION) || \
	GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-tools/cmd/controller-gen@$(CONTROLLER_TOOLS_VERSION)

##@ Testing

.PHONY: test
.SILENT: test
test: envtest ## Run envtests.
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) --bin-dir $(LOCALBIN) -p path)" go test ./... -coverprofile cover.out

.PHONY: envtest
envtest: $(ENVTEST) ## Download envtest-setup locally if necessary.
$(ENVTEST): $(LOCALBIN)
	@test -s $(LOCALBIN)/setup-envtest || GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-runtime/tools/setup-envtest@latest

.PHONY: run
run: ## Run against the configured Kubernetes cluster in the current KUBECONFIG context.
ifeq ($(wildcard /tmp/k8s-webhook-server/serving-certs/tls.crt),)
	mkdir -p /tmp/k8s-webhook-server/serving-certs && \
	openssl req -new -newkey rsa:4096 -x509 -sha256 -days 1 -nodes -batch -out /tmp/k8s-webhook-server/serving-certs/tls.crt -keyout /tmp/k8s-webhook-server/serving-certs/tls.key
endif
	go run ./main.go --enable-controllers

##@ Helm Chart Generation

.PHONY: helm-chart
.SILENT: helm-chart
helm-chart: gen-webhook gen-monitor gen-role gen-role-binding gen-serviceaccount ## Run commands generating a valid helm chart.

.PHONY: helm-template
.SILENT: helm-template
helm-template:
	@command -v helm 1>&/dev/null && \
	helm --debug template $(OPERATOR_NAME) deploy/chart -f deploy/chart/values.yaml 1>&/dev/null || echo "helm not installed, skipping.."

.PHONY: gen-webhook
.SILENT: gen-webhook
gen-webhook: ## Run commands generating a valid webhook config.
	[ -f ./config/webhook/manifests.yaml ] && \
	echo "Creating/Updating deploy/chart/templates/webhooks.yaml" && \
	echo "# AUTOGENERATED BY 'make manifests' - DO NOT EDIT!" > ./deploy/chart/templates/webhooks.yaml && \
	$(SED) 's/mutating-webhook-configuration/{{ include "chart.fullname" . }}/g; s/validating-webhook-configuration/{{ include "chart.fullname" . }}/g; s/webhook-service/{{ include "chart.fullname" . }}-webhooks/g; s/namespace: system/namespace: {{ .Release.Namespace }}/g; s@metadata:@metadata:\n  annotations:\n    cert-manager.io/inject-ca-from: {{ .Release.Namespace }}/{{ include "chart.fullname" . }}-server-cert@g' ./config/webhook/manifests.yaml >> ./deploy/chart/templates/webhooks.yaml \
	|| echo No webhook config found, skipping..

.PHONY: gen-serviceaccount
.SILENT: gen-serviceaccount
gen-serviceaccount: ## Run commands generating a valid webhook config.
	[ -f ./config/rbac/service_account.yaml ] && \
	echo "Creating/Updating deploy/chart/templates/serviceaccount.yaml" && \
	echo "# AUTOGENERATED BY 'make manifests' - DO NOT EDIT!" > ./deploy/chart/templates/serviceaccount.yaml && \
	echo '{{- if .Values.serviceAccount.create }}' >> ./deploy/chart/templates/serviceaccount.yaml && \
	$(SED) 's/controller-manager/{{ include "chart.fullname" . }}/g; s/namespace: system/namespace: {{ .Release.Namespace }}/g;' \
		 ./config/rbac/service_account.yaml >> ./deploy/chart/templates/serviceaccount.yaml && \
	echo -e '  labels:\n{{- include "chart.labels" . | nindent 4 }}' >> ./deploy/chart/templates/serviceaccount.yaml && \
	echo -e '{{- end -}}' >> ./deploy/chart/templates/serviceaccount.yaml \
	|| echo No webhook config found, skipping..

.PHONY: gen-role
.SILENT: gen-role
gen-role: ## Run commands generating a valid role config
	[ -f ./config/rbac/role.yaml ] && \
	echo "Creating/Updating deploy/chart/templates/role.yaml" && \
	echo "# AUTOGENERATED BY 'make manifests' - DO NOT EDIT!" > ./deploy/chart/templates/role.yaml && \
	$(SED) 's/manager-role/{{ include "chart.name" . }}/g' ./config/rbac/role.yaml >> ./deploy/chart/templates/role.yaml \
	|| echo No role config found, skipping..

.PHONY: gen-role-binding
.SILENT: gen-role-binding
gen-role-binding: ## Run commands generating a valid role-binding config
	[ -f ./config/rbac/role_binding.yaml ] && \
	echo "Creating/Updating deploy/chart/templates/role_binding.yaml" && \
	echo "# AUTOGENERATED BY 'make manifests' - DO NOT EDIT!" > ./deploy/chart/templates/role_binding.yaml && \
	$(SED) 's/default/{{ include "chart.name" . }}/g; s/manager-rolebinding/{{ include "chart.name" . }}/g; s/manager-role/{{ include "chart.name" . }}/g; s/name\: default/name\: {{ include "chart.name" . }}/g; s/namespace\: system/namespace\: {{ .Release.Namespace }}/g; s/controller-manager/{{ include "chart.fullname" . }}/g' \
		 ./config/rbac/role_binding.yaml >> ./deploy/chart/templates/role_binding.yaml \
	|| echo No role_binding config found, skipping..

.PHONY: gen-monitor
.SILENT: gen-monitor
gen-monitor: ## Run commands generating a valid prometheus monitor config
	[ -f ./config/prometheus/monitor.yaml ] && \
	echo "Creating/Updating deploy/chart/templates/monitor.yaml" && \
	echo "# AUTOGENERATED BY 'make manifests' - DO NOT EDIT!" > ./deploy/chart/templates/monitor.yaml && \
	$(SED) 's/controller-manager-metrics-monitor/{{ include "chart.fullname" . }}/g; 1,/control-plane: controller-manager/ s/control-plane: controller-manager/monitoring.systems.mittwald.cloud\/allowed: "true"/g; 2,/control-plane: controller-manager/ s/control-plane: controller-manager/app.kubernetes.io\/instance: {{ include "chart.name" . }}/g; s/port: https/port: metrics/g; s/namespace\: system/namespace\: {{ .Release.Namespace }}/g' \
		 ./config/prometheus/monitor.yaml >> ./deploy/chart/templates/monitor.yaml \
	|| echo No monitor config found, skipping..
