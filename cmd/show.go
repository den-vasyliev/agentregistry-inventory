package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var showCmd = &cobra.Command{
	Use:   "show <resource-type> <resource-name>",
	Short: "Show details of a resource",
	Long:  `Shows detailed information about a resource (mcp, skill, registry).`,
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		resourceType := args[0]
		resourceName := args[1]

		fmt.Printf("Details for %s: %s\n", resourceType, resourceName)
		// TODO: Implement show logic
		// 1. Query database for resource details
		// 2. Display formatted information
		fmt.Println("Name: ", resourceName)
		fmt.Println("Type: ", resourceType)
		fmt.Println("Version: 1.0.0")
		fmt.Println("Description: Placeholder description")
	},
}

func init() {
	rootCmd.AddCommand(showCmd)
}
