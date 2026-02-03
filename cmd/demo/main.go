package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/go-logr/zerologr"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/metrics/server"

	agentregistryv1alpha1 "github.com/agentregistry-dev/agentregistry/api/v1alpha1"
	"github.com/agentregistry-dev/agentregistry/internal/cluster"
	"github.com/agentregistry-dev/agentregistry/internal/controller"
	"github.com/agentregistry-dev/agentregistry/internal/httpapi"

	kagentv1alpha2 "github.com/kagent-dev/kagent/go/api/v1alpha2"
	kmcpv1alpha1 "github.com/kagent-dev/kmcp/api/v1alpha1"
)

var scheme = runtime.NewScheme()

func init() {
	_ = clientgoscheme.AddToScheme(scheme)
	_ = agentregistryv1alpha1.AddToScheme(scheme)
	_ = kagentv1alpha2.AddToScheme(scheme)
	_ = kmcpv1alpha1.AddToScheme(scheme)
}

func main() {
	var (
		httpAPIAddr string
		skipUI      bool
	)

	flag.StringVar(&httpAPIAddr, "http-api-address", ":8080", "The address the HTTP API server binds to.")
	flag.BoolVar(&skipUI, "skip-ui", false, "Skip starting the UI dev server.")
	flag.Parse()

	// Set up logging
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnixMs
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: "15:04:05"}).With().Timestamp().Logger()
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	logf.SetLogger(zerologr.New(&log.Logger))

	log.Info().Msg("=== Agent Registry Demo ===")

	// Disable auth for demo
	os.Setenv("AGENTREGISTRY_DISABLE_AUTH", "true")

	// Find project root
	projectRoot, err := findProjectRoot()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to find project root")
	}

	// Start envtest
	log.Info().Msg("starting envtest...")
	env := &envtest.Environment{
		CRDDirectoryPaths: []string{
			filepath.Join(projectRoot, "config", "crd"),
			filepath.Join(projectRoot, "config", "external-crds"),
		},
		ErrorIfCRDPathMissing: true,
	}

	config, err := env.Start()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to start envtest")
	}
	log.Info().Str("host", config.Host).Msg("envtest started")

	// Write kubeconfig
	kubeconfigPath, err := writeKubeconfig(env)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to write kubeconfig")
	}

	// Create manager
	mgr, err := ctrl.NewManager(config, ctrl.Options{
		Scheme:                 scheme,
		Metrics:                server.Options{BindAddress: "0"},
		HealthProbeBindAddress: "0",
		LeaderElection:         false,
	})
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create manager")
	}

	// Setup indexes and reconcilers
	if err := controller.SetupIndexes(mgr); err != nil {
		log.Fatal().Err(err).Msg("failed to setup indexes")
	}
	setupReconcilers(mgr, log.Logger)

	// Create a client.WithWatch for the local cluster
	localClient, err := client.NewWithWatch(config, client.Options{Scheme: scheme})
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create local watch client")
	}

	// Initialize remote client factory
	clusterFactory := cluster.NewFactory(localClient, log.Logger)
	controller.RemoteClientFactory = clusterFactory.CreateClientFunc()

	// Setup HTTP API
	httpServer := httpapi.NewServer(mgr.GetClient(), mgr.GetCache(), log.Logger)
	if err := mgr.Add(httpServer.Runnable(httpAPIAddr)); err != nil {
		log.Fatal().Err(err).Msg("failed to add HTTP API server")
	}

	// Start manager
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		if err := mgr.Start(ctx); err != nil {
			log.Error().Err(err).Msg("manager error")
		}
	}()

	// Wait for cache
	time.Sleep(500 * time.Millisecond)
	if !mgr.GetCache().WaitForCacheSync(ctx) {
		log.Fatal().Msg("cache sync timeout")
	}

	// Create sample resources
	log.Info().Msg("creating sample resources...")
	if err := createSampleResources(ctx, mgr.GetClient()); err != nil {
		log.Fatal().Err(err).Msg("failed to create sample resources")
	}

	// Start UI
	var uiCmd *exec.Cmd
	if !skipUI {
		log.Info().Msg("starting UI...")
		uiCmd = exec.Command("npm", "run", "dev")
		uiCmd.Dir = filepath.Join(projectRoot, "ui")
		uiCmd.Env = append(os.Environ(), "NEXT_PUBLIC_API_URL=http://localhost:8080")
		uiCmd.Stdout = os.Stdout
		uiCmd.Stderr = os.Stderr
		uiCmd.Start()
	}

	// Print info
	fmt.Println()
	fmt.Println("════════════════════════════════════════════════════════")
	fmt.Println("  Demo Running - Multi-Namespace Setup")
	fmt.Println("════════════════════════════════════════════════════════")
	fmt.Printf("  API:        http://localhost%s\n", httpAPIAddr)
	if !skipUI {
		fmt.Println("  UI:         http://localhost:3000")
	}
	fmt.Printf("  Kubeconfig: %s\n", kubeconfigPath)
	fmt.Println()
	fmt.Println("  Catalog resources in 'agentregistry' namespace")
	fmt.Println("  DiscoveryConfig watching: dev, staging, prod namespaces")
	fmt.Println()
	fmt.Println("  Example commands:")
	fmt.Println("  kubectl --kubeconfig=" + kubeconfigPath + " get mcpservercatalog -n agentregistry")
	fmt.Println("  kubectl --kubeconfig=" + kubeconfigPath + " get registrydeployment -n agentregistry")
	fmt.Println("  kubectl --kubeconfig=" + kubeconfigPath + " get discoveryconfig")
	fmt.Println("  curl http://localhost:8080/v0/servers")
	fmt.Println("  curl http://localhost:8080/v0/environments")
	fmt.Println()
	fmt.Println("  Press Ctrl+C to stop")
	fmt.Println("════════════════════════════════════════════════════════")

	// Wait for signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Info().Msg("shutting down...")
	if uiCmd != nil && uiCmd.Process != nil {
		uiCmd.Process.Signal(syscall.SIGTERM)
	}
	cancel()
	env.Stop()
	os.Remove(kubeconfigPath)
}

func findProjectRoot() (string, error) {
	dir, _ := os.Getwd()
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("project root not found")
		}
		dir = parent
	}
}

func writeKubeconfig(env *envtest.Environment) (string, error) {
	if len(env.KubeConfig) == 0 {
		return "", fmt.Errorf("no kubeconfig from envtest")
	}
	// Use fixed path in current directory for easy access
	kubeconfigPath := "./demo-kubeconfig.yaml"
	if err := os.WriteFile(kubeconfigPath, env.KubeConfig, 0600); err != nil {
		return "", err
	}
	return kubeconfigPath, nil
}

func setupReconcilers(mgr ctrl.Manager, logger zerolog.Logger) {
	(&controller.MCPServerCatalogReconciler{Client: mgr.GetClient(), Scheme: mgr.GetScheme(), Logger: logger}).SetupWithManager(mgr)
	(&controller.AgentCatalogReconciler{Client: mgr.GetClient(), Scheme: mgr.GetScheme(), Logger: logger}).SetupWithManager(mgr)
	(&controller.SkillCatalogReconciler{Client: mgr.GetClient(), Scheme: mgr.GetScheme(), Logger: logger}).SetupWithManager(mgr)
	(&controller.RegistryDeploymentReconciler{Client: mgr.GetClient(), Scheme: mgr.GetScheme(), Logger: logger}).SetupWithManager(mgr)
	(&controller.DiscoveryConfigReconciler{Client: mgr.GetClient(), Scheme: mgr.GetScheme(), Logger: logger}).SetupWithManager(mgr)
}

func createSampleResources(ctx context.Context, c client.Client) error {
	now := metav1.Now()

	// Create agentregistry namespace for all Agent Registry resources
	agentregistryNs := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "agentregistry",
			Labels: map[string]string{
				"app": "agentregistry",
			},
		},
	}
	if err := c.Create(ctx, agentregistryNs); err != nil {
		log.Warn().Err(err).Msg("agentregistry namespace may already exist")
	}

	// Create namespaces for multi-environment demo (for discovered resources)
	namespaces := []string{"dev", "staging", "prod"}
	for _, ns := range namespaces {
		namespace := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: ns,
				Labels: map[string]string{
					"environment": ns,
				},
			},
		}
		if err := c.Create(ctx, namespace); err != nil {
			log.Warn().Err(err).Str("namespace", ns).Msg("namespace may already exist")
		}
	}
	log.Info().Msg("created namespaces: agentregistry, dev, staging, prod")

	// Track all created resources for final count
	var allServers, allAgents, allSkills, allDeployments int

	// Create catalog resources in agentregistry namespace
	// These represent the centralized catalog that Agent Registry manages
	catalogNamespace := "agentregistry"

	// Create resources per environment in the catalog
	for _, ns := range namespaces {
		log.Info().Str("environment", ns).Msg("creating catalog entries for environment")

		// MCP Servers - all in agentregistry namespace
		servers := []*agentregistryv1alpha1.MCPServerCatalog{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      fmt.Sprintf("filesystem-server-%s-v1.0.0", ns),
					Namespace: catalogNamespace,
					Labels:    map[string]string{"environment": ns},
				},
				Spec: agentregistryv1alpha1.MCPServerCatalogSpec{
					Name:        fmt.Sprintf("filesystem-server-%s", ns),
					Version:     "1.0.0",
					Title:       fmt.Sprintf("Filesystem MCP Server (%s)", ns),
					Description: fmt.Sprintf("File system operations for AI agents in %s", ns),
					Packages: []agentregistryv1alpha1.Package{
						{
							RegistryType: "npm",
							Identifier:   "@modelcontextprotocol/server-filesystem",
							Transport:    agentregistryv1alpha1.Transport{Type: "stdio"},
						},
					},
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      fmt.Sprintf("github-server-%s-v2.1.0", ns),
					Namespace: catalogNamespace,
					Labels:    map[string]string{"environment": ns},
				},
				Spec: agentregistryv1alpha1.MCPServerCatalogSpec{
					Name:        fmt.Sprintf("github-server-%s", ns),
					Version:     "2.1.0",
					Title:       fmt.Sprintf("GitHub MCP Server (%s)", ns),
					Description: fmt.Sprintf("GitHub API integration in %s", ns),
					Remotes: []agentregistryv1alpha1.Transport{
						{Type: "streamable-http", URL: fmt.Sprintf("https://mcp.%s.example.com/github", ns)},
					},
				},
			},
		}

		for _, s := range servers {
			if err := c.Create(ctx, s); err != nil {
				return err
			}
		}
		time.Sleep(100 * time.Millisecond)
		for _, s := range servers {
			if err := c.Get(ctx, client.ObjectKeyFromObject(s), s); err != nil {
				return err
			}
			s.Status.Published = true
			s.Status.IsLatest = true
			s.Status.PublishedAt = &now
			s.Status.Status = agentregistryv1alpha1.CatalogStatusActive
			if err := c.Status().Update(ctx, s); err != nil {
				return err
			}
		}
		allServers += len(servers)

		// Agents - all in agentregistry namespace
		agents := []*agentregistryv1alpha1.AgentCatalog{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      fmt.Sprintf("research-agent-%s-v0.5.0", ns),
					Namespace: catalogNamespace,
					Labels:    map[string]string{"environment": ns},
				},
				Spec: agentregistryv1alpha1.AgentCatalogSpec{
					Name:          fmt.Sprintf("research-agent-%s", ns),
					Version:       "0.5.0",
					Title:         fmt.Sprintf("Research Agent (%s)", ns),
					Description:   fmt.Sprintf("AI research assistant in %s", ns),
					Image:         "ghcr.io/example/research-agent:0.5.0",
					Framework:     "langgraph",
					ModelProvider: "anthropic",
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      fmt.Sprintf("code-review-agent-%s-v1.2.0", ns),
					Namespace: catalogNamespace,
					Labels:    map[string]string{"environment": ns},
				},
				Spec: agentregistryv1alpha1.AgentCatalogSpec{
					Name:          fmt.Sprintf("code-review-agent-%s", ns),
					Version:       "1.2.0",
					Title:         fmt.Sprintf("Code Review Agent (%s)", ns),
					Description:   fmt.Sprintf("Automated code review in %s", ns),
					Image:         "ghcr.io/example/code-review-agent:1.2.0",
					Framework:     "autogen",
					ModelProvider: "openai",
				},
			},
		}

		for _, a := range agents {
			if err := c.Create(ctx, a); err != nil {
				return err
			}
		}
		time.Sleep(100 * time.Millisecond)
		for _, a := range agents {
			if err := c.Get(ctx, client.ObjectKeyFromObject(a), a); err != nil {
				return err
			}
			a.Status.Published = true
			a.Status.IsLatest = true
			a.Status.PublishedAt = &now
			a.Status.Status = agentregistryv1alpha1.CatalogStatusActive
			if err := c.Status().Update(ctx, a); err != nil {
				return err
			}
		}
		allAgents += len(agents)

		// Skills - all in agentregistry namespace
		skills := []*agentregistryv1alpha1.SkillCatalog{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      fmt.Sprintf("terraform-skill-%s-v1.5.0", ns),
					Namespace: catalogNamespace,
					Labels:    map[string]string{"environment": ns},
				},
				Spec: agentregistryv1alpha1.SkillCatalogSpec{
					Name:        fmt.Sprintf("terraform-skill-%s", ns),
					Version:     "1.5.0",
					Title:       fmt.Sprintf("Terraform Skill (%s)", ns),
					Category:    "infrastructure",
					Description: fmt.Sprintf("Infrastructure management in %s", ns),
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      fmt.Sprintf("sql-query-skill-%s-v0.8.0", ns),
					Namespace: catalogNamespace,
					Labels:    map[string]string{"environment": ns},
				},
				Spec: agentregistryv1alpha1.SkillCatalogSpec{
					Name:        fmt.Sprintf("sql-query-skill-%s", ns),
					Version:     "0.8.0",
					Title:       fmt.Sprintf("SQL Query Skill (%s)", ns),
					Category:    "data",
					Description: fmt.Sprintf("SQL query generation in %s", ns),
				},
			},
		}

		for _, s := range skills {
			if err := c.Create(ctx, s); err != nil {
				return err
			}
		}
		time.Sleep(100 * time.Millisecond)
		for _, s := range skills {
			if err := c.Get(ctx, client.ObjectKeyFromObject(s), s); err != nil {
				return err
			}
			s.Status.Published = true
			s.Status.IsLatest = true
			s.Status.PublishedAt = &now
			s.Status.Status = agentregistryv1alpha1.CatalogStatusActive
			if err := c.Status().Update(ctx, s); err != nil {
				return err
			}
		}
		allSkills += len(skills)

		// RegistryDeployments - in agentregistry namespace, deploying to target namespaces
		deployments := []*agentregistryv1alpha1.RegistryDeployment{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      fmt.Sprintf("filesystem-server-%s-deploy", ns),
					Namespace: catalogNamespace,
					Labels:    map[string]string{"environment": ns},
				},
				Spec: agentregistryv1alpha1.RegistryDeploymentSpec{
					ResourceName: fmt.Sprintf("filesystem-server-%s", ns),
					Version:      "1.0.0",
					ResourceType: agentregistryv1alpha1.ResourceTypeMCP,
					Runtime:      agentregistryv1alpha1.RuntimeTypeKubernetes,
					Namespace:    ns, // Deploy TO this namespace
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      fmt.Sprintf("research-agent-%s-deploy", ns),
					Namespace: catalogNamespace,
					Labels:    map[string]string{"environment": ns},
				},
				Spec: agentregistryv1alpha1.RegistryDeploymentSpec{
					ResourceName: fmt.Sprintf("research-agent-%s", ns),
					Version:      "0.5.0",
					ResourceType: agentregistryv1alpha1.ResourceTypeAgent,
					Runtime:      agentregistryv1alpha1.RuntimeTypeKubernetes,
					Namespace:    ns, // Deploy TO this namespace
				},
			},
		}

		for _, d := range deployments {
			if err := c.Create(ctx, d); err != nil {
				return err
			}
		}
		allDeployments += len(deployments)
	}

	// Models (cluster-scoped)
	models := []*agentregistryv1alpha1.ModelCatalog{
		{
			ObjectMeta: metav1.ObjectMeta{Name: "claude-3-opus-prod"},
			Spec: agentregistryv1alpha1.ModelCatalogSpec{
				Name:        "claude-3-opus-prod",
				Provider:    "Anthropic",
				Model:       "claude-3-opus-20240229",
				Description: "Claude 3 Opus",
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{Name: "gpt-4-dev"},
			Spec: agentregistryv1alpha1.ModelCatalogSpec{
				Name:        "gpt-4-dev",
				Provider:    "OpenAI",
				Model:       "gpt-4-turbo",
				Description: "GPT-4 Turbo",
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{Name: "llama3-local"},
			Spec: agentregistryv1alpha1.ModelCatalogSpec{
				Name:        "llama3-local",
				Provider:    "Ollama",
				Model:       "llama3:70b",
				BaseURL:     "http://localhost:11434",
				Description: "Local Llama 3",
			},
		},
	}
	for _, m := range models {
		if err := c.Create(ctx, m); err != nil {
			return err
		}
	}
	time.Sleep(100 * time.Millisecond)
	for _, m := range models {
		if err := c.Get(ctx, client.ObjectKeyFromObject(m), m); err != nil {
			return err
		}
		m.Status.Published = true
		m.Status.PublishedAt = &now
		m.Status.Status = agentregistryv1alpha1.CatalogStatusActive
		m.Status.Ready = true
		if err := c.Status().Update(ctx, m); err != nil {
			return err
		}
	}

	// Create DiscoveryConfig for multi-namespace discovery
	discoveryConfig := &agentregistryv1alpha1.DiscoveryConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name: "multi-namespace-discovery",
		},
		Spec: agentregistryv1alpha1.DiscoveryConfigSpec{
			Environments: []agentregistryv1alpha1.Environment{
				{
					Name: "agentregistry",
					Cluster: agentregistryv1alpha1.ClusterConfig{
						Name:                "local",
						Namespace:           "agentregistry",
						UseWorkloadIdentity: false,
					},
					Namespaces:       []string{"agentregistry"},
					ResourceTypes:    []string{"MCPServer", "Agent", "ModelConfig"},
					DiscoveryEnabled: true,
					Labels: map[string]string{
						"environment": "agentregistry",
						"demo":        "true",
					},
				},
				{
					Name: "dev",
					Cluster: agentregistryv1alpha1.ClusterConfig{
						Name:                "local",
						Namespace:           "dev",
						UseWorkloadIdentity: false,
					},
					Namespaces:       []string{"dev"},
					ResourceTypes:    []string{"MCPServer", "Agent", "ModelConfig"},
					DiscoveryEnabled: true,
					Labels: map[string]string{
						"environment": "dev",
						"demo":        "true",
					},
				},
				{
					Name: "staging",
					Cluster: agentregistryv1alpha1.ClusterConfig{
						Name:                "local",
						Namespace:           "staging",
						UseWorkloadIdentity: false,
					},
					Namespaces:       []string{"staging"},
					ResourceTypes:    []string{"MCPServer", "Agent", "ModelConfig"},
					DiscoveryEnabled: true,
					Labels: map[string]string{
						"environment": "staging",
						"demo":        "true",
					},
				},
				{
					Name: "prod",
					Cluster: agentregistryv1alpha1.ClusterConfig{
						Name:                "local",
						Namespace:           "prod",
						UseWorkloadIdentity: false,
					},
					Namespaces:       []string{"prod"},
					ResourceTypes:    []string{"MCPServer", "Agent", "ModelConfig"},
					DiscoveryEnabled: true,
					Labels: map[string]string{
						"environment": "prod",
						"demo":        "true",
					},
				},
			},
		},
	}
	if err := c.Create(ctx, discoveryConfig); err != nil {
		log.Warn().Err(err).Msg("failed to create DiscoveryConfig (may already exist)")
	} else {
		log.Info().Str("name", discoveryConfig.Name).Int("environments", len(discoveryConfig.Spec.Environments)).Msg("created DiscoveryConfig")
	}

	log.Info().
		Int("servers", allServers).
		Int("agents", allAgents).
		Int("skills", allSkills).
		Int("models", len(models)).
		Int("deployments", allDeployments).
		Int("namespaces", len(namespaces)).
		Msg("sample resources created")
	return nil
}
