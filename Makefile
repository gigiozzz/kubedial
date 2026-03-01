# Kubedial Makefile

# Variables
BINARY_DIR := bin
CERTS_DIR := certs
COMMANDER_BINARY := $(BINARY_DIR)/kubecommander
DIALER_BINARY := $(BINARY_DIR)/kubedialer
DOCKER_REGISTRY ?= docker.io/gigiozzz
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-X main.Version=$(VERSION)"

# Go settings
GOCMD := go
GOBUILD := $(GOCMD) build
GOTEST := $(GOCMD) test
GOMOD := $(GOCMD) mod
GOLINT := golangci-lint

# envtest settings
ENVTEST_K8S_VERSION := 1.33.0
ENVTEST := $(shell pwd)/bin/setup-envtest
ENVTEST_ASSETS_DIR := $(shell pwd)/bin/k8s

.PHONY: all build build-commander build-dialer clean test test-short test-integration lint docker-build docker-push envtest deps tidy help generate-certs deploy-tls-secrets

.DEFAULT_GOAL := help

##@ General

help: ## Display this help
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Build

build: build-commander build-dialer ## Build all binaries

build-commander: ## Build kubecommander binary
	@echo "Building kubecommander..."
	@mkdir -p $(BINARY_DIR)
	cd kubecommander && $(GOBUILD) $(LDFLAGS) -o ../$(COMMANDER_BINARY) ./cmd/

build-dialer: ## Build kubedialer binary
	@echo "Building kubedialer..."
	@mkdir -p $(BINARY_DIR)
	cd kubedialer && $(GOBUILD) $(LDFLAGS) -o ../$(DIALER_BINARY) ./cmd/

clean: ## Remove build artifacts
	@echo "Cleaning..."
	@rm -rf $(BINARY_DIR)
	@rm -rf $(ENVTEST_ASSETS_DIR)
	@rm -rf $(CERTS_DIR)

##@ Test

test: envtest ## Run all tests (unit + integration)
	@echo "Running all tests..."
	@ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) --bin-dir $(shell pwd)/bin -p path)"; \
	(cd common && KUBEBUILDER_ASSETS="$$ASSETS" $(GOTEST) -v ./...) && \
	(cd kubecommander && KUBEBUILDER_ASSETS="$$ASSETS" $(GOTEST) -v ./...) && \
	(cd kubedialer && KUBEBUILDER_ASSETS="$$ASSETS" $(GOTEST) -v ./...)

test-short: ## Run unit tests only (skip integration)
	@echo "Running unit tests only..."
	cd common && $(GOTEST) -v -short ./...
	cd kubecommander && $(GOTEST) -v -short ./...
	cd kubedialer && $(GOTEST) -v -short ./...

test-integration: envtest ## Run integration tests only
	@echo "Running integration tests..."
	@ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) --bin-dir $(shell pwd)/bin -p path)"; \
	(cd common && KUBEBUILDER_ASSETS="$$ASSETS" $(GOTEST) -v -run Integration ./...) && \
	(cd kubecommander && KUBEBUILDER_ASSETS="$$ASSETS" $(GOTEST) -v -run Integration ./...) && \
	(cd kubedialer && KUBEBUILDER_ASSETS="$$ASSETS" $(GOTEST) -v -run Integration ./...)

envtest:
	@echo "Setting up envtest..."
	@mkdir -p $(shell pwd)/bin
	@test -s $(ENVTEST) || GOBIN=$(shell pwd)/bin go install sigs.k8s.io/controller-runtime/tools/setup-envtest@release-0.21
	@$(ENVTEST) use $(ENVTEST_K8S_VERSION) --bin-dir $(shell pwd)/bin -p path > /dev/null

##@ Docker

docker-build: docker-build-commander docker-build-dialer ## Build Docker images

docker-build-commander:
	@echo "Building kubecommander Docker image..."
	docker build -f Dockerfile.commander -t $(DOCKER_REGISTRY)/kubecommander:$(VERSION) .
	docker tag $(DOCKER_REGISTRY)/kubecommander:$(VERSION) $(DOCKER_REGISTRY)/kubecommander:latest

docker-build-dialer:
	@echo "Building kubedialer Docker image..."
	docker build -f Dockerfile.dialer -t $(DOCKER_REGISTRY)/kubedialer:$(VERSION) .
	docker tag $(DOCKER_REGISTRY)/kubedialer:$(VERSION) $(DOCKER_REGISTRY)/kubedialer:latest

docker-push: docker-push-commander docker-push-dialer ## Push Docker images to registry

docker-push-commander:
	@echo "Pushing kubecommander Docker image..."
	docker push $(DOCKER_REGISTRY)/kubecommander:$(VERSION)
	docker push $(DOCKER_REGISTRY)/kubecommander:latest

docker-push-dialer:
	@echo "Pushing kubedialer Docker image..."
	docker push $(DOCKER_REGISTRY)/kubedialer:$(VERSION)
	docker push $(DOCKER_REGISTRY)/kubedialer:latest

##@ Development

lint: ## Run golangci-lint
	@echo "Running linter..."
	cd common && $(GOLINT) run ./...
	cd kubecommander && $(GOLINT) run ./...
	cd kubedialer && $(GOLINT) run ./...

tidy: ## Tidy go.mod files
	@echo "Tidying dependencies..."
	cd common && $(GOMOD) tidy
	cd kubecommander && $(GOMOD) tidy
	cd kubedialer && $(GOMOD) tidy

deps: ## Download dependencies
	@echo "Downloading dependencies..."
	cd common && $(GOMOD) download
	cd kubecommander && $(GOMOD) download
	cd kubedialer && $(GOMOD) download

##@ Security

generate-certs: ## Generate CA, server, and client TLS certificates in certs/
	@echo "Generating TLS certificates..."
	@mkdir -p $(CERTS_DIR)
	@# Generate CA key and self-signed cert (4096-bit RSA, 10 years)
	openssl req -x509 -newkey rsa:4096 -keyout $(CERTS_DIR)/ca.key -out $(CERTS_DIR)/ca.crt \
		-days 3650 -nodes -subj "/CN=kubedial-ca"
	@# Generate server key and CSR
	openssl req -newkey rsa:2048 -keyout $(CERTS_DIR)/server.key -out $(CERTS_DIR)/server.csr \
		-nodes -subj "/CN=kubecommander"
	@# Sign server cert with CA (1 year, SANs)
	openssl x509 -req -in $(CERTS_DIR)/server.csr -CA $(CERTS_DIR)/ca.crt -CAkey $(CERTS_DIR)/ca.key \
		-CAcreateserial -out $(CERTS_DIR)/server.crt -days 365 \
		-extfile <(printf "subjectAltName=DNS:kubecommander,DNS:kubecommander.kubedial.svc,DNS:kubecommander.kubedial.svc.cluster.local,DNS:localhost,IP:127.0.0.1")
	@# Generate client key and CSR
	openssl req -newkey rsa:2048 -keyout $(CERTS_DIR)/client.key -out $(CERTS_DIR)/client.csr \
		-nodes -subj "/CN=kubedialer"
	@# Sign client cert with CA (1 year, clientAuth)
	openssl x509 -req -in $(CERTS_DIR)/client.csr -CA $(CERTS_DIR)/ca.crt -CAkey $(CERTS_DIR)/ca.key \
		-CAcreateserial -out $(CERTS_DIR)/client.crt -days 365 \
		-extfile <(printf "extendedKeyUsage=clientAuth")
	@# Cleanup CSR and SRL files
	@rm -f $(CERTS_DIR)/*.csr $(CERTS_DIR)/*.srl
	@echo "Certificates generated in $(CERTS_DIR)/"

deploy-tls-secrets: ## Generate deploy/tls-secrets.yaml from certs/ (requires generate-certs first)
	@echo "Generating TLS secrets manifest..."
	kubectl create secret generic kubecommander-tls \
		--from-file=ca.crt=$(CERTS_DIR)/ca.crt \
		--from-file=server.crt=$(CERTS_DIR)/server.crt \
		--from-file=server.key=$(CERTS_DIR)/server.key \
		--dry-run=client -o yaml > deploy/tls-secrets.yaml
	kubectl create secret generic kubedialer-tls \
		--from-file=ca.crt=$(CERTS_DIR)/ca.crt \
		--from-file=client.crt=$(CERTS_DIR)/client.crt \
		--from-file=client.key=$(CERTS_DIR)/client.key \
		--dry-run=client -o yaml >> deploy/tls-secrets.yaml
	@echo "TLS secrets manifest written to deploy/tls-secrets.yaml"
