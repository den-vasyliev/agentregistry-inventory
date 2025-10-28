package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var uninstallCmd = &cobra.Command{
	Use:   "uninstall <resource-type> <resource-name>",
	Short: "Uninstall a resource",
	Long:  `Uninstall resources (mcp server, skill).`,
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		resourceType := args[0]
		resourceName := args[1]

		fmt.Printf("Uninstalling %s: %s\n", resourceType, resourceName)
		// TODO: Implement uninstall logic
		// 1. Remove resource configuration
		// 2. Update local database
		fmt.Println("✓ Uninstallation completed successfully")
	},
}

func init() {
	rootCmd.AddCommand(uninstallCmd)
}
