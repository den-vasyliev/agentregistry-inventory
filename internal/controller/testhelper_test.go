package controller

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/metrics/server"

	agentregistryv1alpha1 "github.com/agentregistry-dev/agentregistry/api/v1alpha1"
	kagentv1alpha2 "github.com/kagent-dev/kagent/go/api/v1alpha2"
	kmcpv1alpha1 "github.com/kagent-dev/kmcp/api/v1alpha1"
)

// TestEnvHelper holds the test environment state
type TestEnvHelper struct {
	Env       *envtest.Environment
	Config    *rest.Config
	Client    client.Client
	Scheme    *runtime.Scheme
	Manager   manager.Manager
	Ctx       context.Context
	Cancel    context.CancelFunc
	MgrCancel context.CancelFunc
}

// SetupTestEnv creates a new test environment for controller testing
func SetupTestEnv(t *testing.T, timeout time.Duration, startMgr bool) *TestEnvHelper {
	helper := &TestEnvHelper{}

	// Create a new scheme that includes both core Kubernetes types and our CRDs
	helper.Scheme = runtime.NewScheme()

	// Add the core Kubernetes schemes
	err := scheme.AddToScheme(helper.Scheme)
	require.NoError(t, err)

	// Add CRD scheme
	err = apiextensionsv1.AddToScheme(helper.Scheme)
	require.NoError(t, err)

	// Add AgentRegistry CRDs
	err = agentregistryv1alpha1.AddToScheme(helper.Scheme)
	require.NoError(t, err)

	// Add KAgent schemes (for deployment reconciler)
	err = kagentv1alpha2.AddToScheme(helper.Scheme)
	require.NoError(t, err)

	// Add KMCP schemes
	err = kmcpv1alpha1.AddToScheme(helper.Scheme)
	require.NoError(t, err)

	// Setup envtest environment
	helper.Env = &envtest.Environment{
		CRDDirectoryPaths: []string{
			"../../charts/agentregistry/crds/",
		},
		ErrorIfCRDPathMissing:    true,
		AttachControlPlaneOutput: false,
	}

	// Create a longer context timeout for environment startup
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var startErr = make(chan error)
	go func() {
		var err error
		helper.Config, err = helper.Env.Start()
		startErr <- err
	}()

	// Wait for environment to start with timeout
	select {
	case err := <-startErr:
		require.NoError(t, err, "Failed to start test environment")
	case <-ctx.Done():
		t.Fatal("Timeout waiting for test environment to start")
	}

	// Create the client with the combined scheme
	helper.Client, err = client.New(helper.Config, client.Options{Scheme: helper.Scheme})
	require.NoError(t, err, "Failed to create test client")

	// Initialize manager
	helper.Manager, err = ctrl.NewManager(helper.Config, ctrl.Options{
		Scheme:                  helper.Scheme,
		LeaderElection:          false,
		LeaderElectionNamespace: "default",
		HealthProbeBindAddress:  "0",
		Metrics:                 server.Options{BindAddress: "0"},
	})
	require.NoError(t, err, "Failed to create manager")

	// Setup indexes
	err = SetupIndexes(helper.Manager)
	require.NoError(t, err, "Failed to setup indexes")

	// Create test context
	helper.Ctx, helper.Cancel = context.WithTimeout(context.Background(), timeout)

	// Start manager in a goroutine if requested
	if startMgr {
		var mgrCtx context.Context
		mgrCtx, helper.MgrCancel = context.WithCancel(context.Background())
		go func() {
			if err := helper.Manager.Start(mgrCtx); err != nil {
				t.Logf("Manager stopped with error: %v", err)
			}
		}()

		// Wait for cache to sync
		syncCtx, syncCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer syncCancel()

		if !helper.Manager.GetCache().WaitForCacheSync(syncCtx) {
			t.Fatal("Timeout waiting for cache to sync")
		}
	}

	return helper
}

// Cleanup tears down the test environment
func (h *TestEnvHelper) Cleanup(t *testing.T) {
	if h.MgrCancel != nil {
		h.MgrCancel()
	}
	if h.Cancel != nil {
		h.Cancel()
	}
	if h.Env != nil {
		err := h.Env.Stop()
		require.NoError(t, err, "Failed to stop test environment")
	}
}

// SetupReconcilers sets up all catalog reconcilers for testing
func (h *TestEnvHelper) SetupReconcilers(t *testing.T, logger interface{}) {
	// Note: In actual tests, you would pass a proper slog.Logger
	// For now, we'll skip this since the logger setup needs discussion
}
