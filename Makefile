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

.PHONY: help install-ui build-ui clean-ui build-cli build-controller build install dev-ui test clean fmt lint all release-cli ko-server ko-controller ko-build ko-tag-as-dev

# Default target
help:
	@echo "Available targets:"
	@echo "  install-ui           - Install UI dependencies"
	@echo "  build-ui             - Build the Next.js UI"
	@echo "  clean-ui             - Clean UI build artifacts"
	@echo "  build-cli            - Build the Go CLI"
	@echo "  build-controller     - Build the controller binary"
	@echo "  build                - Build both UI and Go CLI"
	@echo "  install              - Install the CLI to GOPATH/bin"
	@echo "  dev-ui               - Run Next.js in development mode"
	@echo "  test                 - Run Go tests"
	@echo "  clean                - Clean all build artifacts"
	@echo "  all                  - Clean and build everything"
	@echo "  fmt                  - Run the formatter"
	@echo "  lint                 - Run the linter"
	@echo "  release              - Build and release the CLI"
	@echo "  ko-build             - Build all images with ko (server, controller)"
	@echo "  ko-server            - Build server image with ko"
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

# Build the Next.js UI (outputs to internal/registry/api/ui/dist)
build-ui: install-ui
	@echo "Building Next.js UI for embedding..."
	cd ui && npm run build:export
	@echo "Copying built files to internal/registry/api/ui/dist..."
	cp -r ui/out/* internal/registry/api/ui/dist/
# best effort - bring back the gitignore so that dist folder is kept in git (won't work in docker).
	git checkout -- internal/registry/api/ui/dist/.gitignore || :
	@echo "UI built successfully to internal/registry/api/ui/dist/"

# Clean UI build artifacts
clean-ui:
	@echo "Cleaning UI build artifacts..."
	git clean -xdf ./internal/registry/api/ui/dist/
	git clean -xdf ./ui/out/
	git clean -xdf ./ui/.next/
	@echo "UI artifacts cleaned"

# Build the Go CLI
build-cli:
	@echo "Building Go CLI..."
	@echo "Downloading Go dependencies..."
	go mod download
	@echo "Building binary..."
	go build -ldflags "$(LDFLAGS)" \
		-o bin/arctl cmd/cli/main.go
	@echo "Binary built successfully: bin/arctl"

# Build the Go server (with embedded UI)
build-server:
	@echo "Building Go server..."
	@echo "Downloading Go dependencies..."
	go mod download
	@echo "Building binary..."
	go build -ldflags "$(LDFLAGS)" \
		-o bin/arctl-server cmd/server/main.go
	@echo "Binary built successfully: bin/arctl-server"

# Build the controller binary
build-controller:
	@echo "Building controller..."
	@echo "Downloading Go dependencies..."
	go mod download
	@echo "Building binary..."
	go build -ldflags "$(LDFLAGS)" \
		-o bin/controller cmd/controller/main.go
	@echo "Binary built successfully: bin/controller"

# Build everything (UI + Go)
build: build-ui build-cli
	@echo "Build complete!"
	@echo "Run './bin/arctl --help' to get started"

# Install the CLI to GOPATH/bin
install: build
	@echo "Installing arctl to GOPATH/bin..."
	go install
	@echo "Installation complete! Run 'arctl --help' to get started"

# Run Next.js in development mode
dev-ui:
	@echo "Starting Next.js development server..."
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

# Quick development build (skips cleaning)
dev-build: build-ui
	@echo "Building Go CLI (development mode)..."
	go build -o bin/arctl cmd/cli/main.go
	@echo "Development build complete!"

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
.PHONY: ko-build ko-server ko-controller

KO_REGISTRY_PREFIX ?= $(KO_REGISTRY)/$(KO_DOCKER_REPO)

ko-server:
	@echo "Building server image with ko..."
	KO_DOCKER_REPO=$(KO_REGISTRY_PREFIX)/server ko build --local --tags=$(VERSION),latest cmd/server/main.go
	@echo "✓ Server image built: $(KO_REGISTRY_PREFIX)/server:$(VERSION)"

ko-controller:
	@echo "Building controller image with ko..."
	KO_DOCKER_REPO=$(KO_REGISTRY_PREFIX)/controller ko build --local --tags=$(VERSION),latest cmd/controller/main.go
	@echo "✓ Controller image built: $(KO_REGISTRY_PREFIX)/controller:$(VERSION)"

ko-build: ko-server ko-controller
	@echo "✓ All images built with ko"

ko-tag-as-dev:
	@echo "Tagging images as dev with ko..."
	KO_DOCKER_REPO=$(KO_REGISTRY_PREFIX)/server ko build --tags=dev,latest cmd/server/main.go
	KO_DOCKER_REPO=$(KO_REGISTRY_PREFIX)/controller ko build --tags=dev,latest cmd/controller/main.go
	@echo "✓ Images tagged as dev"

bin/arctl-linux-amd64:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o bin/arctl-linux-amd64 cmd/cli/main.go

bin/arctl-linux-amd64.sha256: bin/arctl-linux-amd64
	sha256sum bin/arctl-linux-amd64 > bin/arctl-linux-amd64.sha256

bin/arctl-linux-arm64:
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o bin/arctl-linux-arm64 cmd/cli/main.go

bin/arctl-linux-arm64.sha256: bin/arctl-linux-arm64
	sha256sum bin/arctl-linux-arm64 > bin/arctl-linux-arm64.sha256

bin/arctl-darwin-amd64:
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o bin/arctl-darwin-amd64 cmd/cli/main.go

bin/arctl-darwin-amd64.sha256: bin/arctl-darwin-amd64
	sha256sum bin/arctl-darwin-amd64 > bin/arctl-darwin-amd64.sha256

bin/arctl-darwin-arm64:
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o bin/arctl-darwin-arm64 cmd/cli/main.go

bin/arctl-darwin-arm64.sha256: bin/arctl-darwin-arm64
	sha256sum bin/arctl-darwin-arm64 > bin/arctl-darwin-arm64.sha256

bin/arctl-windows-amd64.exe:
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o bin/arctl-windows-amd64.exe cmd/cli/main.go

bin/arctl-windows-amd64.exe.sha256: bin/arctl-windows-amd64.exe
	sha256sum bin/arctl-windows-amd64.exe > bin/arctl-windows-amd64.exe.sha256

release-cli: bin/arctl-linux-amd64.sha256  
release-cli: bin/arctl-linux-arm64.sha256  
release-cli: bin/arctl-darwin-amd64.sha256  
release-cli: bin/arctl-darwin-arm64.sha256  
release-cli: bin/arctl-windows-amd64.exe.sha256

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