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
LOCALOS ?= $(shell uname -s | tr '[:upper:]' '[:lower:]')

.PHONY: help build build-ui build-controller test lint clean image push release version run fmt dev demo-stop

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
		go test -coverprofile=coverage.out -covermode=atomic \
		-ldflags "$(LDFLAGS)" -tags=integration \
		./internal/cluster \
		./internal/controller \
		./internal/runtime \
		./internal/runtime/translation/kagent
	@go tool cover -func=coverage.out | grep total:

dev: envtest ## Start interactive dev environment with envtest and sample data
	@echo "═══════════════════════════════════════════════════════════════════"
	@echo "  Starting dev environment..."
	@echo "═══════════════════════════════════════════════════════════════════"
	@echo ""
	@echo "  The kubeconfig will be saved to: /tmp/agentregistry-dev-kubeconfig.yaml"
	@echo ""
	@echo "  After startup, to use kubectl:"
	@echo "    export KUBECONFIG=/tmp/agentregistry-dev-kubeconfig.yaml"
	@echo "    kubectl get mcpservercatalog -A"
	@echo ""
	@echo "  To start the UI in another terminal:"
	@echo "    cd ui && NEXT_PUBLIC_API_URL=http://localhost:8080 npm run dev"
	@echo ""
	@echo "  Press Ctrl+C to stop"
	@echo "═══════════════════════════════════════════════════════════════════"
	@echo ""
	@cd ui && npm install && NEXT_PUBLIC_DISABLE_AUTH=true npm run dev&
	@DEVENV=1 KUBEBUILDER_ASSETS="$$($(LOCALBIN)/setup-envtest use --bin-dir $(LOCALBIN) -p path)" \
		go test -run TestDevEnv -timeout 30m -v ./test/devenv/

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

push: build-ui ## Build and push controller image
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
