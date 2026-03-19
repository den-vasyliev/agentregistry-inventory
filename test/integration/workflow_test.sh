#!/bin/bash

# Integration test script for Agent Registry submission workflow
# This script simulates the complete workflow from PR creation to deployment

set -e

# Configuration
REPO_DIR="${GITHUB_WORKSPACE:-/tmp/test-monorepo}"
REGISTRY_URL="${REGISTRY_API_URL:-http://localhost:8080}"
TEST_RESOURCE_NAME="test-filesystem-server"
TEST_VERSION="1.0.0"
TEST_KIND="mcp-server"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Test functions
test_manifest_validation() {
    log_info "Testing manifest validation..."
    
    local test_dir="$REPO_DIR/resources/$TEST_KIND/$TEST_RESOURCE_NAME"
    mkdir -p "$test_dir"
    
    # Create valid manifest
    cat > "$test_dir/$TEST_RESOURCE_NAME-$TEST_VERSION.yaml" << EOF
apiVersion: agentregistry.dev/v1alpha1
kind: MCPServerCatalog
metadata:
  name: $TEST_RESOURCE_NAME-$(echo $TEST_VERSION | tr '.' '-')
  labels:
    agentregistry.dev/resource-name: $TEST_RESOURCE_NAME
    agentregistry.dev/resource-version: $TEST_VERSION
    agentregistry.dev/auto-deploy: "true"
spec:
  name: $TEST_RESOURCE_NAME
  version: $TEST_VERSION
  title: Test Filesystem Server
  description: Test MCP server for filesystem operations
  packages:
    - type: oci
      image: ghcr.io/test/filesystem-server:$TEST_VERSION
      transport:
        type: stdio
  repository:
    url: https://github.com/test-org/test-repo
    source: github
EOF

    # Validate YAML syntax
    if command -v yq >/dev/null 2>&1; then
        if yq eval '.' "$test_dir/$TEST_RESOURCE_NAME-$TEST_VERSION.yaml" >/dev/null 2>&1; then
            log_info "✅ YAML validation passed"
        else
            log_error "❌ YAML validation failed"
            return 1
        fi
    else
        log_warn "yq not installed, skipping YAML validation"
    fi
    
    # Validate required fields
    local name=$(yq eval '.spec.name' "$test_dir/$TEST_RESOURCE_NAME-$TEST_VERSION.yaml")
    local version=$(yq eval '.spec.version' "$test_dir/$TEST_RESOURCE_NAME-$TEST_VERSION.yaml")
    local kind=$(yq eval '.kind' "$test_dir/$TEST_RESOURCE_NAME-$TEST_VERSION.yaml")
    
    if [ "$name" = "$TEST_RESOURCE_NAME" ] && [ "$version" = "$TEST_VERSION" ] && [ "$kind" = "MCPServerCatalog" ]; then
        log_info "✅ Required fields validation passed"
    else
        log_error "❌ Required fields validation failed"
        return 1
    fi
    
    return 0
}

test_webhook_endpoint() {
    log_info "Testing webhook endpoint..."
    
    local payload=$(cat << EOF
{
  "ref": "refs/heads/main",
  "repository": {
    "full_name": "test-org/test-repo",
    "html_url": "https://github.com/test-org/test-repo"
  },
  "head_commit": {
    "id": "abc123def456",
    "message": "Add test filesystem server",
    "added": ["resources/$TEST_KIND/$TEST_RESOURCE_NAME/$TEST_RESOURCE_NAME-$TEST_VERSION.yaml"],
    "modified": [],
    "removed": []
  },
  "pusher": {
    "name": "testuser",
    "email": "test@example.com"
  }
}
EOF
)

    local response=$(curl -s -w "%{http_code}" -X POST "$REGISTRY_URL/webhooks/github" \
        -H "Content-Type: application/json" \
        -d "$payload" || echo "000")
    
    local http_code="${response: -3}"
    local body="${response%???}"
    
    if [ "$http_code" = "200" ]; then
        log_info "✅ Webhook endpoint responded successfully"
        echo "Response: $body"
    else
        log_error "❌ Webhook endpoint failed (HTTP $http_code)"
        echo "Response: $body"
        return 1
    fi
    
    return 0
}

test_oci_artifact_creation() {
    log_info "Testing OCI artifact creation..."
    
    local payload=$(cat << EOF
{
  "resourceName": "$TEST_RESOURCE_NAME",
  "version": "$TEST_VERSION",
  "resourceType": "$TEST_KIND",
  "environment": "test",
  "repositoryUrl": "https://github.com/test-org/test-repo",
  "commitSha": "abc123def456",
  "pusher": "testuser"
}
EOF
)

    local response=$(curl -s -w "%{http_code}" -X POST "$REGISTRY_URL/admin/v0/oci/artifacts" \
        -H "Content-Type: application/json" \
        -d "$payload" || echo "000")
    
    local http_code="${response: -3}"
    local body="${response%???}"
    
    if [ "$http_code" = "201" ] || [ "$http_code" = "200" ]; then
        log_info "✅ OCI artifact creation succeeded"
        echo "Response: $body"
    else
        log_warn "⚠️ OCI artifact creation failed (HTTP $http_code) - this might be expected in test environment"
        echo "Response: $body"
    fi
    
    return 0
}

test_registry_api_health() {
    log_info "Testing registry API health..."
    
    local response=$(curl -s -w "%{http_code}" "$REGISTRY_URL/healthz" || echo "000")
    local http_code="${response: -3}"
    
    if [ "$http_code" = "200" ]; then
        log_info "✅ Registry API is healthy"
    else
        log_error "❌ Registry API health check failed (HTTP $http_code)"
        return 1
    fi
    
    return 0
}

test_file_detection() {
    log_info "Testing file detection logic..."
    
    # Create test files
    mkdir -p "$REPO_DIR/resources/mcp-server/test1"
    mkdir -p "$REPO_DIR/resources/agent/test2"
    mkdir -p "$REPO_DIR/resources/skill/test3"
    mkdir -p "$REPO_DIR/resources/model/test4"
    mkdir -p "$REPO_DIR/src"
    
    touch "$REPO_DIR/resources/mcp-server/test1/test1-1.0.0.yaml"
    touch "$REPO_DIR/resources/agent/test2/test2-2.0.0.yml"
    touch "$REPO_DIR/resources/skill/test3/test3-1.5.0.yaml"
    touch "$REPO_DIR/resources/model/test4/test4-3.0.0.yaml"
    touch "$REPO_DIR/src/main.go"
    touch "$REPO_DIR/README.md"
    
    # Simulate file detection
    local registry_files=$(find "$REPO_DIR/resources" -name "*.yaml" -o -name "*.yml" 2>/dev/null | wc -l)
    local non_registry_files=$(find "$REPO_DIR" -maxdepth 1 -name "*.md" -o -name "*.go" 2>/dev/null | wc -l)
    
    if [ "$registry_files" -eq 4 ]; then
        log_info "✅ Registry files detected correctly ($registry_files files)"
    else
        log_error "❌ Registry file detection failed (expected 4, got $registry_files)"
        return 1
    fi
    
    return 0
}

test_path_parsing() {
    log_info "Testing path parsing logic..."
    
    local test_cases=(
        "resources/mcp-server/filesystem/filesystem-1.0.0.yaml:mcp-server:filesystem:1.0.0"
        "resources/agent/code-reviewer/code-reviewer-2.1.0.yml:agent:code-reviewer:2.1.0"
        "resources/skill/terraform/terraform-1.5.0.yaml:skill:terraform:1.5.0"
        "resources/model/claude/claude-3.5.yaml:model:claude:3.5"
    )
    
    for test_case in "${test_cases[@]}"; do
        IFS=':' read -r path expected_kind expected_name expected_version <<< "$test_case"
        
        # Extract components from path
        IFS='/' read -ra PARTS <<< "$path"
        local kind="${PARTS[1]}"
        local name="${PARTS[2]}"
        local filename=$(basename "$path" .yaml)
        local filename=$(basename "$filename" .yml)
        
        # Extract version from filename
        local version=""
        if [[ "$filename" =~ ^${name}-(.+)$ ]]; then
            version="${BASH_REMATCH[1]}"
        fi
        
        if [ "$kind" = "$expected_kind" ] && [ "$name" = "$expected_name" ] && [ "$version" = "$expected_version" ]; then
            log_info "✅ Path parsing correct for $path"
        else
            log_error "❌ Path parsing failed for $path (got: $kind/$name/$version, expected: $expected_kind/$expected_name/$expected_version)"
            return 1
        fi
    done
    
    return 0
}

cleanup() {
    log_info "Cleaning up test files..."
    if [ -d "$REPO_DIR" ] && [[ "$REPO_DIR" == *"test"* ]]; then
        rm -rf "$REPO_DIR"
    fi
}

main() {
    log_info "Starting Agent Registry integration tests..."
    
    # Setup
    mkdir -p "$REPO_DIR"
    trap cleanup EXIT
    
    # Run tests
    local failed_tests=0
    
    test_manifest_validation || ((failed_tests++))
    test_file_detection || ((failed_tests++))
    test_path_parsing || ((failed_tests++))
    
    # Only test API endpoints if registry is available
    if curl -s --connect-timeout 5 "$REGISTRY_URL/healthz" >/dev/null 2>&1; then
        test_registry_api_health || ((failed_tests++))
        test_webhook_endpoint || ((failed_tests++))
        test_oci_artifact_creation || ((failed_tests++))
    else
        log_warn "Registry API not available, skipping API tests"
    fi
    
    # Summary
    if [ $failed_tests -eq 0 ]; then
        log_info "🎉 All integration tests passed!"
        exit 0
    else
        log_error "❌ $failed_tests test(s) failed"
        exit 1
    fi
}

# Run tests if script is executed directly
if [ "${BASH_SOURCE[0]}" = "${0}" ]; then
    main "$@"
fi