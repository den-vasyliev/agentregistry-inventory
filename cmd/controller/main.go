package main

import (
	"flag"
	"os"

	// Import all Kubernetes client auth plugins
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"github.com/go-logr/zerologr"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/metrics/server"

	agentregistryv1alpha1 "github.com/agentregistry-dev/agentregistry/api/v1alpha1"
	"github.com/agentregistry-dev/agentregistry/internal/controller"
	"github.com/agentregistry-dev/agentregistry/internal/httpapi"
	"github.com/agentregistry-dev/agentregistry/internal/version"

	kagentv1alpha2 "github.com/kagent-dev/kagent/go/api/v1alpha2"
	kmcpv1alpha1 "github.com/kagent-dev/kmcp/api/v1alpha1"
)

var (
	scheme = runtime.NewScheme()
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(agentregistryv1alpha1.AddToScheme(scheme))
	utilruntime.Must(kagentv1alpha2.AddToScheme(scheme))
	utilruntime.Must(kmcpv1alpha1.AddToScheme(scheme))
}

func main() {
	var (
		metricsAddr          string
		enableLeaderElection bool
		probeAddr            string
		httpAPIAddr          string
		enableHTTPAPI        bool
	)

	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8081", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8082", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.StringVar(&httpAPIAddr, "http-api-address", ":8080", "The address the HTTP API server binds to.")
	flag.BoolVar(&enableHTTPAPI, "enable-http-api", true, "Enable the HTTP API server.")

	// Set up structured logging with zerolog before parsing flags
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = zerolog.New(os.Stdout).With().Timestamp().Caller().Logger()

	// Set up controller-runtime logger using zerologr
	logf.SetLogger(zerologr.New(&log.Logger))

	// Parse flags (controller-runtime adds --kubeconfig flag automatically)
	flag.Parse()

	log.Info().
		Str("version", version.Version).
		Str("commit", version.GitCommit).
		Str("build-date", version.BuildDate).
		Str("metrics-addr", metricsAddr).
		Str("probe-addr", probeAddr).
		Bool("leader-elect", enableLeaderElection).
		Str("http-api-addr", httpAPIAddr).
		Bool("enable-http-api", enableHTTPAPI).
		Msg("starting agent registry controller")

	// Get Kubernetes config (uses --kubeconfig flag or KUBECONFIG env var or in-cluster)
	config := ctrl.GetConfigOrDie()

	mgr, err := ctrl.NewManager(config, ctrl.Options{
		Scheme: scheme,
		Metrics: server.Options{
			BindAddress: metricsAddr,
		},
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "agentregistry.dev",
	})
	if err != nil {
		log.Error().Err(err).Msg("unable to create manager")
		os.Exit(1)
	}

	// Set up cache indexes for efficient queries
	if err := controller.SetupIndexes(mgr); err != nil {
		log.Error().Err(err).Msg("unable to setup indexes")
		os.Exit(1)
	}

	// Create controller logger
	ctrlLogger := log.Logger.With().Str("component", "controller").Logger()

	// Set up MCPServerCatalog reconciler
	if err := (&controller.MCPServerCatalogReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
		Logger: ctrlLogger.With().Str("controller", "mcpservercatalog").Logger(),
	}).SetupWithManager(mgr); err != nil {
		log.Error().Err(err).Str("controller", "MCPServerCatalog").Msg("unable to create controller")
		os.Exit(1)
	}

	// Set up AgentCatalog reconciler
	if err := (&controller.AgentCatalogReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
		Logger: ctrlLogger.With().Str("controller", "agentcatalog").Logger(),
	}).SetupWithManager(mgr); err != nil {
		log.Error().Err(err).Str("controller", "AgentCatalog").Msg("unable to create controller")
		os.Exit(1)
	}

	// Set up SkillCatalog reconciler
	if err := (&controller.SkillCatalogReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
		Logger: ctrlLogger.With().Str("controller", "skillcatalog").Logger(),
	}).SetupWithManager(mgr); err != nil {
		log.Error().Err(err).Str("controller", "SkillCatalog").Msg("unable to create controller")
		os.Exit(1)
	}

	// Set up RegistryDeployment reconciler
	if err := (&controller.RegistryDeploymentReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
		Logger: ctrlLogger.With().Str("controller", "registrydeployment").Logger(),
	}).SetupWithManager(mgr); err != nil {
		log.Error().Err(err).Str("controller", "RegistryDeployment").Msg("unable to create controller")
		os.Exit(1)
	}

	// Set up MCPServer discovery reconciler (auto-creates catalog from deployed MCPServers)
	if err := (&controller.MCPServerDiscoveryReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
		Logger: ctrlLogger.With().Str("controller", "mcpserver-discovery").Logger(),
	}).SetupWithManager(mgr); err != nil {
		log.Error().Err(err).Str("controller", "MCPServerDiscovery").Msg("unable to create controller")
		os.Exit(1)
	}

	// Set up Agent discovery reconciler (auto-creates catalog from deployed Agents)
	if err := (&controller.AgentDiscoveryReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
		Logger: ctrlLogger.With().Str("controller", "agent-discovery").Logger(),
	}).SetupWithManager(mgr); err != nil {
		log.Error().Err(err).Str("controller", "AgentDiscovery").Msg("unable to create controller")
		os.Exit(1)
	}

	// Set up Skill discovery reconciler (auto-creates catalog from Agent skill refs)
	if err := (&controller.SkillDiscoveryReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
		Logger: ctrlLogger.With().Str("controller", "skill-discovery").Logger(),
	}).SetupWithManager(mgr); err != nil {
		log.Error().Err(err).Str("controller", "SkillDiscovery").Msg("unable to create controller")
		os.Exit(1)
	}

	// Set up Model discovery reconciler (auto-creates catalog from ModelConfig)
	if err := (&controller.ModelDiscoveryReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
		Logger: ctrlLogger.With().Str("controller", "model-discovery").Logger(),
	}).SetupWithManager(mgr); err != nil {
		log.Error().Err(err).Str("controller", "ModelDiscovery").Msg("unable to create controller")
		os.Exit(1)
	}

	// Set up HTTP API server if enabled
	if enableHTTPAPI {
		apiLogger := log.Logger.With().Str("component", "httpapi").Logger()
		httpServer := httpapi.NewServer(
			mgr.GetClient(),
			mgr.GetCache(),
			apiLogger,
		)
		if err := mgr.Add(httpServer.Runnable(httpAPIAddr)); err != nil {
			log.Error().Err(err).Msg("unable to add HTTP API server")
			os.Exit(1)
		}
	}

	// Add health check endpoints
	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		log.Error().Err(err).Msg("unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		log.Error().Err(err).Msg("unable to set up ready check")
		os.Exit(1)
	}

	log.Info().Msg("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		log.Error().Err(err).Msg("problem running manager")
		os.Exit(1)
	}
}
