package devenv_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/rs/zerolog"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
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

// TestDevEnv starts a real envtest apiserver, registers every reconciler and
// the HTTP API, seeds sample data, then blocks until you kill it.
//
// Usage:
//
//	DEVENV=1 KUBEBUILDER_ASSETS=$(bin/setup-envtest use -p path) \
//	  go test -run TestDevEnv -timeout 30m -v ./test/devenv/
//
// or:
//
//	make test-dev-env
//
// Then in another terminal start the UI:
//
//	cd ui && NEXT_PUBLIC_API_URL=http://localhost:8080 npm run dev
//
// The test prints a kubeconfig path you can use with kubectl.
func TestDevEnv(t *testing.T) {
	if os.Getenv("DEVENV") == "" {
		t.Skip("set DEVENV=1 to run the interactive dev-env test")
	}

	logger := zerolog.New(os.Stderr).With().Timestamp().Logger()

	// --- envtest ---
	env := &envtest.Environment{
		CRDDirectoryPaths:    []string{"../../config/crd", "../../config/external-crds"},
		ErrorIfCRDPathMissing: true,
	}
	cfg, err := env.Start()
	if err != nil {
		t.Fatalf("start envtest: %v", err)
	}
	t.Cleanup(func() { env.Stop() })

	// --- manager ---
	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme:                 scheme,
		Metrics:                server.Options{BindAddress: "0"},
		HealthProbeBindAddress: "0",
		LeaderElection:         false,
	})
	if err != nil {
		t.Fatalf("create manager: %v", err)
	}

	if err := controller.SetupIndexes(mgr); err != nil {
		t.Fatalf("setup indexes: %v", err)
	}

	// --- reconcilers (same set as cmd/controller/main.go) ---
	ctrlLog := logger.With().Str("component", "controller").Logger()

	if err := (&controller.MCPServerCatalogReconciler{
		Client: mgr.GetClient(), Scheme: mgr.GetScheme(),
		Logger: ctrlLog.With().Str("controller", "mcpservercatalog").Logger(),
	}).SetupWithManager(mgr); err != nil {
		t.Fatalf("setup MCPServerCatalogReconciler: %v", err)
	}
	if err := (&controller.AgentCatalogReconciler{
		Client: mgr.GetClient(), Scheme: mgr.GetScheme(),
		Logger: ctrlLog.With().Str("controller", "agentcatalog").Logger(),
	}).SetupWithManager(mgr); err != nil {
		t.Fatalf("setup AgentCatalogReconciler: %v", err)
	}
	if err := (&controller.SkillCatalogReconciler{
		Client: mgr.GetClient(), Scheme: mgr.GetScheme(),
		Logger: ctrlLog.With().Str("controller", "skillcatalog").Logger(),
	}).SetupWithManager(mgr); err != nil {
		t.Fatalf("setup SkillCatalogReconciler: %v", err)
	}
	if err := (&controller.RegistryDeploymentReconciler{
		Client: mgr.GetClient(), Scheme: mgr.GetScheme(),
		Logger: ctrlLog.With().Str("controller", "registrydeployment").Logger(),
	}).SetupWithManager(mgr); err != nil {
		t.Fatalf("setup RegistryDeploymentReconciler: %v", err)
	}
	if err := (&controller.DiscoveryConfigReconciler{
		Client: mgr.GetClient(), Scheme: mgr.GetScheme(),
		Logger:  ctrlLog.With().Str("controller", "discoveryconfig").Logger(),
		Manager: mgr,
	}).SetupWithManager(mgr); err != nil {
		t.Fatalf("setup DiscoveryConfigReconciler: %v", err)
	}

	// remote client factory — points back at the same envtest cluster
	watchClient, err := client.NewWithWatch(cfg, client.Options{Scheme: scheme})
	if err != nil {
		t.Fatalf("create watch client: %v", err)
	}
	controller.RemoteClientFactory = cluster.NewFactory(watchClient, logger).CreateClientFunc()

	// --- HTTP API on :8080 ---
	os.Setenv("AGENTREGISTRY_DISABLE_AUTH", "true")
	httpServer := httpapi.NewServer(mgr.GetClient(), mgr.GetCache(), logger)
	if err := mgr.Add(httpServer.Runnable(":8080")); err != nil {
		t.Fatalf("add HTTP API: %v", err)
	}

	// --- start manager ---
	mgrCtx, mgrCancel := context.WithCancel(context.Background())
	t.Cleanup(mgrCancel)
	go func() {
		if err := mgr.Start(mgrCtx); err != nil {
			t.Logf("manager: %v", err)
		}
	}()

	syncCtx, syncCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer syncCancel()
	if !mgr.GetCache().WaitForCacheSync(syncCtx) {
		t.Fatal("cache sync timeout")
	}

	// --- seed ---
	if err := seed(context.Background(), mgr.GetClient()); err != nil {
		t.Fatalf("seed: %v", err)
	}

	// --- kubeconfig for kubectl ---
	kubeconfigPath := t.TempDir() + "/kubeconfig.yaml"
	if err := os.WriteFile(kubeconfigPath, env.KubeConfig, 0600); err != nil {
		t.Fatalf("write kubeconfig: %v", err)
	}

	t.Logf("")
	t.Logf("════════════════════════════════════════════════════════")
	t.Logf("  Dev environment ready")
	t.Logf("════════════════════════════════════════════════════════")
	t.Logf("  API:        http://localhost:8080")
	t.Logf("  Kubeconfig: %s", kubeconfigPath)
	t.Logf("  UI:         cd ui && NEXT_PUBLIC_API_URL=http://localhost:8080 npm run dev")
	t.Logf("")
	t.Logf("  Ctrl+C or -timeout to stop")
	t.Logf("════════════════════════════════════════════════════════")

	// block until killed
	select {}
}

// ---------------------------------------------------------------------------
// seed
// ---------------------------------------------------------------------------

func metadata(orgVerified, publisherVerified bool) *apiextensionsv1.JSON {
	raw, _ := json.Marshal(map[string]any{
		"io.modelcontextprotocol.registry/publisher-provided": map[string]any{
			"aregistry.ai/metadata": map[string]any{
				"identity": map[string]any{
					"org_is_verified":                    orgVerified,
					"publisher_identity_verified_by_jwt": publisherVerified,
				},
			},
		},
	})
	return &apiextensionsv1.JSON{Raw: raw}
}

func seed(ctx context.Context, c client.Client) error {
	now := metav1.Now()
	ns := "agentregistry"

	// namespaces
	for _, name := range []string{"agentregistry", "dev", "staging", "prod"} {
		_ = c.Create(ctx, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{
			Name: name, Labels: map[string]string{"environment": name},
		}})
	}

	// published entries per env
	for _, env := range []string{"dev", "staging", "prod"} {
		servers := []*agentregistryv1alpha1.MCPServerCatalog{
			mcpServer(env, "filesystem", "1.0.0", fmt.Sprintf("File system operations in %s", env), metadata(true, false),
				&agentregistryv1alpha1.Package{RegistryType: "npm", Identifier: "@modelcontextprotocol/server-filesystem", Transport: agentregistryv1alpha1.Transport{Type: "stdio"}}, nil),
			mcpServer(env, "github", "2.1.0", fmt.Sprintf("GitHub API integration in %s", env), metadata(true, true),
				nil, &agentregistryv1alpha1.Transport{Type: "streamable-http", URL: fmt.Sprintf("https://mcp.%s.example.com/github", env)}),
			mcpServer(env, "slack", "1.3.0", fmt.Sprintf("Slack messaging integration in %s", env), metadata(false, true),
				&agentregistryv1alpha1.Package{RegistryType: "npm", Identifier: "@modelcontextprotocol/server-slack", Transport: agentregistryv1alpha1.Transport{Type: "stdio"}}, nil),
			mcpServer(env, "postgres", "0.9.0", fmt.Sprintf("PostgreSQL database operations in %s", env), nil,
				&agentregistryv1alpha1.Package{RegistryType: "npm", Identifier: "@modelcontextprotocol/server-postgres", Transport: agentregistryv1alpha1.Transport{Type: "stdio"}}, nil),
		}
		if err := publishServers(ctx, c, servers, &now); err != nil {
			return err
		}

		agents := []*agentregistryv1alpha1.AgentCatalog{
			agentCatalog(env, "research-agent", "0.5.0", "ghcr.io/example/research-agent:0.5.0", "langgraph", "anthropic"),
			agentCatalog(env, "code-review-agent", "1.2.0", "ghcr.io/example/code-review-agent:1.2.0", "autogen", "openai"),
		}
		if err := publishAgents(ctx, c, agents, &now); err != nil {
			return err
		}

		skills := []*agentregistryv1alpha1.SkillCatalog{
			skillCatalog(env, "terraform-skill", "1.5.0", "infrastructure", metadata(true, false)),
			skillCatalog(env, "sql-query-skill", "0.8.0", "data", metadata(true, true)),
			skillCatalog(env, "kubernetes-skill", "1.0.0", "infrastructure", metadata(false, true)),
			skillCatalog(env, "python-skill", "0.5.0", "development", nil),
		}
		if err := publishSkills(ctx, c, skills, &now); err != nil {
			return err
		}

		// deployments
		for _, d := range []*agentregistryv1alpha1.RegistryDeployment{
			regDeploy(env, fmt.Sprintf("filesystem-server-%s", env), "1.0.0", agentregistryv1alpha1.ResourceTypeMCP),
			regDeploy(env, fmt.Sprintf("research-agent-%s", env), "0.5.0", agentregistryv1alpha1.ResourceTypeAgent),
		} {
			if err := c.Create(ctx, d); err != nil {
				return fmt.Errorf("create deployment %s: %w", d.Name, err)
			}
		}
	}

	// inventory-only (dev, unpublished, no deployments)
	for _, s := range []*agentregistryv1alpha1.MCPServerCatalog{
		{
			ObjectMeta: metav1.ObjectMeta{Name: "sqlite-server-dev-v0.1.0", Namespace: ns, Labels: map[string]string{"environment": "dev"}},
			Spec: agentregistryv1alpha1.MCPServerCatalogSpec{
				Name: "sqlite-server-dev", Version: "0.1.0",
				Title: "SQLite MCP Server (dev)", Description: "SQLite database operations — work in progress",
				Packages: []agentregistryv1alpha1.Package{{RegistryType: "npm", Identifier: "@modelcontextprotocol/server-sqlite", Transport: agentregistryv1alpha1.Transport{Type: "stdio"}}},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{Name: "brave-search-server-dev-v0.2.0", Namespace: ns, Labels: map[string]string{"environment": "dev"}},
			Spec: agentregistryv1alpha1.MCPServerCatalogSpec{
				Name: "brave-search-server-dev", Version: "0.2.0",
				Title: "Brave Search MCP Server (dev)", Description: "Brave Search API integration — early draft",
				Remotes: []agentregistryv1alpha1.Transport{{Type: "streamable-http", URL: "https://mcp.dev.example.com/brave-search"}},
			},
		},
	} {
		if err := c.Create(ctx, s); err != nil {
			return fmt.Errorf("create inventory server %s: %w", s.Name, err)
		}
	}
	if err := c.Create(ctx, &agentregistryv1alpha1.AgentCatalog{
		ObjectMeta: metav1.ObjectMeta{Name: "summarizer-agent-dev-v0.1.0", Namespace: ns, Labels: map[string]string{"environment": "dev"}},
		Spec: agentregistryv1alpha1.AgentCatalogSpec{
			Name: "summarizer-agent-dev", Version: "0.1.0",
			Title: "Summarizer Agent (dev)", Description: "Text summarization agent — not yet published",
			Image: "ghcr.io/example/summarizer-agent:0.1.0", Framework: "langgraph", ModelProvider: "anthropic",
		},
	}); err != nil {
		return fmt.Errorf("create inventory agent: %w", err)
	}
	if err := c.Create(ctx, &agentregistryv1alpha1.SkillCatalog{
		ObjectMeta: metav1.ObjectMeta{Name: "csv-parser-skill-dev-v0.1.0", Namespace: ns, Labels: map[string]string{"environment": "dev"}},
		Spec: agentregistryv1alpha1.SkillCatalogSpec{
			Name: "csv-parser-skill-dev", Version: "0.1.0",
			Title: "CSV Parser Skill (dev)", Category: "data", Description: "CSV parsing and transformation — draft",
		},
	}); err != nil {
		return fmt.Errorf("create inventory skill: %w", err)
	}

	// models (cluster-scoped)
	models := []*agentregistryv1alpha1.ModelCatalog{
		{ObjectMeta: metav1.ObjectMeta{Name: "claude-3-opus-prod"}, Spec: agentregistryv1alpha1.ModelCatalogSpec{Name: "claude-3-opus-prod", Provider: "Anthropic", Model: "claude-3-opus-20240229", Description: "Claude 3 Opus"}},
		{ObjectMeta: metav1.ObjectMeta{Name: "gpt-4-dev"}, Spec: agentregistryv1alpha1.ModelCatalogSpec{Name: "gpt-4-dev", Provider: "OpenAI", Model: "gpt-4-turbo", Description: "GPT-4 Turbo"}},
		{ObjectMeta: metav1.ObjectMeta{Name: "llama3-local"}, Spec: agentregistryv1alpha1.ModelCatalogSpec{Name: "llama3-local", Provider: "Ollama", Model: "llama3:70b", BaseURL: "http://localhost:11434", Description: "Local Llama 3"}},
	}
	for _, m := range models {
		if err := c.Create(ctx, m); err != nil {
			return fmt.Errorf("create model %s: %w", m.Name, err)
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

	// discovery config
	return c.Create(ctx, &agentregistryv1alpha1.DiscoveryConfig{
		ObjectMeta: metav1.ObjectMeta{Name: "environments", Namespace: "agentregistry"},
		Spec: agentregistryv1alpha1.DiscoveryConfigSpec{
			Environments: []agentregistryv1alpha1.Environment{
				{Name: "dev", Cluster: agentregistryv1alpha1.ClusterConfig{Name: "local", Namespace: "dev"}, Namespaces: []string{"dev"}, DiscoveryEnabled: true},
				{Name: "staging", Cluster: agentregistryv1alpha1.ClusterConfig{Name: "local", Namespace: "staging"}, Namespaces: []string{"staging"}, DiscoveryEnabled: true},
				{Name: "prod", Cluster: agentregistryv1alpha1.ClusterConfig{Name: "local", Namespace: "prod"}, Namespaces: []string{"prod"}, DiscoveryEnabled: true},
			},
		},
	})
}

// ---------------------------------------------------------------------------
// builders
// ---------------------------------------------------------------------------

func mcpServer(env, shortName, version, desc string, meta *apiextensionsv1.JSON, pkg *agentregistryv1alpha1.Package, remote *agentregistryv1alpha1.Transport) *agentregistryv1alpha1.MCPServerCatalog {
	name := fmt.Sprintf("%s-server-%s", shortName, env)
	s := &agentregistryv1alpha1.MCPServerCatalog{
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("%s-v%s", name, version), Namespace: "agentregistry",
			Labels: map[string]string{"environment": env},
		},
		Spec: agentregistryv1alpha1.MCPServerCatalogSpec{
			Name: name, Version: version,
			Title: fmt.Sprintf("%s MCP Server (%s)", capitalize(shortName), env),
			Description: desc, Metadata: meta,
		},
	}
	if pkg != nil {
		s.Spec.Packages = []agentregistryv1alpha1.Package{*pkg}
	}
	if remote != nil {
		s.Spec.Remotes = []agentregistryv1alpha1.Transport{*remote}
	}
	return s
}

func agentCatalog(env, name, version, image, framework, provider string) *agentregistryv1alpha1.AgentCatalog {
	return &agentregistryv1alpha1.AgentCatalog{
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("%s-%s-v%s", name, env, version), Namespace: "agentregistry",
			Labels: map[string]string{"environment": env},
		},
		Spec: agentregistryv1alpha1.AgentCatalogSpec{
			Name: fmt.Sprintf("%s-%s", name, env), Version: version,
			Title: fmt.Sprintf("%s (%s)", capitalize(name), env),
			Description: fmt.Sprintf("%s in %s", capitalize(name), env),
			Image: image, Framework: framework, ModelProvider: provider,
		},
	}
}

func skillCatalog(env, name, version, category string, meta *apiextensionsv1.JSON) *agentregistryv1alpha1.SkillCatalog {
	return &agentregistryv1alpha1.SkillCatalog{
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("%s-%s-v%s", name, env, version), Namespace: "agentregistry",
			Labels: map[string]string{"environment": env},
		},
		Spec: agentregistryv1alpha1.SkillCatalogSpec{
			Name: fmt.Sprintf("%s-%s", name, env), Version: version,
			Title: fmt.Sprintf("%s (%s)", capitalize(name), env),
			Category: category,
			Description: fmt.Sprintf("%s in %s", capitalize(name), env),
			Metadata: meta,
		},
	}
}

func regDeploy(env, resourceName, version string, resType agentregistryv1alpha1.ResourceType) *agentregistryv1alpha1.RegistryDeployment {
	return &agentregistryv1alpha1.RegistryDeployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("%s-%s-deploy", resourceName, env), Namespace: "agentregistry",
			Labels: map[string]string{"environment": env},
		},
		Spec: agentregistryv1alpha1.RegistryDeploymentSpec{
			ResourceName: resourceName, Version: version,
			ResourceType: resType, Runtime: agentregistryv1alpha1.RuntimeTypeKubernetes,
			Namespace: env,
		},
	}
}

// ---------------------------------------------------------------------------
// publish helpers (status is a subresource)
// ---------------------------------------------------------------------------

func publishServers(ctx context.Context, c client.Client, servers []*agentregistryv1alpha1.MCPServerCatalog, now *metav1.Time) error {
	for _, s := range servers {
		if err := c.Create(ctx, s); err != nil {
			return fmt.Errorf("create server %s: %w", s.Name, err)
		}
	}
	time.Sleep(100 * time.Millisecond)
	for _, s := range servers {
		if err := c.Get(ctx, client.ObjectKeyFromObject(s), s); err != nil {
			return err
		}
		s.Status.Published = true
		s.Status.IsLatest = true
		s.Status.PublishedAt = now
		s.Status.Status = agentregistryv1alpha1.CatalogStatusActive
		if err := c.Status().Update(ctx, s); err != nil {
			return err
		}
	}
	return nil
}

func publishAgents(ctx context.Context, c client.Client, agents []*agentregistryv1alpha1.AgentCatalog, now *metav1.Time) error {
	for _, a := range agents {
		if err := c.Create(ctx, a); err != nil {
			return fmt.Errorf("create agent %s: %w", a.Name, err)
		}
	}
	time.Sleep(100 * time.Millisecond)
	for _, a := range agents {
		if err := c.Get(ctx, client.ObjectKeyFromObject(a), a); err != nil {
			return err
		}
		a.Status.Published = true
		a.Status.IsLatest = true
		a.Status.PublishedAt = now
		a.Status.Status = agentregistryv1alpha1.CatalogStatusActive
		if err := c.Status().Update(ctx, a); err != nil {
			return err
		}
	}
	return nil
}

func publishSkills(ctx context.Context, c client.Client, skills []*agentregistryv1alpha1.SkillCatalog, now *metav1.Time) error {
	for _, s := range skills {
		if err := c.Create(ctx, s); err != nil {
			return fmt.Errorf("create skill %s: %w", s.Name, err)
		}
	}
	time.Sleep(100 * time.Millisecond)
	for _, s := range skills {
		if err := c.Get(ctx, client.ObjectKeyFromObject(s), s); err != nil {
			return err
		}
		s.Status.Published = true
		s.Status.IsLatest = true
		s.Status.PublishedAt = now
		s.Status.Status = agentregistryv1alpha1.CatalogStatusActive
		if err := c.Status().Update(ctx, s); err != nil {
			return err
		}
	}
	return nil
}

func capitalize(s string) string {
	if s == "" {
		return s
	}
	return string(s[0]-32) + s[1:]
}
