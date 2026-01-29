package mcp

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/agentregistry-dev/agentregistry/internal/cli/mcp/manifest"
	"github.com/agentregistry-dev/agentregistry/internal/utils"
	"github.com/spf13/cobra"
	"github.com/stoewer/go-strcase"
)

var (
	runVersion    string
	runInspector  bool
	runYes        bool
	runVerbose    bool
	runEnvVars    []string
	runArgVars    []string
	runHeaderVars []string
)

var RunCmd = &cobra.Command{
	Use:   "run <path>",
	Short: "Run a local MCP server",
	Long: `Run a local MCP server project.

Run a local MCP project by path (e.g., 'arctl mcp run .' or 'arctl mcp run ./my-mcp-server').
The server must be built first using 'arctl mcp build'.

For registry servers, use 'arctl mcp deploy' to deploy to Kubernetes.`,
	Args: cobra.ExactArgs(1),
	RunE: runRun,
}

func init() {
	RunCmd.Flags().StringVar(&runVersion, "version", "", "Specify the version of the server to run")
	RunCmd.Flags().BoolVar(&runInspector, "inspector", false, "Launch MCP Inspector to interact with the server")
	RunCmd.Flags().BoolVarP(&runYes, "yes", "y", false, "Automatically accept all prompts (use default values)")
	RunCmd.Flags().BoolVar(&runVerbose, "verbose", false, "Enable verbose logging")
	RunCmd.Flags().StringArrayVarP(&runEnvVars, "env", "e", []string{}, "Environment variables (key=value)")
	RunCmd.Flags().StringArrayVar(&runArgVars, "arg", []string{}, "Runtime arguments (key=value)")
	RunCmd.Flags().StringArrayVar(&runHeaderVars, "header", []string{}, "Headers for remote servers (key=value)")
}

func runRun(cmd *cobra.Command, args []string) error {
	serverNameOrPath := args[0]

	if isLocalPath(serverNameOrPath) {
		return runLocalMCPServer(serverNameOrPath)
	}

	// For registry servers, point users to deploy command
	return fmt.Errorf("to run a registry server, use 'arctl mcp deploy %s' to deploy to Kubernetes", serverNameOrPath)
}

// parseKeyValuePairs parses key=value pairs from command line flags
func parseKeyValuePairs(pairs []string) (map[string]string, error) {
	result := make(map[string]string)
	for _, pair := range pairs {
		idx := findFirstEquals(pair)
		if idx == -1 {
			return nil, fmt.Errorf("invalid key=value pair (missing =): %s", pair)
		}
		key := pair[:idx]
		value := pair[idx+1:]
		result[key] = value
	}
	return result, nil
}

// findFirstEquals finds the first = character in a string
func findFirstEquals(s string) int {
	for i, c := range s {
		if c == '=' {
			return i
		}
	}
	return -1
}

// generateRandomName generates a random hex string for use in naming
func generateRandomName() (string, error) {
	randomBytes := make([]byte, 8)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", fmt.Errorf("failed to generate random name: %w", err)
	}
	return hex.EncodeToString(randomBytes), nil
}

// isLocalPath checks if the given string is a path to a local directory
func isLocalPath(path string) bool {
	// Check for common path indicators
	if path == "." || path == ".." || filepath.IsAbs(path) {
		return true
	}
	// Check for relative paths
	if len(path) > 0 && (path[0] == '.' || path[0] == '/') {
		return true
	}
	// Check if the path exists as a directory
	if info, err := os.Stat(path); err == nil && info.IsDir() {
		return true
	}
	return false
}

// runLocalMCPServer runs a local MCP server from a project directory
func runLocalMCPServer(projectPath string) error {
	absPath, err := filepath.Abs(projectPath)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Load the manifest
	manifestManager := manifest.NewManager(absPath)
	if !manifestManager.Exists() {
		return fmt.Errorf("mcp.yaml not found in %s. Run 'arctl mcp init' first", absPath)
	}

	projectManifest, err := manifestManager.Load()
	if err != nil {
		return fmt.Errorf("failed to load project manifest: %w", err)
	}

	// Determine the Docker image name (same logic as build command)
	version := projectManifest.Version
	if version == "" {
		version = "latest"
	}
	imageName := fmt.Sprintf("%s:%s", strcase.KebabCase(projectManifest.Name), version)

	// Check if the image exists locally
	if err := checkDockerImageExists(imageName); err != nil {
		return fmt.Errorf("docker image %s not found. Run 'arctl mcp build %s' first\n%w", imageName, projectPath, err)
	}

	fmt.Printf("Running local MCP server: %s (version %s)\n", projectManifest.Name, version)
	fmt.Printf("Using Docker image: %s\n", imageName)

	return runLocalMCPServerWithDocker(projectManifest, imageName)
}

// runLocalMCPServerWithDocker runs the Docker container directly for local development
func runLocalMCPServerWithDocker(manifest *manifest.ProjectManifest, imageName string) error {
	port, err := utils.FindAvailablePort()
	if err != nil {
		return fmt.Errorf("failed to find available port: %w", err)
	}

	// Parse environment variables from flags and merge with defaults
	envValues, err := parseKeyValuePairs(runEnvVars)
	if err != nil {
		return fmt.Errorf("failed to parse environment variables: %w", err)
	}

	if envValues["MCP_TRANSPORT_MODE"] == "" {
		envValues["MCP_TRANSPORT_MODE"] = "http"
	}
	if envValues["PORT"] == "" {
		envValues["PORT"] = "3000"
	}
	if envValues["HOST"] == "" {
		// Bind to 0.0.0.0 so the server is accessible from outside the container
		envValues["HOST"] = "0.0.0.0"
	}

	// Build docker run command
	containerName := fmt.Sprintf("arctl-run-%s", manifest.Name)
	args := []string{
		"run",
		"--rm",
		"--name", containerName,
		"-p", fmt.Sprintf("%d:3000", port),
	}

	for k, v := range envValues {
		args = append(args, "-e", fmt.Sprintf("%s=%s", k, v))
	}

	args = append(args, imageName)

	fmt.Printf("\nMCP Server URL: http://localhost:%d/mcp\n", port)
	fmt.Println("\nPress CTRL+C to stop the server...")
	fmt.Println()

	// Create the docker run command
	dockerCmd := exec.Command("docker", args...)
	dockerCmd.Stdout = os.Stdout
	dockerCmd.Stderr = os.Stderr
	dockerCmd.Stdin = os.Stdin

	// Start the container
	if err := dockerCmd.Start(); err != nil {
		return fmt.Errorf("failed to start docker container: %w", err)
	}

	// Launch inspector if requested
	var inspectorCmd *exec.Cmd
	if runInspector {
		serverURL := fmt.Sprintf("http://localhost:%d/mcp", port)
		fmt.Println("Launching MCP Inspector...")
		inspectorCmd = exec.Command("npx", "-y", "@modelcontextprotocol/inspector", "--server-url", serverURL)
		inspectorCmd.Stdout = os.Stdout
		inspectorCmd.Stderr = os.Stderr
		inspectorCmd.Stdin = os.Stdin

		if err := inspectorCmd.Start(); err != nil {
			fmt.Printf("Warning: Failed to start MCP Inspector: %v\n", err)
			fmt.Println("You can manually run: npx @modelcontextprotocol/inspector --server-url " + serverURL)
			inspectorCmd = nil
		} else {
			fmt.Println("✓ MCP Inspector launched")
		}
	}
	return waitForDockerContainer(dockerCmd, containerName, inspectorCmd)
}

// waitForDockerContainer waits for the docker container to finish or handles CTRL+C
func waitForDockerContainer(dockerCmd *exec.Cmd, containerName string, inspectorCmd *exec.Cmd) error {
	// Create a channel to receive OS signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Create a channel to wait for the docker command to finish
	doneChan := make(chan error, 1)
	go func() {
		doneChan <- dockerCmd.Wait()
	}()

	// Wait for either signal or docker command to finish
	select {
	case <-sigChan:
		fmt.Println("\n\nReceived shutdown signal, stopping container...")

		// Stop the inspector if it's running
		if inspectorCmd != nil && inspectorCmd.Process != nil {
			fmt.Println("Stopping MCP Inspector...")
			if err := inspectorCmd.Process.Kill(); err != nil {
				fmt.Printf("Warning: Failed to stop MCP Inspector: %v\n", err)
			} else {
				_ = inspectorCmd.Wait()
				fmt.Println("✓ MCP Inspector stopped")
			}
		}

		// Stop the container
		fmt.Println("Stopping Docker container...")
		stopCmd := exec.Command("docker", "stop", containerName)
		if err := stopCmd.Run(); err != nil {
			fmt.Printf("Warning: Failed to stop container: %v\n", err)
		} else {
			fmt.Println("✓ Docker container stopped")
		}

		// Wait for the docker command to finish
		<-doneChan
		fmt.Println("\n✓ Cleanup completed successfully")
		return nil

	case err := <-doneChan:
		// Container exited on its own
		if inspectorCmd != nil && inspectorCmd.Process != nil {
			_ = inspectorCmd.Process.Kill()
			_ = inspectorCmd.Wait()
		}
		if err != nil {
			return fmt.Errorf("docker container exited with error: %w", err)
		}
		return nil
	}
}

// checkDockerImageExists verifies that a Docker image exists locally
func checkDockerImageExists(imageName string) error {
	cmd := exec.Command("docker", "image", "inspect", imageName)
	cmd.Stdout = nil
	cmd.Stderr = nil
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("image not found. run `arctl mcp build %s` to build the image", imageName)
	}
	return nil
}
