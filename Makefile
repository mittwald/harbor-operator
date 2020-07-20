SHELL=/bin/bash
NAMESPACE=default
KUBECONFIG=/tmp/kubeconfig

.PHONY: install
install: ## Install all resources (RBAC and Operator)
	@echo ....... Applying Roles and Service Account .......
	kubectl apply -f deploy/role.yaml -n ${NAMESPACE}
	kubectl apply -f deploy/role_binding.yaml  -n ${NAMESPACE}
	kubectl apply -f deploy/service_account.yaml  -n ${NAMESPACE}
	@echo ....... Applying Operator .......
	kubectl apply -f deploy/operator.yaml -n ${NAMESPACE}

.PHONY: uninstall
uninstall: ## Uninstall all installations performed in the $ make install
	@echo ....... Uninstalling .......
	@echo ....... Deleting Rules and Service Account .......
	kubectl delete -f deploy/role.yaml -n ${NAMESPACE}
	kubectl delete -f deploy/role_binding.yaml -n ${NAMESPACE}
	kubectl delete -f deploy/service_account.yaml -n ${NAMESPACE}
	@echo ....... Deleting Operator .......
	kubectl delete -f deploy/operator.yaml -n ${NAMESPACE}

.PHONY: test
test:
	@echo go test
	go test ./... -v

.PHONY: fmt
fmt:
	@echo go fmt
	go fmt $$(go list ./...)

.PHONY: build
build:
	operator-sdk build --go-build-args "-ldflags -X=version.Version=${SECRET_OPERATOR_VERSION}" ${DOCKER_IMAGE}
	@exit $(.SHELLSTATUS)

.PHONY: mock
mock:
	@echo generating mocked k8s runtime client via
	@echo sigs.k8s.io/controller-runtime@v0.5.2/pkg/client.Client
	curl -sO https://raw.githubusercontent.com/kubernetes-sigs/controller-runtime/v0.5.2/pkg/client/interfaces.go
	mockery --quiet --dir $(PWD) \
	--name Client --structname MockClient --filename=runtime_client_mock.go \
	--output "$(PWD)/pkg/internal/mocks"
	rm $(PWD)/interfaces.go