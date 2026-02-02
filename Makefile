# Build configuration
REGISTRY ?= ghcr.io/den-vasyliev/agentregistry-enterprise
BUILD_DATE ?= $(shell date -u '+%Y-%m-%d')
GIT_COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# Auto-increment version based on git tags
LAST_TAG ?= $(shell git describe --tags --abbrev=0 2>/dev/null)
COMMITS_SINCE_TAG ?= $(shell git rev-list $(LAST_TAG)..HEAD --count 2>/dev/null || echo "0")
# Use VERSION file as fallback if no tags exist
BASE_VERSION ?= $(shell if [ -n "$(LAST_TAG)" ]; then echo $(LAST_TAG) | sed 's/^v//'; else cat VERSION 2>/dev/null || echo "0.2.1"; fi)

# If on a tag, use that tag; otherwise auto-increment patch version
ifeq ($(shell git describe --exact-match --tags 2>/dev/null),)
# Not on a tag - auto-increment patch version
NEXT_VERSION := $(shell echo $(BASE_VERSION) | awk -F. '{$$3=$$3+1; print $$1"."$$2"."$$3}')
VERSION ?= v$(NEXT_VERSION)-$(GIT_COMMIT)
else
# On a tag - use the tag as-is
VERSION ?= $(LAST_TAG)
endif

LDFLAGS := \
	-s -w \
	-X 'github.com/agentregistry-dev/agentregistry/internal/version.Version=$(VERSION)' \
	-X 'github.com/agentregistry-dev/agentregistry/internal/version.GitCommit=$(GIT_COMMIT)' \
	-X 'github.com/agentregistry-dev/agentregistry/internal/version.BuildDate=$(BUILD_DATE)'

LOCALARCH ?= $(shell uname -m | sed 's/x86_64/amd64/' | sed 's/aarch64/arm64/')

.PHONY: help build build-ui build-controller test lint clean image push release version run fmt dev dev-ui ko-controller

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

generate: ## Generate CRD manifests and deepcopy code
	@echo "Generating CRD manifests and code..."
	@command -v controller-gen >/dev/null 2>&1 || go install sigs.k8s.io/controller-tools/cmd/controller-gen@latest
	@controller-gen object:headerFile="hack/boilerplate.go.txt" paths="./api/v1alpha1"
	@controller-gen crd paths="./api/v1alpha1" output:crd:artifacts:config=config/crd
	@echo "✓ Generation complete"

build-ui: ## Build UI static export
	@echo "Building UI..."
	@cd ui && npm install && npm run build:export
	@echo "✓ UI build complete: ui/out/"

build-controller: ## Build controller binary only
	@echo "Building controller binary..."
	@CGO_ENABLED=0 GOOS=linux GOARCH=$(LOCALARCH) go build \
		-ldflags "$(LDFLAGS)" \
		-o bin/controller \
		cmd/controller/main.go
	@echo "✓ Controller build complete: bin/controller"

build: build-ui build-controller ## Build both UI and controller

run: build ## Build and run controller locally
	@echo "Running controller..."
	@./bin/controller

dev: ## Run both controller and UI in development mode
	@echo "Starting controller and UI in development mode..."
	@trap 'kill 0' EXIT; \
	go run -ldflags "$(LDFLAGS)" cmd/controller/main.go & \
	cd ui && NEXT_PUBLIC_API_URL=http://localhost:8080 npm run dev & \
	wait

dev-ui: ## Run Next.js UI dev server (for UI development)
	@echo "Starting Next.js dev server..."
	@cd ui && npm install && npm run dev

dev-down: ## Stop all running controller and UI processes
	@echo "Stopping controller and UI processes..."
	@pkill -f "go run.*cmd/controller/main.go" || true
	@pkill -f "bin/controller" || true
	@lsof -ti:8080 | xargs kill -9 2>/dev/null || true
	@lsof -ti:8081 | xargs kill -9 2>/dev/null || true
	@lsof -ti:8082 | xargs kill -9 2>/dev/null || true
	@pkill -f "next dev" || true
	@lsof -ti:3000 | xargs kill -9 2>/dev/null || true
	@echo "✓ All dev processes stopped"

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

ko-controller: build-ui ## Build and push controller image using ko
	@echo "Building and pushing controller image..."
	@echo "Version: $(VERSION)"
	@echo "Base Version: $(BASE_VERSION)"
	@echo "Git Commit: $(GIT_COMMIT)"
	@KO_DOCKER_REPO=$(REGISTRY) ko build \
		--tags=$(VERSION),$(NEXT_VERSION),latest \
		--bare \
		--image-label org.opencontainers.image.version=$(VERSION) \
		--image-label org.opencontainers.image.revision=$(GIT_COMMIT) \
		cmd/controller/main.go
	@echo "✓ Images pushed:"
	@echo "  $(REGISTRY):$(VERSION)"
	@echo "  $(REGISTRY):$(NEXT_VERSION)"
	@echo "  $(REGISTRY):latest"

image: build ## Build container image locally
	@echo "Building container image..."
	@mkdir -p bin/images
	@KO_DOCKER_REPO=ko.local ko build --oci-layout-path=bin/images/controller cmd/controller/main.go
	@echo "✓ Image built: bin/images/controller"

push: ko-controller ## Alias for ko-controller (build and push to registry)

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
