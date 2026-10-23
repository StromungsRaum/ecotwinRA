# Copyright 2025 The Platform Mesh Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

SHELL := /usr/bin/env bash

# Tool installation
GO_INSTALL = ./hack/go-install.sh

TOOLS_DIR = hack/tools
GOBIN_DIR := $(abspath $(TOOLS_DIR))

# controller-gen
CONTROLLER_GEN_VER := v0.16.5
CONTROLLER_GEN_BIN := controller-gen
CONTROLLER_GEN := $(GOBIN_DIR)/$(CONTROLLER_GEN_BIN)-$(CONTROLLER_GEN_VER)
export CONTROLLER_GEN

# apigen - generates APIResourceSchemas from CRDs
APIGEN_VER := v0.28.1-0.20251209130449-436a0347809b
APIGEN_BIN := apigen
APIGEN := $(GOBIN_DIR)/$(APIGEN_BIN)-$(APIGEN_VER)
export APIGEN

# Go parameters
GOCMD = go
GOBUILD = $(GOCMD) build
GORUN = $(GOCMD) run
GOMOD = $(GOCMD) mod
GOFMT = $(GOCMD) fmt

# Binary names
BINARY_NAME = ianus
INIT_BINARY_NAME = ianus-init

# Injection Molding Service provider binary names
IM_BINARY_NAME = injectionmolding
IM_INIT_BINARY_NAME = injectionmolding-init

# Build directory
BUILD_DIR = bin

# Image parameters
IMAGE_REGISTRY ?= ghcr.io/platform-mesh
IMAGE_NAME ?= ianus-controller
IMAGE_TAG ?= dev
IMAGE ?= $(IMAGE_REGISTRY)/$(IMAGE_NAME):$(IMAGE_TAG)

# Portal image parameters
PORTAL_IMAGE_NAME ?= ianus-portal
PORTAL_IMAGE ?= $(IMAGE_REGISTRY)/$(PORTAL_IMAGE_NAME):$(IMAGE_TAG)
PORTAL_PORT ?= 4200

# IANUS proxy api image parameters
API_IMAGE_NAME ?= ianus-api
API_IMAGE ?= $(IMAGE_REGISTRY)/$(API_IMAGE_NAME):$(IMAGE_TAG)
API_PORT ?= 8000

# Injection Molding Service provider image parameters
IM_IMAGE_NAME ?= injectionmolding-controller
IM_IMAGE ?= $(IMAGE_REGISTRY)/$(IM_IMAGE_NAME):$(IMAGE_TAG)

.PHONY: all
all: build

## build: Build all binaries
.PHONY: build
build: build-operator build-init

## build-operator: Build the ianus operator binary
.PHONY: build-operator
build-operator: fmt vet
	$(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/ianus/...

## build-init: Build the init/bootstrap binary
.PHONY: build-init
build-init: fmt vet
	$(GOBUILD) -o $(BUILD_DIR)/$(INIT_BINARY_NAME) ./cmd/init/...

## run: Run the ianus operator locally
.PHONY: run
run: fmt vet
	$(GORUN) ./cmd/ianus/main.go --endpointslice=ianus.platform-mesh.io

## init: Bootstrap provider resources into workspace (requires KUBECONFIG, optional HOST_OVERRIDE)
HOST_OVERRIDE ?=
.PHONY: init
init: build-init
	$(BUILD_DIR)/$(INIT_BINARY_NAME) $(if $(HOST_OVERRIDE),--host-override=$(HOST_OVERRIDE))

## build-operator-injectionmolding: Build the injectionmolding operator binary
.PHONY: build-operator-injectionmolding
build-operator-injectionmolding: fmt vet
	$(GOBUILD) -o $(BUILD_DIR)/$(IM_BINARY_NAME) ./cmd/injectionmolding/...

## build-init-injectionmolding: Build the injectionmolding init/bootstrap binary
.PHONY: build-init-injectionmolding
build-init-injectionmolding: fmt vet
	$(GOBUILD) -o $(BUILD_DIR)/$(IM_INIT_BINARY_NAME) ./cmd/injectionmolding-init/...

## build-injectionmolding: Build both injectionmolding binaries (operator + init)
.PHONY: build-injectionmolding
build-injectionmolding: build-operator-injectionmolding build-init-injectionmolding

## run-injectionmolding: Run the injectionmolding operator locally
.PHONY: run-injectionmolding
run-injectionmolding: fmt vet
	$(GORUN) ./cmd/injectionmolding/main.go --endpointslice=injectionmolding.platform-mesh.io

## init-injectionmolding: Bootstrap injectionmolding provider resources into workspace (requires KUBECONFIG, optional HOST_OVERRIDE)
.PHONY: init-injectionmolding
init-injectionmolding: build-init-injectionmolding
	$(BUILD_DIR)/$(IM_INIT_BINARY_NAME) $(if $(HOST_OVERRIDE),--host-override=$(HOST_OVERRIDE))

## generate: Generate code (deepcopy, etc.) and kcp resources for the ianus provider
.PHONY: generate
generate: $(CONTROLLER_GEN) manifests apiresourceschemas
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./apis/ianus/..."

## manifests: Generate CRD manifests for the ianus provider
.PHONY: manifests
manifests: $(CONTROLLER_GEN)
	$(CONTROLLER_GEN) crd paths="./apis/ianus/..." output:crd:artifacts:config=config/crds

## apiresourceschemas: Generate APIResourceSchemas from CRDs for the ianus provider
.PHONY: apiresourceschemas
apiresourceschemas: manifests $(APIGEN)
	$(APIGEN) --input-dir=config/crds --output-dir=config/kcp

## generate-injectionmolding: Generate code (deepcopy, etc.) and kcp resources for the injectionmolding provider
.PHONY: generate-injectionmolding
generate-injectionmolding: $(CONTROLLER_GEN) manifests-injectionmolding apiresourceschemas-injectionmolding
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./apis/injectionmolding/..."

## manifests-injectionmolding: Generate CRD manifests for the injectionmolding provider
.PHONY: manifests-injectionmolding
manifests-injectionmolding: $(CONTROLLER_GEN)
	$(CONTROLLER_GEN) crd paths="./apis/injectionmolding/..." output:crd:artifacts:config=config/crds-injectionmolding

## apiresourceschemas-injectionmolding: Generate APIResourceSchemas from CRDs for the injectionmolding provider
.PHONY: apiresourceschemas-injectionmolding
apiresourceschemas-injectionmolding: manifests-injectionmolding $(APIGEN)
	$(APIGEN) --input-dir=config/crds-injectionmolding --output-dir=config/kcp-injectionmolding

## fmt: Run go fmt
.PHONY: fmt
fmt:
	$(GOFMT) ./...

## vet: Run go vet
.PHONY: vet
vet:
	$(GOCMD) vet ./...

## tidy: Run go mod tidy
.PHONY: tidy
tidy:
	$(GOMOD) tidy

## image-build: Build controller container image locally
.PHONY: image-build
image-build:
	docker build -t $(IMAGE) -f deploy/Dockerfile .

## image-push: Push controller container image to registry
.PHONY: image-push
image-push: image-build
	docker push $(IMAGE)

## portal-image-build: Build portal container image locally
.PHONY: portal-image-build
portal-image-build:
	docker build -t $(PORTAL_IMAGE) -f deploy/portal.Dockerfile .

## portal-image-push: Push portal container image to registry
.PHONY: portal-image-push
portal-image-push: portal-image-build
	docker push $(PORTAL_IMAGE)

## api-image-build: Build IANUS proxy container image locally
.PHONY: api-image-build
api-image-build:
	docker build -t $(API_IMAGE) -f deploy/api.Dockerfile .

## api-image-push: Push IANUS proxy container image to registry
.PHONY: api-image-push
api-image-push: api-image-build
	docker push $(API_IMAGE)

## image-build-injectionmolding: Build injectionmolding controller container image locally
.PHONY: image-build-injectionmolding
image-build-injectionmolding:
	docker build -t $(IM_IMAGE) -f deploy/injectionmolding.Dockerfile .

## image-push-injectionmolding: Push injectionmolding controller container image to registry
.PHONY: image-push-injectionmolding
image-push-injectionmolding: image-build-injectionmolding
	docker push $(IM_IMAGE)

## images: Build all container images
.PHONY: images
images: image-build portal-image-build image-build-injectionmolding

## images-push: Push all container images
.PHONY: images-push
images-push: image-push portal-image-push image-push-injectionmolding

# Kind cluster parameters
KIND_CLUSTER ?= platform-mesh

## kind-load: Load controller image into kind cluster
.PHONY: kind-load
kind-load: image-build
	kind load docker-image $(IMAGE) --name $(KIND_CLUSTER)

## kind-load-portal: Load portal image into kind cluster
.PHONY: kind-load-portal
kind-load-portal: portal-image-build
	kind load docker-image $(PORTAL_IMAGE) --name $(KIND_CLUSTER)

## kind-load-api: Load api image into kind cluster
.PHONY: kind-load-api
kind-load-api: api-image-build
	kind load docker-image $(API_IMAGE) --name $(KIND_CLUSTER)

## kind-load-injectionmolding: Load injectionmolding controller image into kind cluster
.PHONY: kind-load-injectionmolding
kind-load-injectionmolding: image-build-injectionmolding
	kind load docker-image $(IM_IMAGE) --name $(KIND_CLUSTER)

## kind-load-all: Load all images into kind cluster
.PHONY: kind-load-all
kind-load-all: kind-load kind-load-portal kind-load-injectionmolding

## portal-run: Run portal container locally (accessible at http://localhost:$(PORTAL_PORT))
.PHONY: portal-run
portal-run:
	docker run --rm -p $(PORTAL_PORT):8080 $(PORTAL_IMAGE)

## portal-run-detached: Run portal container in background
.PHONY: portal-run-detached
portal-run-detached:
	docker run -d --rm --name ianus-portal -p $(PORTAL_PORT):80 $(PORTAL_IMAGE)
	@echo "Portal running at http://localhost:$(PORTAL_PORT)"
	@echo "Stop with: docker stop ianus-portal"

## portal-stop: Stop the portal container
.PHONY: portal-stop
portal-stop:
	docker stop ianus-portal

## portal-run: Run portal container locally (accessible at http://localhost:$(PORTAL_PORT))
.PHONY: api-run
api-run:
	docker run --env-file .env --rm -p $(API_PORT):8000 $(API_IMAGE)

## tools: Install all required tools
.PHONY: tools
tools: $(CONTROLLER_GEN) $(APIGEN)

## help: Display this help
.PHONY: help
help:
	@echo "Usage:"
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' | sed -e 's/^/ /'

# Tool installation targets
$(CONTROLLER_GEN):
	GOBIN=$(GOBIN_DIR) $(GO_INSTALL) sigs.k8s.io/controller-tools/cmd/$(CONTROLLER_GEN_BIN) $(CONTROLLER_GEN_BIN) $(CONTROLLER_GEN_VER)

$(APIGEN):
	GOBIN=$(GOBIN_DIR) $(GO_INSTALL) github.com/kcp-dev/sdk/cmd/$(APIGEN_BIN) $(APIGEN_BIN) $(APIGEN_VER)
