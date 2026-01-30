package mcp

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	deleteForceFlag bool
	deleteVersion   string
)

var DeleteCmd = &cobra.Command{
	Use:   "delete <server-name>",
	Short: "Delete an MCP server from the registry",
	Long: `Delete an MCP server from the registry.
The server must not be published or deployed unless --force is used.

Examples:
  arctl mcp delete my-server --version 1.0.0
  arctl mcp delete my-server --version 1.0.0 --force`,
	Args: cobra.ExactArgs(1),
	RunE: runDelete,
}

func init() {
	DeleteCmd.Flags().StringVar(&deleteVersion, "version", "", "Specify the version to delete (required)")
	DeleteCmd.Flags().BoolVar(&deleteForceFlag, "force", false, "Force delete even if published or deployed")
	_ = DeleteCmd.MarkFlagRequired("version")
}

func runDelete(cmd *cobra.Command, args []string) error {
	serverName := args[0]

	if apiClient == nil {
		return fmt.Errorf("API client not initialized")
	}

	// Check if server is published
	isPublished, err := isServerPublished(serverName, deleteVersion)
	if err != nil {
		return fmt.Errorf("failed to check if server is published: %w", err)
	}

	// Fail if published unless --force is used
	if !deleteForceFlag && isPublished {
		return fmt.Errorf("server %s version %s is published. Unpublish it first using 'arctl mcp unpublish %s --version %s', or use --force to delete anyway", serverName, deleteVersion, serverName, deleteVersion)
	}

	// Delete the server
	// Note: Deployment management is now handled by Kubernetes directly (kubectl delete application)
	fmt.Printf("Deleting server %s version %s...\n", serverName, deleteVersion)
	err = apiClient.DeleteMCPServer(serverName, deleteVersion)
	if err != nil {
		return fmt.Errorf("failed to delete server: %w", err)
	}

	fmt.Printf("MCP server '%s' version %s deleted successfully\n", serverName, deleteVersion)
	return nil
}

func isServerPublished(serverName, version string) (bool, error) {
	if apiClient == nil {
		return false, fmt.Errorf("API client not initialized")
	}

	// Get server to check if it exists
	// Note: Published field checking removed - no longer tracked in K8s-native architecture
	server, err := apiClient.GetServerByNameAndVersion(serverName, version, false)
	if err != nil {
		return false, err
	}

	// If server exists, consider it published (in K8s-native arch, existence implies availability)
	return server != nil, nil
}
