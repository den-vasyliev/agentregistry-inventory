# Image configuration
KO_REGISTRY ?= localhost:5001
BASE_IMAGE_REGISTRY ?= ghcr.io
KO_DOCKER_REPO ?= agentregistry-dev/agentregistry
KO_PRESERVE_IMPORT_PATHS ?= true
BUILD_DATE ?= $(shell date -u '+%Y-%m-%d')
GIT_COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BASE_VERSION ?= $(shell cat VERSION 2>/dev/null || echo "0.1.0")
VERSION ?= v$(BASE_VERSION)-$(GIT_COMMIT)

LDFLAGS := \
	-s -w \
	-X 'github.com/agentregistry-dev/agentregistry/internal/version.Version=$(VERSION)' \
	-X 'github.com/agentregistry-dev/agentregistry/internal/version.GitCommit=$(GIT_COMMIT)' \
	-X 'github.com/agentregistry-dev/agentregistry/internal/version.BuildDate=$(BUILD_DATE)' \
	-X 'github.com/agentregistry-dev/agentregistry/internal/version.DockerRegistry=$(DOCKER_REGISTRY)'

# Local architecture detection to build for the current platform
LOCALARCH ?= $(shell uname -m | sed 's/x86_64/amd64/' | sed 's/aarch64/arm64/')

.PHONY: help install-ui build-ui clean-ui build-controller build dev dev-ui dev-controller test clean fmt lint all ko-controller ko-build ko-tag-as-dev

# Default target
help:
	@echo "Available targets:"
	@echo "  install-ui           - Install UI dependencies"
	@echo "  build-ui             - Build the Next.js UI"
	@echo "  clean-ui             - Clean UI build artifacts"
	@echo "  build-controller     - Build the controller binary"
	@echo "  build                - Build UI and controller"
	@echo "  dev                  - Run controller + UI in development mode"
	@echo "  dev-ui               - Run Next.js in development mode only"
	@echo "  dev-controller       - Run controller in development mode only"
	@echo "  test                 - Run Go tests"
	@echo "  clean                - Clean all build artifacts"
	@echo "  all                  - Clean and build everything"
	@echo "  fmt                  - Run the formatter"
	@echo "  lint                 - Run the linter"
	@echo "  ko-build             - Build controller image with ko"
	@echo "  ko-controller        - Build controller image with ko"
	@echo "  ko-tag-as-dev        - Tag images as dev"
	@echo "  version              - Show current version info"
	@echo ""
	@echo "Current version: $(VERSION)"

version:
	@echo "Version:    $(VERSION)"
	@echo "Git Commit: $(GIT_COMMIT)"
	@echo "Build Date: $(BUILD_DATE)"

# Bump patch version (0.1.0 -> 0.1.1)
bump-patch:
	@current=$$(cat VERSION); \
	major=$$(echo $$current | cut -d. -f1); \
	minor=$$(echo $$current | cut -d. -f2); \
	patch=$$(echo $$current | cut -d. -f3); \
	new="$$major.$$minor.$$((patch + 1))"; \
	echo $$new > VERSION; \
	echo "Bumped version: $$current -> $$new"

# Bump minor version (0.1.0 -> 0.2.0)
bump-minor:
	@current=$$(cat VERSION); \
	major=$$(echo $$current | cut -d. -f1); \
	minor=$$(echo $$current | cut -d. -f2); \
	new="$$major.$$((minor + 1)).0"; \
	echo $$new > VERSION; \
	echo "Bumped version: $$current -> $$new"

# Bump major version (0.1.0 -> 1.0.0)
bump-major:
	@current=$$(cat VERSION); \
	major=$$(echo $$current | cut -d. -f1); \
	new="$$((major + 1)).0.0"; \
	echo $$new > VERSION; \
	echo "Bumped version: $$current -> $$new"

# Install UI dependencies
install-ui:
	@echo "Installing UI dependencies..."
	cd ui && npm install

# Build the Next.js UI (static export for standalone deployment)
build-ui: install-ui
	@echo "Building Next.js UI for static export..."
	cd ui && npm run build:export
	@echo "UI built successfully to ui/out/"

# Clean UI build artifacts
clean-ui:
	@echo "Cleaning UI build artifacts..."
	git clean -xdf ./ui/out/
	git clean -xdf ./ui/.next/
	@echo "UI artifacts cleaned"

# Build the controller binary
build-controller:
	@echo "Building controller..."
	@echo "Downloading Go dependencies..."
	go mod download
	@echo "Building binary..."
	go build -ldflags "$(LDFLAGS)" \
		-o bin/controller cmd/controller/main.go
	@echo "Binary built successfully: bin/controller"

# Build everything (UI + controller)
build: build-ui build-controller
	@echo "Build complete!"
	@echo "Controller: ./bin/controller"
	@echo "UI: ./ui/out/"

# Run Next.js in development mode
dev-ui:
	@echo "Starting Next.js development server..."
	cd ui && npm run dev

# Run controller in development mode
dev-controller: build-controller
	@echo "Starting controller with HTTP API..."
	./bin/controller --enable-http-api=true

# Run both UI and controller for development (parallel)
dev: build-controller
	@echo "Starting development environment..."
	@echo "  Controller API: http://localhost:8080"
	@echo "  UI: http://localhost:3000"
	@echo ""
	@echo "Press Ctrl+C to stop both services"
	@trap 'kill 0' EXIT; \
		./bin/controller --enable-http-api=true & \
		cd ui && npm run dev

# Run Go tests
test: envtest
	@echo "Running Go tests..."
	go install gotest.tools/gotestsum@latest
	go install github.com/boumenot/gocover-cobertura@latest
	KUBEBUILDER_ASSETS="$$($(ENVTEST) use --bin-dir $(LOCALBIN) -p path)" \
		gotestsum --junitfile report.xml --format testname -- \
		-coverprofile=coverage.out -covermode=count \
		-ldflags "$(LDFLAGS)" -tags=integration \
		./... $(TEST_ARGS)
	go tool cover -func=coverage.out
	gocover-cobertura < coverage.out > coverage.xml
	@echo "Coverage report: coverage.out, coverage.xml"

# Clean all build artifacts
clean: clean-ui
	@echo "Cleaning Go build artifacts..."
	rm -rf bin/
	go clean
	@echo "All artifacts cleaned"

# Clean and build everything
all: clean build 
	@echo "Clean build complete!"

# Run controller locally (for development)
# Usage: make run-controller KUBECONFIG=~/.kube/config
run-controller: build-controller
	@echo "Running controller locally..."
	@echo "Using kubeconfig: $(KUBECONFIG)"
	./bin/controller --kubeconfig="$(KUBECONFIG)"


fmt: goimports
	$(GOIMPORT) -w .
	@echo "✓ Formatted code"


# Build images using ko
.PHONY: ko-build ko-controller

KO_REGISTRY_PREFIX ?= $(KO_REGISTRY)/$(KO_DOCKER_REPO)

ko-controller:
	@echo "Building controller image with ko..."
	mkdir -p bin/images
	KO_DOCKER_REPO=ko.local ko build --oci-layout-path=bin/images/controller cmd/controller/main.go
	@echo "✓ Controller image built: bin/images/controller"

ko-build: ko-controller
	@echo "✓ All images built with ko"

ko-tag-as-dev:
	@echo "Tagging controller image as dev with ko..."
	KO_DOCKER_REPO=$(KO_REGISTRY_PREFIX)/controller ko build --tags=dev,latest cmd/controller/main.go
	@echo "✓ Controller image tagged as dev"

.PHONY: lint
lint: golangci-lint ## Run golangci-lint linter
	$(GOLANGCI_LINT) run

.PHONY: lint-fix
lint-fix: golangci-lint ## Run golangci-lint linter and perform fixes
	$(GOLANGCI_LINT) run --fix

.PHONY: lint-config
lint-config: golangci-lint ## Verify golangci-lint linter configuration
	$(GOLANGCI_LINT) config verify

##@ Dependencies

## Location to install dependencies to
LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)


GOIMPORT = $(LOCALBIN)/goimports
GOIMPORT_VERSION ?= v0.41

GOLANGCI_LINT = $(LOCALBIN)/golangci-lint
GOLANGCI_LINT_VERSION ?= v2.8.0

ENVTEST = $(LOCALBIN)/setup-envtest
ENVTEST_VERSION ?= release-0.19

.PHONY: goimports
goimports: $(GOIMPORT) ## Download goimports locally if necessary.
$(GOIMPORT): $(LOCALBIN)
	$(call go-install-tool,$(GOIMPORT),golang.org/x/tools/cmd/goimports,$(GOIMPORT_VERSION))

.PHONY: golangci-lint
golangci-lint: $(GOLANGCI_LINT) ## Download golangci-lint locally if necessary.
$(GOLANGCI_LINT): $(LOCALBIN)
	$(call go-install-tool,$(GOLANGCI_LINT),github.com/golangci/golangci-lint/v2/cmd/golangci-lint,$(GOLANGCI_LINT_VERSION))

.PHONY: envtest
envtest: $(ENVTEST) ## Download setup-envtest locally if necessary.
$(ENVTEST): $(LOCALBIN)
	$(call go-install-tool,$(ENVTEST),sigs.k8s.io/controller-runtime/tools/setup-envtest,$(ENVTEST_VERSION))

.PHONY: test-controller
test-controller: envtest ## Run controller tests with envtest
	@echo "Running controller tests with envtest..."
	go install gotest.tools/gotestsum@latest
	go install github.com/boumenot/gocover-cobertura@latest
	KUBEBUILDER_ASSETS="$$($(ENVTEST) use --bin-dir $(LOCALBIN) -p path)" \
		gotestsum --junitfile report.xml --format testname -- \
		-coverprofile=coverage.out -covermode=count \
		./internal/controller/... $(TEST_ARGS)
	go tool cover -func=coverage.out
	gocover-cobertura < coverage.out > coverage.xml
	@echo "Coverage report: coverage.out, coverage.xml"

.PHONY: test-coverage
test-coverage: envtest ## Run all tests with coverage
	@echo "Running all tests with coverage..."
	go install gotest.tools/gotestsum@latest
	KUBEBUILDER_ASSETS="$$($(ENVTEST) use --bin-dir $(LOCALBIN) -p path)" \
		gotestsum --junitfile report.xml --format testname -- \
		-coverprofile=coverage.out -covermode=count \
		./internal/controller/... ./internal/httpapi/... $(TEST_ARGS)
	go tool cover -func=coverage.out
	@echo "Coverage report: coverage.out"

.PHONY: coverage
coverage: ## View test coverage in browser
	@if [ ! -f coverage.out ]; then \
		echo "Coverage file not found. Running tests first..."; \
		$(MAKE) test; \
	fi
	go tool cover -html=coverage.out

# go-install-tool will 'go install' any package with custom target and name of binary, if it doesn't exist
# $1 - target path with name of binary
# $2 - package url which can be installed
# $3 - specific version of package
define go-install-tool
@[ -f "$(1)-$(3)" ] || { \
set -e; \
package=$(2)@$(3) ;\
echo "Downloading $${package}" ;\
rm -f $(1) || true ;\
GOBIN=$(LOCALBIN) go install $${package} ;\
mv $(1) $(1)-$(3) ;\
} ;\
ln -sf $(1)-$(3) $(1)
endef