.PHONY: help install-ui build-ui clean-ui build-go build install dev-ui test clean api scraper swag fmt lint all build-agentgateway rebuild-agentgateway

# Default target
help:
	@echo "Available targets:"
	@echo "  install-ui           - Install UI dependencies"
	@echo "  build-ui             - Build the Next.js UI"
	@echo "  clean-ui             - Clean UI build artifacts"
	@echo "  build-go             - Build the Go CLI"
	@echo "  build                - Build both UI and Go CLI"
	@echo "  install              - Install the CLI to GOPATH/bin"
	@echo "  dev-ui               - Run Next.js in development mode"
	@echo "  test                 - Run Go tests"
	@echo "  clean                - Clean all build artifacts"
	@echo "  all                  - Clean and build everything"
	@echo "  build-agentgateway   - Build custom agent gateway Docker image"
	@echo "  rebuild-agentgateway - Force rebuild agent gateway Docker image"
	@echo "  api                  - Run the API"
	@echo "  scraper              - Run the scraper"
	@echo "  swag                 - Run the Swag"
	@echo "  fmt                  - Run the formatter"
	@echo "  lint                 - Run the linter"

# Install UI dependencies
install-ui:
	@echo "Installing UI dependencies..."
	cd ui && npm install

# Build the Next.js UI (outputs to internal/api/ui/dist)
build-ui: install-ui
	@echo "Building Next.js UI for embedding..."
	cd ui && npm run build:export
	@echo "UI built successfully to internal/api/ui/dist/"

# Clean UI build artifacts
clean-ui:
	@echo "Cleaning UI build artifacts..."
	rm -rf ui/.next
	rm -rf internal/api/ui/dist/*
	@echo "UI artifacts cleaned"

# Build the Go CLI (with embedded UI)
build-go:
	@echo "Building Go CLI..."
	@echo "Downloading Go dependencies..."
	go mod download
	@echo "Building binary..."
	go build -ldflags="-X 'github.com/agentregistry-dev/agentregistry/cmd.Version=$$(git describe --tags --always --dirty)' \
		-X 'github.com/agentregistry-dev/agentregistry/cmd.GitCommit=$$(git rev-parse HEAD)' \
		-X 'github.com/agentregistry-dev/agentregistry/cmd.BuildDate=$$(date -u +%Y-%m-%dT%H:%M:%SZ)'" \
		-o bin/arctl main.go
	@echo "Binary built successfully: bin/arctl"

# Build everything (UI + Go)
build: build-ui build-go
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
test:
	@echo "Running Go tests..."
	go test -v ./... && go test -v -tags=integration ./...

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
	go build -o bin/arctl main.go
	@echo "Development build complete!"

api:
	go run ./cmd/registry-api

scraper:
	go run ./cmd/scraper-cli --sources=$(SOURCES)

swag:
	swag init -g ./cmd/registry-api/main.go -o ./api

fmt:
	gofmt -s -w .

lint:
	golangci-lint run --timeout=5m

# Build custom agent gateway image with npx/uvx support
build-agentgateway:
	@echo "Building custom agent gateway image..."
	@if docker image inspect arctl-agentgateway:latest >/dev/null 2>&1; then \
		echo "Image arctl-agentgateway:latest already exists. Use 'make rebuild-agentgateway' to force rebuild."; \
	else \
		docker build -f internal/runtime/agentgateway.Dockerfile -t arctl-agentgateway:latest .; \
		echo "✓ Agent gateway image built successfully"; \
	fi

# Force rebuild custom agent gateway image
rebuild-agentgateway:
	@echo "Rebuilding custom agent gateway image..."
	docker build --no-cache -f internal/runtime/agentgateway.Dockerfile -t arctl-agentgateway:latest .
	@echo "✓ Agent gateway image rebuilt successfully"

