package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var searchCmd = &cobra.Command{
	Use:   "search <resource-type> <search-term>",
	Short: "Search for resources",
	Long:  `Search for resources from the connected registries.`,
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		resourceType := args[0]
		searchTerm := args[1]

		fmt.Printf("Searching for %s: %s\n", resourceType, searchTerm)
		// TODO: Implement search logic
		// 1. Query database with search term
		// 2. Return matching results
		fmt.Println("No results found")
	},
}

func init() {
	rootCmd.AddCommand(searchCmd)
}
