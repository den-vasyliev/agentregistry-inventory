# Build configuration
REGISTRY ?= ghcr.io/den-vasyliev/agentregistry-enterprise
BUILD_DATE ?= $(shell date -u '+%Y-%m-%d')
GIT_COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BASE_VERSION ?= $(shell cat VERSION 2>/dev/null || echo "0.1.0")
VERSION ?= v$(BASE_VERSION)-$(GIT_COMMIT)

LDFLAGS := \
	-s -w \
	-X 'github.com/agentregistry-dev/agentregistry/internal/version.Version=$(VERSION)' \
	-X 'github.com/agentregistry-dev/agentregistry/internal/version.GitCommit=$(GIT_COMMIT)' \
	-X 'github.com/agentregistry-dev/agentregistry/internal/version.BuildDate=$(BUILD_DATE)'

LOCALARCH ?= $(shell uname -m | sed 's/x86_64/amd64/' | sed 's/aarch64/arm64/')

.PHONY: help build test lint clean image push release version run fmt dev-ui

##@ General

help: ## Display this help
	@echo "Agent Registry Enterprise - Controller"
	@echo ""
	@echo "Usage: make <target>"
	@echo ""
	@echo "Targets:"
	@awk 'BEGIN {FS = ":.*##"; printf "\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  %-15s %s\n", $$1, $$2 } /^##@/ { printf "\n%s\n", substr($$0, 5) } ' $(MAKEFILE_LIST)
	@echo ""
	@echo "Current version: $(VERSION)"

version: ## Show version information
	@echo "Version:    $(VERSION)"
	@echo "Commit:     $(GIT_COMMIT)"
	@echo "Build Date: $(BUILD_DATE)"
	@echo "Registry:   $(REGISTRY)"

##@ Development

build: ## Build controller binary (includes embedded UI)
	@echo "Building UI..."
	@cd ui && npm install && npm run build:export
	@echo "Building controller binary..."
	@CGO_ENABLED=0 GOOS=linux GOARCH=$(LOCALARCH) go build \
		-ldflags "$(LDFLAGS)" \
		-o bin/controller \
		cmd/controller/main.go
	@echo "✓ Build complete: bin/controller"

run: build ## Build and run controller locally
	@echo "Running controller..."
	@./bin/controller

dev: ## Run controller in development mode with live reload
	@echo "Starting controller in development mode..."
	@echo "To run UI dev server in parallel, open another terminal and run: make dev-ui"
	@go run -ldflags "$(LDFLAGS)" cmd/controller/main.go

dev-ui: ## Run Next.js UI dev server (for UI development)
	@echo "Starting Next.js dev server..."
	@cd ui && npm install && npm run dev

##@ Testing & Quality

test: envtest ## Run all tests with coverage
	@echo "Running tests..."
	@command -v gotestsum >/dev/null 2>&1 || go install gotest.tools/gotestsum@latest
	@command -v gocover-cobertura >/dev/null 2>&1 || go install github.com/boumenot/gocover-cobertura@latest
	@KUBEBUILDER_ASSETS="$$($(LOCALBIN)/setup-envtest use --bin-dir $(LOCALBIN) -p path)" \
		gotestsum --junitfile report.xml --format testname -- \
		-coverprofile=coverage.out -covermode=count \
		-ldflags "$(LDFLAGS)" -tags=integration \
		./...
	@go tool cover -func=coverage.out | grep total:
	@gocover-cobertura < coverage.out > coverage.xml

lint: ## Run linters (gofmt, go vet)
	@echo "Running gofmt..."
	@gofmt_output=$$(gofmt -l .); \
	if [ -n "$$gofmt_output" ]; then \
		echo "Error: Files not formatted:"; \
		echo "$$gofmt_output"; \
		echo "Run 'make fmt' to fix"; \
		exit 1; \
	fi
	@echo "✓ gofmt passed"
	@echo "Running go vet..."
	@go vet ./...
	@echo "✓ go vet passed"

fmt: ## Format Go code
	@echo "Formatting code..."
	@gofmt -w .
	@echo "✓ Code formatted"

##@ Container Images

image: build ## Build container image locally
	@echo "Building container image..."
	@mkdir -p bin/images
	@KO_DOCKER_REPO=ko.local ko build --oci-layout-path=bin/images/controller cmd/controller/main.go
	@echo "✓ Image built: bin/images/controller"

push: ## Build and push container image to registry
	@echo "Building and pushing image to $(REGISTRY)..."
	@KO_DOCKER_REPO=$(REGISTRY) ko build --tags=$(BASE_VERSION),latest --bare cmd/controller/main.go
	@echo "✓ Images pushed:"
	@echo "  $(REGISTRY):$(BASE_VERSION)"
	@echo "  $(REGISTRY):latest"

##@ Release

release: clean test lint build push ## Full release build (clean, test, lint, build, push)
	@echo ""
	@echo "✓ Release v$(BASE_VERSION) complete!"
	@echo "  Commit: $(GIT_COMMIT)"
	@echo "  Binary: bin/controller"
	@echo "  Images: $(REGISTRY):$(BASE_VERSION), $(REGISTRY):latest"

##@ Cleanup

clean: ## Clean all build artifacts
	@echo "Cleaning build artifacts..."
	@rm -rf bin/ ui/out/ ui/.next/ ui/node_modules/
	@go clean
	@echo "✓ Cleaned"

##@ Dependencies

LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	@mkdir -p $(LOCALBIN)

GOIMPORT = $(LOCALBIN)/goimports
ENVTEST = $(LOCALBIN)/setup-envtest

.PHONY: goimports envtest
goimports: $(GOIMPORT)
$(GOIMPORT): $(LOCALBIN)
	@GOBIN=$(LOCALBIN) go install golang.org/x/tools/cmd/goimports@v0.41

envtest: $(ENVTEST)
$(ENVTEST): $(LOCALBIN)
	@GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-runtime/tools/setup-envtest@release-0.19
