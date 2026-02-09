package controller

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/rs/zerolog"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	agentregistryv1alpha1 "github.com/agentregistry-dev/agentregistry/api/v1alpha1"
	"github.com/agentregistry-dev/agentregistry/internal/config"
	"github.com/agentregistry-dev/agentregistry/internal/masteragent"

	"google.golang.org/adk/agent"
	"google.golang.org/adk/cmd/launcher"
	"google.golang.org/adk/cmd/launcher/web"
	weba2a "google.golang.org/adk/cmd/launcher/web/a2a"
)

// MasterAgentReconciler reconciles MasterAgentConfig objects
type MasterAgentReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Logger zerolog.Logger

	mu           sync.Mutex
	agent        *masteragent.MasterAgent
	hub          *masteragent.EventHub
	cancelFn     context.CancelFunc
	a2aCancel    context.CancelFunc
	currentModel string // tracks current model name to detect changes
}

// GetHub returns the event hub (used by HTTP handlers to push events)
func (r *MasterAgentReconciler) GetHub() *masteragent.EventHub {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.hub
}

// GetAgent returns the master agent (used by HTTP handlers for status)
func (r *MasterAgentReconciler) GetAgent() *masteragent.MasterAgent {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.agent
}

// +kubebuilder:rbac:groups=agentregistry.dev,resources=masteragentconfigs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=agentregistry.dev,resources=masteragentconfigs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=agentregistry.dev,resources=modelcatalogs,verbs=get;list;watch

// Reconcile handles MasterAgentConfig changes
func (r *MasterAgentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := r.Logger.With().Str("masteragentconfig", req.Name).Logger()

	// Fetch MasterAgentConfig
	var mac agentregistryv1alpha1.MasterAgentConfig
	if err := r.Get(ctx, req.NamespacedName, &mac); err != nil {
		if apierrors.IsNotFound(err) {
			logger.Info().Msg("MasterAgentConfig deleted, stopping agent")
			r.stopAgent()
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// If disabled, stop agent
	if !mac.Spec.Enabled {
		logger.Info().Msg("MasterAgentConfig disabled, stopping agent")
		r.stopAgent()
		r.updateStatus(ctx, &mac, false)
		return ctrl.Result{}, nil
	}

	logger.Trace().Msg("reconciling MasterAgentConfig")

	// Resolve model from ModelCatalog
	defaultModel, err := r.resolveModel(ctx, mac.Spec.Models.Default)
	if err != nil {
		logger.Error().Err(err).Str("model", mac.Spec.Models.Default).Msg("failed to resolve default model")
		r.setCondition(ctx, &mac, "ModelResolved", metav1.ConditionFalse, "ModelNotFound", err.Error())
		return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
	}

	r.setCondition(ctx, &mac, "ModelResolved", metav1.ConditionTrue, "ModelResolved",
		fmt.Sprintf("Using model %s at %s", defaultModel.Name(), mac.Spec.Models.Default))

	// Check if agent needs restart (new agent or model changed)
	r.mu.Lock()
	needsRestart := r.agent == nil || r.currentModel != defaultModel.Name()
	if needsRestart && r.agent != nil {
		logger.Info().
			Str("old_model", r.currentModel).
			Str("new_model", defaultModel.Name()).
			Msg("model changed, restarting agent")
	}
	r.mu.Unlock()

	if needsRestart {
		// Stop old agent if running
		r.stopAgent()

		if err := r.startAgent(ctx, &mac, defaultModel); err != nil {
			logger.Error().Err(err).Msg("failed to start agent")
			r.setCondition(ctx, &mac, "AgentReady", metav1.ConditionFalse, "StartFailed", err.Error())
			return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
		}

		// Track current model
		r.mu.Lock()
		r.currentModel = defaultModel.Name()
		r.mu.Unlock()

		r.setCondition(ctx, &mac, "AgentReady", metav1.ConditionTrue, "AgentRunning", "Agent is running")
	}

	// Sync world state to CRD status
	r.updateStatus(ctx, &mac, true)

	// Requeue to periodically flush status
	return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
}

// resolveModel looks up a ModelCatalog by spec.name and returns a GatewayModel
func (r *MasterAgentReconciler) resolveModel(ctx context.Context, modelName string) (*masteragent.GatewayModel, error) {
	namespace := config.GetNamespace()

	var modelList agentregistryv1alpha1.ModelCatalogList
	if err := r.List(ctx, &modelList, client.InNamespace(namespace)); err != nil {
		return nil, fmt.Errorf("list model catalogs: %w", err)
	}

	for _, mc := range modelList.Items {
		if mc.Spec.Name == modelName {
			if mc.Spec.BaseURL == "" {
				return nil, fmt.Errorf("model %q has no baseUrl configured", modelName)
			}
			return masteragent.NewGatewayModel(mc.Spec.Model, mc.Spec.BaseURL, os.Getenv("LLM_API_KEY")), nil
		}
	}

	return nil, fmt.Errorf("model %q not found in ModelCatalog", modelName)
}

// startAgent initializes the ADK agent, event hub, and optionally the A2A server
func (r *MasterAgentReconciler) startAgent(ctx context.Context, mac *agentregistryv1alpha1.MasterAgentConfig, model *masteragent.GatewayModel) error {
	hub := masteragent.NewEventHub(1000, 100)

	agentCtx, cancel := context.WithCancel(ctx)

	ag, err := masteragent.NewMasterAgent(agentCtx, mac.Spec, model, hub, r.Logger)
	if err != nil {
		cancel()
		return fmt.Errorf("create master agent: %w", err)
	}

	r.mu.Lock()
	r.agent = ag
	r.hub = hub
	r.cancelFn = cancel
	r.mu.Unlock()

	// Start event processing workers
	ag.Run(agentCtx)

	// Start A2A server if enabled
	if mac.Spec.A2A.Enabled {
		port := mac.Spec.A2A.Port
		if port <= 0 {
			port = 8084
		}
		r.startA2AServer(agentCtx, ag, port)
	}

	r.Logger.Info().Msg("master agent started")
	return nil
}

// startA2AServer starts the A2A protocol server in a goroutine
func (r *MasterAgentReconciler) startA2AServer(ctx context.Context, ag *masteragent.MasterAgent, port int) {
	a2aCtx, cancel := context.WithCancel(ctx)
	r.mu.Lock()
	r.a2aCancel = cancel
	r.mu.Unlock()

	go func() {
		webLauncher := web.NewLauncher(weba2a.NewLauncher())
		// Parse port flag before running
		a2aURL := fmt.Sprintf("http://localhost:%d", port)
		if _, err := webLauncher.Parse([]string{"--port", fmt.Sprintf("%d", port), "a2a", fmt.Sprintf("--a2a_agent_url=%s", a2aURL)}); err != nil {
			r.Logger.Error().Err(err).Msg("failed to parse A2A launcher args")
			return
		}
		err := webLauncher.Run(a2aCtx, &launcher.Config{
			AgentLoader:    agent.NewSingleLoader(ag.Agent()),
			SessionService: ag.SessionService(),
		})
		if err != nil && a2aCtx.Err() == nil {
			r.Logger.Error().Err(err).Msg("A2A server error")
		}
	}()

	r.Logger.Info().Int("port", port).Msg("A2A server starting")
}

// stopAgent shuts down the running agent and A2A server
func (r *MasterAgentReconciler) stopAgent() {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.a2aCancel != nil {
		r.a2aCancel()
		r.a2aCancel = nil
	}
	if r.cancelFn != nil {
		r.cancelFn()
		r.cancelFn = nil
	}
	r.agent = nil
	r.hub = nil
}

// updateStatus syncs the world state and agent status to the CRD
func (r *MasterAgentReconciler) updateStatus(ctx context.Context, mac *agentregistryv1alpha1.MasterAgentConfig, running bool) {
	r.mu.Lock()
	ag := r.agent
	hub := r.hub
	r.mu.Unlock()

	if ag != nil && hub != nil {
		mac.Status.WorldState = ag.State().ToStatus(hub.QueueDepth())
		mac.Status.Incidents = ag.State().GetIncidents()
		mac.Status.QueueDepth = hub.QueueDepth()
		mac.Status.LLMAvailable = true
	} else {
		mac.Status.LLMAvailable = false
	}

	if mac.Spec.A2A.Enabled {
		port := mac.Spec.A2A.Port
		if port <= 0 {
			port = 8084
		}
		mac.Status.A2AEndpoint = fmt.Sprintf("http://localhost:%d", port)
	}

	if err := r.Status().Update(ctx, mac); err != nil {
		if !apierrors.IsConflict(err) {
			r.Logger.Error().Err(err).Msg("failed to update status")
		}
	}
}

// setCondition sets or updates a condition on the MasterAgentConfig
func (r *MasterAgentReconciler) setCondition(ctx context.Context, mac *agentregistryv1alpha1.MasterAgentConfig, condType string, status metav1.ConditionStatus, reason, message string) {
	now := metav1.Now()
	found := false
	for i, c := range mac.Status.Conditions {
		if c.Type == condType {
			if c.Status != status {
				mac.Status.Conditions[i].LastTransitionTime = now
			}
			mac.Status.Conditions[i].Status = status
			mac.Status.Conditions[i].Reason = reason
			mac.Status.Conditions[i].Message = message
			mac.Status.Conditions[i].ObservedGeneration = mac.Generation
			found = true
			break
		}
	}
	if !found {
		mac.Status.Conditions = append(mac.Status.Conditions, metav1.Condition{
			Type:               condType,
			Status:             status,
			ObservedGeneration: mac.Generation,
			LastTransitionTime: now,
			Reason:             reason,
			Message:            message,
		})
	}

	if err := r.Status().Update(ctx, mac); err != nil {
		if !apierrors.IsConflict(err) {
			r.Logger.Error().Err(err).Msg("failed to update condition")
		}
	}
}

// SetupWithManager sets up the controller with the manager
func (r *MasterAgentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&agentregistryv1alpha1.MasterAgentConfig{}).
		Complete(r)
}
