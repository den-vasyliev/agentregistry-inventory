package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var installCmd = &cobra.Command{
	Use:   "install <resource-type> <resource-name> [version] [config]",
	Short: "Install a resource",
	Long:  `Install resources (mcp server, skill) from connected registries.`,
	Args:  cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		resourceType := args[0]
		resourceName := args[1]
		version := "latest"
		if len(args) > 2 {
			version = args[2]
		}

		fmt.Printf("Installing %s: %s@%s\n", resourceType, resourceName, version)

		// TODO: Implement install logic
		// 1. Fetch resource from registry
		// 2. For MCP servers, prompt for environment variables
		// 3. Install and configure resource
		// 4. Update local database

		if resourceType == "mcp" {
			fmt.Println("Environment variables required:")
			fmt.Println("  - API_KEY (required)")
			fmt.Println("  - API_URL (optional)")
			// TODO: Prompt user for values
		}

		fmt.Println("✓ Installation completed successfully")
	},
}

func init() {
	rootCmd.AddCommand(installCmd)
}
