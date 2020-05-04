.PHONY: test

test:
	go test -v ./...

sdk-build:
	operator-sdk build harbor-operator:dev
	docker rmi harbor-operator:dev

sdk-gen:
	operator-sdk generate k8s && \
	operator-sdk generate crds