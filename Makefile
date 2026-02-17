# Build configuration
REGISTRY ?= ghcr.io/den-vasyliev/agentregistry-inventory
BUILD_DATE ?= $(shell date -u '+%Y-%m-%d')
BUILD_TIMESTAMP ?= $(shell date -u '+%Y%m%d%H%M%S')
GIT_COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# Auto-increment version based on git tags
LAST_TAG ?= $(shell git describe --tags --abbrev=0 2>/dev/null)
COMMITS_SINCE_TAG ?= $(shell git rev-list $(LAST_TAG)..HEAD --count 2>/dev/null || echo "0")
# Fall back to 0.0.0 if no tags exist yet
BASE_VERSION ?= $(shell if [ -n "$(LAST_TAG)" ]; then echo $(LAST_TAG) | sed 's/^v//'; else echo "0.0.0"; fi)

# If on a tag, use that tag; otherwise auto-increment patch version with timestamp
ifeq ($(shell git describe --exact-match --tags 2>/dev/null),)
# Not on a tag - auto-increment patch version with timestamp for unique builds
NEXT_VERSION := $(shell echo $(BASE_VERSION) | awk -F. '{$$3=$$3+1; print $$1"."$$2"."$$3}')
VERSION ?= v$(NEXT_VERSION)-$(GIT_COMMIT)-$(BUILD_TIMESTAMP)
else
# On a tag - use the tag as-is (strip v prefix if present)
VERSION ?= $(shell echo $(LAST_TAG) | sed 's/^v//')
endif

LDFLAGS := \
	-s -w \
	-X 'github.com/agentregistry-dev/agentregistry/internal/version.Version=$(VERSION)' \
	-X 'github.com/agentregistry-dev/agentregistry/internal/version.GitCommit=$(GIT_COMMIT)' \
	-X 'github.com/agentregistry-dev/agentregistry/internal/version.BuildDate=$(BUILD_DATE)'

LOCALARCH ?= $(shell uname -m | sed 's/x86_64/amd64/' | sed 's/aarch64/arm64/')
LOCALOS ?= $(shell uname -s | tr '[:upper:]' '[:lower:]')

.PHONY: help build build-ui build-controller test lint clean image push release version run fmt dev demo-stop generate sync-crds

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

test: ## Run all tests (unit, integration, linting)
	@./test/run_tests.sh

test-unit: ## Run only unit tests
	@./test/run_tests.sh --unit-tests

test-ui: ## Run only UI tests
	@./test/run_tests.sh --ui-tests

test-integration: ## Run only integration tests
	@./test/run_tests.sh --integration

test-security: ## Run security checks
	@./test/run_tests.sh --security

test-benchmarks: ## Run performance benchmarks
	@./test/run_tests.sh --bench

lint: ## Run linting checks
	@./test/run_tests.sh --lint

coverage: ## Generate test coverage report
	@go test -coverprofile=coverage.out -covermode=atomic ./internal/...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

generate: ## Generate CRD manifests and deepcopy code
	@echo "Generating CRD manifests and code..."
	@command -v controller-gen >/dev/null 2>&1 || go install sigs.k8s.io/controller-tools/cmd/controller-gen@latest
	@controller-gen object:headerFile="hack/boilerplate.go.txt" paths="./api/v1alpha1"
	@controller-gen crd paths="./api/v1alpha1" output:crd:artifacts:config=config/crd
	@$(MAKE) sync-crds
	@echo "✓ Generation complete"

sync-crds: ## Sync CRDs from config/crd/ to Helm chart
	@echo "Syncing CRDs to Helm chart..."
	@cp config/crd/agentregistry.dev_*.yaml charts/agentregistry/crds/
	@echo "✓ CRDs synced to charts/agentregistry/crds/"

build-ui: ## Build UI static export
	@echo "Building UI..."
	@cd ui && npm install && npm run build:export
	@echo "✓ UI build complete: ui/out/"

build-controller: prepare-ui-embed ## Build controller binary with embedded UI
	@echo "Building controller binary..."
	@CGO_ENABLED=0 GOOS=$(LOCALOS) GOARCH=$(LOCALARCH) go build \
		-ldflags "$(LDFLAGS)" \
		-o bin/controller \
		cmd/controller/main.go
	@echo "✓ Controller build complete: bin/controller"

build: build-ui build-controller ## Build both UI and controller

run: build ## Build and run controller and ui with your kubeconfig
	@echo "Running controller..."
	@cd ui && npm install && NEXT_PUBLIC_DISABLE_AUTH=true npm run dev&
	@echo "Starting Next.js dev server..."
	@AGENTREGISTRY_DISABLE_AUTH=true ./bin/controller --log-level=debug


demo-stop: ## Stop demo environment
	@echo "Stopping demo environment..."
	@pkill -f "TestDevEnv" 2>/dev/null || true
	@lsof -ti:8080 | xargs kill -9 2>/dev/null || true
	@pkill -f "next dev" 2>/dev/null || true
	@lsof -ti:3000 | xargs kill -9 2>/dev/null || true
	@pkill -f "etcd" 2>/dev/null || true
	@pkill -f "kube-apiserver" 2>/dev/null || true
	@rm -f demo-kubeconfig.yaml /tmp/demo-kubeconfig-*.yaml 2>/dev/null || true
	@echo "Demo stopped"


##@ Testing & Quality

test: envtest ## Run all tests with coverage
	@echo "Running tests..."
	@KUBEBUILDER_ASSETS="$$($(LOCALBIN)/setup-envtest use --bin-dir $(LOCALBIN) -p path)" \
		go test -p 4 -parallel 8 -coverprofile=coverage.out -covermode=atomic \
		-ldflags "$(LDFLAGS)" -tags=integration \
		./internal/cluster \
		./internal/config \
		./internal/controller \
		./internal/httpapi \
		./internal/httpapi/handlers \
		./internal/runtime \
		./internal/runtime/translation/kagent \
		./internal/validation
	@go tool cover -func=coverage.out | grep total:

dev: envtest ## Start interactive dev environment with envtest
	@echo "═══════════════════════════════════════════════════════════════════"
	@echo "  Starting dev environment..."
	@echo "═══════════════════════════════════════════════════════════════════"
	@echo ""
	@echo "  To start the UI in another terminal:"
	@echo "    cd ui && NEXT_PUBLIC_API_URL=http://localhost:8080 npm run dev"
	@echo ""
	@echo "  Press Ctrl+C to stop"
	@echo "═══════════════════════════════════════════════════════════════════"

lint: prepare-ui-embed ## Run linters (gofmt, go vet)
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

prepare-ui-embed: build-ui ## Prepare UI files for embedding (internal target)
	@echo "Preparing UI files for embedding..."
	@rm -rf cmd/controller/ui/out
	@mkdir -p cmd/controller/ui/out
	@cp -r ui/out/* cmd/controller/ui/out/ 2>/dev/null || touch cmd/controller/ui/out/.gitkeep
	@echo "✓ UI files prepared for embedding"

fmt: ## Format Go code
	@echo "Formatting code..."
	@gofmt -w .
	@echo "✓ Code formatted"

##@ Container Images

push: prepare-ui-embed ## Build and push controller image
	@echo "Building and pushing controller image..."
	@echo "Version: $(VERSION)"
	@echo "Base Version: $(BASE_VERSION)"
	@echo "Git Commit: $(GIT_COMMIT)"
	@KO_DOCKER_REPO=$(REGISTRY) ko build \
		--tags=$(VERSION),latest \
		--bare \
		--image-label org.opencontainers.image.version=$(VERSION) \
		--image-label org.opencontainers.image.revision=$(GIT_COMMIT) \
		cmd/controller/main.go
	@echo "✓ Images pushed:"
	@echo "  $(REGISTRY):$(VERSION)"
	@echo "  $(REGISTRY):latest"

image: build ## Build container image locally
	@echo "Building container image..."
	@mkdir -p bin/images
	@KO_DOCKER_REPO=ko.local ko build --oci-layout-path=bin/images/controller cmd/controller/main.go
	@echo "✓ Image built: bin/images/controller"

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
