package agent

import (
	"fmt"
	"os"
	"strings"

	"github.com/agentregistry-dev/agentregistry/pkg/models"
	"github.com/spf13/cobra"
)

var DeployCmd = &cobra.Command{
	Use:   "deploy [agent-name]",
	Short: "Deploy an agent to Kubernetes",
	Long: `Deploy an agent from the registry to Kubernetes.

Example:
  arctl agent deploy my-agent --version latest
  arctl agent deploy my-agent --version 1.2.3 --namespace my-namespace`,
	Args: cobra.ExactArgs(1),
	RunE: runDeploy,
}

func runDeploy(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return cmd.Help()
	}

	name := args[0]
	version, _ := cmd.Flags().GetString("version")
	namespace, _ := cmd.Flags().GetString("namespace")

	if version == "" {
		version = "latest"
	}

	if namespace == "" {
		namespace = "kagent"
	}

	if apiClient == nil {
		return fmt.Errorf("API client not initialized")
	}

	agentModel, err := apiClient.GetAgentByNameAndVersion(name, version)
	if err != nil {
		return fmt.Errorf("failed to fetch agent %q: %w", name, err)
	}
	if agentModel == nil {
		return fmt.Errorf("agent not found: %s (version %s)", name, version)
	}

	manifest := &agentModel.Agent.AgentManifest

	// Validate that required API keys are set
	if err := validateAPIKey(manifest.ModelProvider); err != nil {
		return err
	}

	// Build config map with environment variables
	config := buildDeployConfig(manifest)
	config["KAGENT_NAMESPACE"] = namespace

	// Deploy to Kubernetes
	fmt.Printf("Deploying agent to Kubernetes...\n")
	deployment, err := apiClient.DeployAgent(name, version, config, "kubernetes")
	if err != nil {
		return fmt.Errorf("failed to deploy agent: %w", err)
	}

	fmt.Printf("\n✓ Deployed agent '%s' version '%s' to Kubernetes\n", deployment.ServerName, deployment.Version)
	fmt.Printf("Namespace: %s\n", namespace)
	return nil
}

// buildDeployConfig creates the configuration map with all necessary environment variables
func buildDeployConfig(manifest *models.AgentManifest) map[string]string {
	config := make(map[string]string)

	// Add model provider API key if available
	providerAPIKeys := map[string]string{
		"openai":      "OPENAI_API_KEY",
		"anthropic":   "ANTHROPIC_API_KEY",
		"azureopenai": "AZUREOPENAI_API_KEY",
		"gemini":      "GOOGLE_API_KEY",
	}

	if envVar, ok := providerAPIKeys[strings.ToLower(manifest.ModelProvider)]; ok && envVar != "" {
		if value := os.Getenv(envVar); value != "" {
			config[envVar] = value
		}
	}

	if manifest.TelemetryEndpoint != "" {
		config["OTEL_EXPORTER_OTLP_TRACES_ENDPOINT"] = manifest.TelemetryEndpoint
	}

	return config
}

// validateAPIKey checks if the required API key for a model provider is set
func validateAPIKey(modelProvider string) error {
	providerAPIKeys := map[string]string{
		"openai":      "OPENAI_API_KEY",
		"anthropic":   "ANTHROPIC_API_KEY",
		"azureopenai": "AZUREOPENAI_API_KEY",
		"gemini":      "GOOGLE_API_KEY",
	}

	envVar, ok := providerAPIKeys[strings.ToLower(modelProvider)]
	if !ok || envVar == "" {
		return nil
	}
	if os.Getenv(envVar) == "" {
		return fmt.Errorf("required API key %s not set for model provider %s", envVar, modelProvider)
	}
	return nil
}

func init() {
	DeployCmd.Flags().String("version", "latest", "Agent version to deploy")
	DeployCmd.Flags().Bool("prefer-remote", false, "Prefer using a remote source when available")
	DeployCmd.Flags().String("namespace", "kagent", "Kubernetes namespace for agent deployment")
}
