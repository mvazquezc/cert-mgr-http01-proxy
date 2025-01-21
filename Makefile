CONTAINER_TOOL ?= podman
REGISTRY ?= quay.io
REGISTRY_NAMESPACE ?= mavazque
CONTAINER_IMAGE ?= cert-mgr-http01-proxy
CONTAINER_IMAGE_TAG ?= latest

.PHONY: build-image run get-dependencies

build-image: 
	$(info Building Container Image for x86)
	mkdir -p ./out/
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o ./out/cert-mgr-http01-proxy main.go
	${CONTAINER_TOOL} build -t ${REGISTRY}/${REGISTRY_NAMESPACE}/${CONTAINER_IMAGE}:${CONTAINER_IMAGE_TAG} -f Dockerfile .
	
push-image:
	${CONTAINER_TOOL} push ${REGISTRY}/${REGISTRY_NAMESPACE}/${CONTAINER_IMAGE}:${CONTAINER_IMAGE_TAG}

run: get-dependencies
	go run main.go

get-dependencies:
	$(info Downloading dependencies)
	go mod download
