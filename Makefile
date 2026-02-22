# Kubedial Makefile

# Variables
BINARY_DIR := bin
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

.PHONY: all build build-commander build-dialer clean test test-short test-integration lint docker-build docker-push envtest deps tidy

# Default target
all: build

# Build all binaries
build: build-commander build-dialer

# Build kubecommander
build-commander:
	@echo "Building kubecommander..."
	@mkdir -p $(BINARY_DIR)
	cd kubecommander && $(GOBUILD) $(LDFLAGS) -o ../$(COMMANDER_BINARY) ./cmd/

# Build kubedialer
build-dialer:
	@echo "Building kubedialer..."
	@mkdir -p $(BINARY_DIR)
	cd kubedialer && $(GOBUILD) $(LDFLAGS) -o ../$(DIALER_BINARY) ./cmd/

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -rf $(BINARY_DIR)
	@rm -rf $(ENVTEST_ASSETS_DIR)

# Run all tests (unit + integration)
test: envtest
	@echo "Running all tests..."
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) --bin-dir $(shell pwd)/bin -p path)" \
		$(GOTEST) -v ./...

# Run unit tests only (skip integration tests)
test-short:
	@echo "Running unit tests only..."
	$(GOTEST) -v -short ./...

# Run integration tests only
test-integration: envtest
	@echo "Running integration tests..."
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) --bin-dir $(shell pwd)/bin -p path)" \
		$(GOTEST) -v -run Integration ./...

# Install envtest binary
envtest:
	@echo "Setting up envtest..."
	@mkdir -p $(shell pwd)/bin
	@test -s $(ENVTEST) || GOBIN=$(shell pwd)/bin go install sigs.k8s.io/controller-runtime/tools/setup-envtest@release-0.21
	@$(ENVTEST) use $(ENVTEST_K8S_VERSION) --bin-dir $(shell pwd)/bin -p path > /dev/null

# Run linter
lint:
	@echo "Running linter..."
	$(GOLINT) run ./...

# Tidy dependencies
tidy:
	@echo "Tidying dependencies..."
	cd common && $(GOMOD) tidy
	cd kubecommander && $(GOMOD) tidy
	cd kubedialer && $(GOMOD) tidy

# Download dependencies
deps:
	@echo "Downloading dependencies..."
	cd common && $(GOMOD) download
	cd kubecommander && $(GOMOD) download
	cd kubedialer && $(GOMOD) download

# Docker build targets
docker-build: docker-build-commander docker-build-dialer

docker-build-commander:
	@echo "Building kubecommander Docker image..."
	docker build -f Dockerfile.commander -t $(DOCKER_REGISTRY)/kubecommander:$(VERSION) .
	docker tag $(DOCKER_REGISTRY)/kubecommander:$(VERSION) $(DOCKER_REGISTRY)/kubecommander:latest

docker-build-dialer:
	@echo "Building kubedialer Docker image..."
	docker build -f Dockerfile.dialer -t $(DOCKER_REGISTRY)/kubedialer:$(VERSION) .
	docker tag $(DOCKER_REGISTRY)/kubedialer:$(VERSION) $(DOCKER_REGISTRY)/kubedialer:latest

# Docker push targets
docker-push: docker-push-commander docker-push-dialer

docker-push-commander:
	@echo "Pushing kubecommander Docker image..."
	docker push $(DOCKER_REGISTRY)/kubecommander:$(VERSION)
	docker push $(DOCKER_REGISTRY)/kubecommander:latest

docker-push-dialer:
	@echo "Pushing kubedialer Docker image..."
	docker push $(DOCKER_REGISTRY)/kubedialer:$(VERSION)
	docker push $(DOCKER_REGISTRY)/kubedialer:latest

# Help target
help:
	@echo "Kubedial Makefile targets:"
	@echo "  build             - Build all binaries"
	@echo "  build-commander   - Build kubecommander binary"
	@echo "  build-dialer      - Build kubedialer binary"
	@echo "  clean             - Remove build artifacts"
	@echo "  test              - Run all tests (unit + integration)"
	@echo "  test-short        - Run unit tests only"
	@echo "  test-integration  - Run integration tests only"
	@echo "  lint              - Run golangci-lint"
	@echo "  tidy              - Tidy go.mod files"
	@echo "  deps              - Download dependencies"
	@echo "  docker-build      - Build Docker images"
	@echo "  docker-push       - Push Docker images to registry"
	@echo "  envtest           - Install envtest binary"
	@echo "  help              - Show this help"
