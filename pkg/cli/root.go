package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/agentregistry-dev/agentregistry/internal/cli"
	"github.com/agentregistry-dev/agentregistry/internal/cli/agent"
	agentutils "github.com/agentregistry-dev/agentregistry/internal/cli/agent/utils"
	"github.com/agentregistry-dev/agentregistry/internal/cli/configure"
	"github.com/agentregistry-dev/agentregistry/internal/cli/mcp"
	"github.com/agentregistry-dev/agentregistry/internal/cli/skill"
	"github.com/agentregistry-dev/agentregistry/internal/client"
	"github.com/agentregistry-dev/agentregistry/pkg/types"
	"github.com/spf13/cobra"
)

// CLIOptions configures the CLI behavior
type CLIOptions struct {
	// AuthnProvider provides CLI-specific authentication.
	// If nil, uses ARCTL_API_TOKEN env var.
	AuthnProvider types.CLIAuthnProvider
}

var cliOptions CLIOptions
var registryURL string
var registryToken string

// Configure applies options to the root command
func Configure(opts CLIOptions) {
	cliOptions = opts
}

var rootCmd = &cobra.Command{
	Use:   "arctl",
	Short: "Agent Registry CLI",
	Long:  `arctl is a CLI tool for managing agents, MCP servers and skills.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		baseURL, token := resolveRegistryTarget()

		// Get authentication token if no token override was provided
		if token == "" && cliOptions.AuthnProvider != nil {
			var err error
			token, err = cliOptions.AuthnProvider.Authenticate(cmd.Context())
			if err != nil {
				return fmt.Errorf("CLI authentication failed: %w", err)
			}
		}

		// Create API client
		c, err := client.NewClientWithConfig(baseURL, token)
		if err != nil {
			return fmt.Errorf("API client not initialized: %w", err)
		}

		APIClient = c
		mcp.SetAPIClient(APIClient)
		agent.SetAPIClient(APIClient)
		agentutils.SetDefaultRegistryURL(APIClient.BaseURL)
		skill.SetAPIClient(APIClient)
		cli.SetAPIClient(APIClient)
		return nil
	},
}

// APIClient is the shared API client used by CLI commands
var APIClient *client.Client
var verbose bool

func Execute() {
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "V", false, "Verbose output")
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	envBaseURL := os.Getenv("ARCTL_API_BASE_URL")
	envToken := os.Getenv("ARCTL_API_TOKEN")
	rootCmd.PersistentFlags().StringVar(&registryURL, "registry-url", envBaseURL, "Registry base URL (overrides ARCTL_API_BASE_URL; default http://localhost:12121)")
	rootCmd.PersistentFlags().StringVar(&registryToken, "registry-token", envToken, "Registry bearer token (overrides ARCTL_API_TOKEN)")

	// Add subcommands
	rootCmd.AddCommand(mcp.McpCmd)
	rootCmd.AddCommand(agent.AgentCmd)
	rootCmd.AddCommand(skill.SkillCmd)
	rootCmd.AddCommand(configure.ConfigureCmd)
	rootCmd.AddCommand(cli.VersionCmd)
	rootCmd.AddCommand(cli.ImportCmd)
	rootCmd.AddCommand(cli.ExportCmd)
	rootCmd.AddCommand(cli.EmbeddingsCmd)
}

func Root() *cobra.Command {
	return rootCmd
}

func resolveRegistryTarget() (string, string) {
	base := strings.TrimSpace(registryURL)
	if base == "" {
		base = strings.TrimSpace(os.Getenv("ARCTL_API_BASE_URL"))
	}
	base = normalizeBaseURL(base)

	token := registryToken
	if token == "" {
		token = os.Getenv("ARCTL_API_TOKEN")
	}

	return base, token
}

func normalizeBaseURL(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return client.DefaultBaseURL
	}
	if strings.HasPrefix(trimmed, "http://") || strings.HasPrefix(trimmed, "https://") {
		return trimmed
	}
	return "http://" + trimmed
}
