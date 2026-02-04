package controller

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
