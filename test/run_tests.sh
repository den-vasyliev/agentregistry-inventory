#!/bin/bash

# Test runner for Agent Registry components
# Runs unit tests, integration tests, and linting

set -e

# Configuration
PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
GO_TEST_TIMEOUT="10m"
COVERAGE_THRESHOLD=80

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

log_header() {
    echo -e "\n${BLUE}=== $1 ===${NC}"
}

# Function to run Go tests
run_go_tests() {
    log_header "Running Go unit tests"
    
    cd "$PROJECT_ROOT"
    
    # Run tests with coverage
    go test -timeout="$GO_TEST_TIMEOUT" -race -coverprofile=coverage.out -covermode=atomic ./internal/... || {
        log_error "Go tests failed"
        return 1
    }
    
    # Check coverage
    local coverage=$(go tool cover -func=coverage.out | grep total | awk '{print $3}' | sed 's/%//')
    log_info "Test coverage: ${coverage}%"
    
    if (( $(echo "$coverage >= $COVERAGE_THRESHOLD" | bc -l) )); then
        log_info "✅ Coverage threshold met (>= ${COVERAGE_THRESHOLD}%)"
    else
        log_warn "⚠️ Coverage below threshold (${coverage}% < ${COVERAGE_THRESHOLD}%)"
    fi
    
    # Generate HTML coverage report
    go tool cover -html=coverage.out -o coverage.html
    log_info "Coverage report generated: coverage.html"
}

# Function to run UI tests
run_ui_tests() {
    log_header "Running UI tests"
    
    cd "$PROJECT_ROOT/ui"
    
    # Check if package.json exists
    if [ ! -f "package.json" ]; then
        log_warn "No package.json found, skipping UI tests"
        return 0
    fi
    
    # Install dependencies if needed
    if [ ! -d "node_modules" ]; then
        log_info "Installing UI dependencies..."
        npm install
    fi
    
    # Run tests
    npm test -- --watchAll=false --coverage || {
        log_error "UI tests failed"
        return 1
    }
    
    log_info "✅ UI tests passed"
}

# Function to run linting
run_linting() {
    log_header "Running linting checks"
    
    cd "$PROJECT_ROOT"
    
    # Go linting
    if command -v golangci-lint >/dev/null 2>&1; then
        log_info "Running golangci-lint..."
        golangci-lint run ./... || {
            log_error "Go linting failed"
            return 1
        }
        log_info "✅ Go linting passed"
    else
        log_warn "golangci-lint not installed, skipping Go linting"
    fi
    
    # Go formatting check
    local unformatted=$(gofmt -l . 2>/dev/null | grep -v vendor || true)
    if [ -n "$unformatted" ]; then
        log_error "Go files not formatted:"
        echo "$unformatted"
        return 1
    fi
    log_info "✅ Go formatting check passed"
    
    # UI linting
    if [ -f "ui/package.json" ] && command -v npm >/dev/null 2>&1; then
        cd "$PROJECT_ROOT/ui"
        if npm run lint >/dev/null 2>&1; then
            log_info "✅ UI linting passed"
        else
            log_warn "UI linting failed or not configured"
        fi
    fi
}

# Function to run integration tests
run_integration_tests() {
    log_header "Running integration tests"
    
    local integration_test="$PROJECT_ROOT/test/integration/workflow_test.sh"
    
    if [ -f "$integration_test" ]; then
        chmod +x "$integration_test"
        "$integration_test" || {
            log_error "Integration tests failed"
            return 1
        }
        log_info "✅ Integration tests passed"
    else
        log_warn "Integration test script not found, skipping"
    fi
}

# Function to validate GitHub Actions workflow
validate_github_actions() {
    log_header "Validating GitHub Actions workflows"
    
    local workflow_dir="$PROJECT_ROOT/.github/workflows"
    
    if [ ! -d "$workflow_dir" ]; then
        log_warn "No GitHub Actions workflows found"
        return 0
    fi
    
    # Check for actionlint
    if command -v actionlint >/dev/null 2>&1; then
        actionlint "$workflow_dir"/*.yml || {
            log_error "GitHub Actions workflow validation failed"
            return 1
        }
        log_info "✅ GitHub Actions workflows valid"
    else
        log_warn "actionlint not installed, skipping workflow validation"
    fi
}

# Function to run security checks
run_security_checks() {
    log_header "Running security checks"
    
    cd "$PROJECT_ROOT"
    
    # Go vulnerability check
    if command -v govulncheck >/dev/null 2>&1; then
        govulncheck ./... || {
            log_error "Vulnerability check failed"
            return 1
        }
        log_info "✅ No known vulnerabilities found"
    else
        log_warn "govulncheck not installed, skipping vulnerability check"
    fi
    
    # Check for hardcoded secrets
    local secrets=$(grep -r -E "(api[_-]?key|secret|token|password)" --include="*.go" --include="*.yaml" --include="*.yml" . | grep -v test | grep -v example || true)
    if [ -n "$secrets" ]; then
        log_warn "Potential secrets found:"
        echo "$secrets"
    else
        log_info "✅ No hardcoded secrets detected"
    fi
}

# Function to run benchmarks
run_benchmarks() {
    log_header "Running benchmarks"
    
    cd "$PROJECT_ROOT"
    
    local bench_files=$(find . -name "*_test.go" -exec grep -l "func Benchmark" {} \; 2>/dev/null || true)
    if [ -n "$bench_files" ]; then
        go test -bench=. -benchmem ./... || {
            log_error "Benchmarks failed"
            return 1
        }
        log_info "✅ Benchmarks completed"
    else
        log_warn "No benchmark tests found"
    fi
}

# Function to check dependencies
check_dependencies() {
    log_header "Checking dependencies"
    
    cd "$PROJECT_ROOT"
    
    # Go mod tidy check
    cp go.mod go.mod.bak
    cp go.sum go.sum.bak
    go mod tidy
    
    if ! diff go.mod go.mod.bak >/dev/null || ! diff go.sum go.sum.bak >/dev/null; then
        log_error "go.mod/go.sum not tidy"
        mv go.mod.bak go.mod
        mv go.sum.bak go.sum
        return 1
    fi
    
    rm go.mod.bak go.sum.bak
    log_info "✅ Dependencies are tidy"
    
    # Check for outdated dependencies
    if command -v go-mod-outdated >/dev/null 2>&1; then
        local outdated=$(go list -u -m -json all | go-mod-outdated -update -direct 2>/dev/null || true)
        if [ -n "$outdated" ]; then
            log_warn "Outdated dependencies found:"
            echo "$outdated"
        else
            log_info "✅ All dependencies up to date"
        fi
    fi
}

# Function to build artifacts
build_artifacts() {
    log_header "Building artifacts"
    
    cd "$PROJECT_ROOT"
    
    # Build controller
    go build -o bin/controller ./cmd/controller || {
        log_error "Controller build failed"
        return 1
    }
    log_info "✅ Controller built successfully"
    
    # Build UI if applicable
    if [ -f "ui/package.json" ] && [ -d "ui/node_modules" ]; then
        cd ui
        npm run build >/dev/null 2>&1 || {
            log_error "UI build failed"
            return 1
        }
        log_info "✅ UI built successfully"
        cd ..
    fi
    
    # Generate API documentation
    if command -v swag >/dev/null 2>&1 && [ -f "docs/swagger.yaml" ]; then
        swag init -g cmd/controller/main.go -o docs/ || {
            log_warn "API documentation generation failed"
        }
    fi
}

# Function to cleanup
cleanup() {
    log_info "Cleaning up..."
    
    # Remove temporary files
    rm -f coverage.out coverage.html
    rm -rf bin/
    
    # Clean UI build artifacts
    if [ -d "ui/build" ]; then
        rm -rf ui/build
    fi
}

# Main function
main() {
    local run_all=true
    local failed_checks=0
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            --unit-tests)
                run_all=false
                run_go_tests || ((failed_checks++))
                ;;
            --ui-tests)
                run_all=false
                run_ui_tests || ((failed_checks++))
                ;;
            --lint)
                run_all=false
                run_linting || ((failed_checks++))
                ;;
            --integration)
                run_all=false
                run_integration_tests || ((failed_checks++))
                ;;
            --security)
                run_all=false
                run_security_checks || ((failed_checks++))
                ;;
            --bench)
                run_all=false
                run_benchmarks || ((failed_checks++))
                ;;
            --build)
                run_all=false
                build_artifacts || ((failed_checks++))
                ;;
            --clean)
                cleanup
                exit 0
                ;;
            --help)
                echo "Usage: $0 [--unit-tests] [--ui-tests] [--lint] [--integration] [--security] [--bench] [--build] [--clean] [--help]"
                echo "Run without arguments to execute all checks"
                exit 0
                ;;
            *)
                log_error "Unknown option: $1"
                exit 1
                ;;
        esac
        shift
    done
    
    # Run all checks if no specific options provided
    if [ "$run_all" = true ]; then
        log_info "Running all checks..."
        
        check_dependencies || ((failed_checks++))
        run_linting || ((failed_checks++))
        run_go_tests || ((failed_checks++))
        run_ui_tests || ((failed_checks++))
        validate_github_actions || ((failed_checks++))
        run_security_checks || ((failed_checks++))
        run_benchmarks || true  # Don't fail on benchmark issues
        run_integration_tests || ((failed_checks++))
        build_artifacts || ((failed_checks++))
    fi
    
    # Summary
    log_header "Test Summary"
    
    if [ $failed_checks -eq 0 ]; then
        log_info "🎉 All checks passed!"
        exit 0
    else
        log_error "❌ $failed_checks check(s) failed"
        exit 1
    fi
}

# Setup trap for cleanup
trap cleanup EXIT

# Run main function
main "$@"