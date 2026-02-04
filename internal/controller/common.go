package controller

import (
	"fmt"
	"strings"
)

// Discovery label constants shared across discovery handlers
const (
	discoveryLabel  = "agentregistry.dev/discovered"
	sourceKindLabel = "agentregistry.dev/source-kind"
	sourceNameLabel = "agentregistry.dev/source-name"
	sourceNSLabel   = "agentregistry.dev/source-namespace"
)

// getEnvironmentFromNamespace extracts environment from namespace
// Returns the namespace as environment if not recognized
func getEnvironmentFromNamespace(namespace string) string {
	// Map common namespace patterns to environment names
	switch namespace {
	case "dev", "development":
		return "dev"
	case "staging", "stage":
		return "staging"
	case "prod", "production":
		return "prod"
	default:
		return namespace
	}
}

// generateCatalogName creates a catalog name from namespace and resource name
func generateCatalogName(namespace, name string) string {
	combined := fmt.Sprintf("%s-%s", namespace, name)
	// Sanitize for K8s naming
	combined = strings.ReplaceAll(combined, "/", "-")
	combined = strings.ReplaceAll(combined, "_", "-")
	combined = strings.ToLower(combined)
	if len(combined) > 63 {
		combined = combined[:63]
	}
	// K8s names must end with alphanumeric, not hyphen
	combined = strings.TrimRight(combined, "-")
	return combined
}

// generateAgentCatalogName creates an agent catalog name from namespace and agent CR name
func generateAgentCatalogName(namespace, agentCRName string) string {
	combined := fmt.Sprintf("%s-%s", namespace, agentCRName)
	combined = strings.ReplaceAll(combined, "/", "-")
	combined = strings.ReplaceAll(combined, "_", "-")
	combined = strings.ToLower(combined)
	if len(combined) > 63 {
		combined = combined[:63]
	}
	return combined
}

// generateModelCatalogName creates a model catalog name from namespace and model name
func generateModelCatalogName(namespace, name string) string {
	combined := fmt.Sprintf("%s-%s", namespace, name)
	combined = strings.ReplaceAll(combined, "/", "-")
	combined = strings.ReplaceAll(combined, "_", "-")
	combined = strings.ToLower(combined)
	if len(combined) > 63 {
		combined = combined[:63]
	}
	return combined
}

// parseSkillRef parses a skill reference into name and version
func parseSkillRef(ref string) (name, version string) {
	// Handle digest references
	if idx := strings.LastIndex(ref, "@"); idx != -1 {
		return ref[:idx], ref[idx+1:]
	}

	// Handle tag references
	if idx := strings.LastIndex(ref, ":"); idx != -1 {
		// Make sure we're not splitting on a port number
		possibleTag := ref[idx+1:]
		if !strings.Contains(possibleTag, "/") {
			return ref[:idx], possibleTag
		}
	}

	return ref, "latest"
}

// generateSkillCatalogName creates a valid K8s name from skill name and version
func generateSkillCatalogName(name, version string) string {
	// Remove registry prefix for shorter names
	shortName := name
	if strings.Contains(name, "/") {
		// Keep only the last two parts (org/skill)
		parts := strings.Split(name, "/")
		if len(parts) >= 2 {
			shortName = strings.Join(parts[len(parts)-2:], "-")
		}
	}

	combined := fmt.Sprintf("%s-%s", shortName, version)
	combined = strings.ReplaceAll(combined, "/", "-")
	combined = strings.ReplaceAll(combined, "_", "-")
	combined = strings.ReplaceAll(combined, ".", "-")
	combined = strings.ToLower(combined)

	if len(combined) > 63 {
		combined = combined[:63]
	}

	// Trim trailing dashes
	combined = strings.TrimRight(combined, "-")

	return combined
}
