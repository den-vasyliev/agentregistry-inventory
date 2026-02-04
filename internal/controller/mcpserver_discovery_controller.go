package controller

import (
	"context"
	"fmt"
	"strings"

	"github.com/rs/zerolog"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	agentregistryv1alpha1 "github.com/agentregistry-dev/agentregistry/api/v1alpha1"
	kmcpv1alpha1 "github.com/kagent-dev/kmcp/api/v1alpha1"
)

// MCPServerDiscoveryReconciler watches MCPServer resources and creates catalog entries
type MCPServerDiscoveryReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Logger zerolog.Logger
}

const (
	discoveryLabel  = "agentregistry.dev/discovered"
	sourceKindLabel = "agentregistry.dev/source-kind"
	sourceNameLabel = "agentregistry.dev/source-name"
	sourceNSLabel   = "agentregistry.dev/source-namespace"
)

// +kubebuilder:rbac:groups=kagent.dev,resources=mcpservers,verbs=get;list;watch

// Reconcile watches MCPServer and creates/updates MCPServerCatalog
func (r *MCPServerDiscoveryReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := r.Logger.With().
		Str("mcpserver", req.Name).
		Str("namespace", req.Namespace).
		Logger()

	// Fetch the MCPServer
	var mcpServer kmcpv1alpha1.MCPServer
	if err := r.Get(ctx, req.NamespacedName, &mcpServer); err != nil {
		if apierrors.IsNotFound(err) {
			// MCPServer deleted - catalog entry will be orphaned but kept for history
			logger.Info().Msg("MCPServer deleted, catalog entry preserved")
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	logger.Debug().Msg("discovered MCPServer, syncing to catalog")

	// Generate catalog entry name
	catalogName := generateCatalogName(mcpServer.Namespace, mcpServer.Name)

	// Check if catalog entry exists
	var catalog agentregistryv1alpha1.MCPServerCatalog
	err := r.Get(ctx, types.NamespacedName{Name: catalogName}, &catalog)

	if apierrors.IsNotFound(err) {
		// Create new catalog entry
		catalog = r.buildCatalogFromMCPServer(&mcpServer, catalogName)
		if err := r.Create(ctx, &catalog); err != nil {
			logger.Error().Err(err).Msg("failed to create catalog entry")
			return ctrl.Result{}, err
		}
		logger.Info().Str("catalog", catalogName).Msg("created catalog entry from MCPServer")
	} else if err != nil {
		return ctrl.Result{}, err
	}

	// Update status from MCPServer
	r.syncStatusFromMCPServer(&catalog, &mcpServer)

	if err := r.Status().Update(ctx, &catalog); err != nil {
		if apierrors.IsConflict(err) {
			// Conflict means resource was modified, requeue to retry with latest version
			logger.Debug().Msg("conflict updating catalog status, will retry")
			return ctrl.Result{Requeue: true}, nil
		}
		logger.Error().Err(err).Msg("failed to update catalog status")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// buildCatalogFromMCPServer creates a MCPServerCatalog from MCPServer
func (r *MCPServerDiscoveryReconciler) buildCatalogFromMCPServer(mcpServer *kmcpv1alpha1.MCPServer, catalogName string) agentregistryv1alpha1.MCPServerCatalog {
	// Extract version from labels or use "latest"
	version := "latest"
	if v, ok := mcpServer.Labels["kmcp.dev/version"]; ok {
		version = v
	}
	if v, ok := mcpServer.Labels["app.kubernetes.io/version"]; ok {
		version = v
	}

	// Extract title/description from annotations
	title := mcpServer.Name
	description := ""
	if t, ok := mcpServer.Annotations["kmcp.dev/project-name"]; ok {
		title = t
	}
	if d, ok := mcpServer.Annotations["kmcp.dev/description"]; ok {
		description = d
	}

	// Build package info from deployment spec
	var packages []agentregistryv1alpha1.Package

	transportType := "stdio"
	if mcpServer.Spec.TransportType == "http" {
		transportType = "streamable-http"
	}

	// Build env vars list
	envVars := make([]agentregistryv1alpha1.KeyValueInput, 0)
	for name, value := range mcpServer.Spec.Deployment.Env {
		envVars = append(envVars, agentregistryv1alpha1.KeyValueInput{
			Name:  name,
			Value: value,
		})
	}

	pkg := agentregistryv1alpha1.Package{
		RegistryType: "oci",
		Identifier:   mcpServer.Spec.Deployment.Image,
		Transport: agentregistryv1alpha1.Transport{
			Type: transportType,
		},
		EnvironmentVariables: envVars,
	}
	packages = append(packages, pkg)

	// Determine environment from namespace
	environment := getEnvironmentFromNamespace(mcpServer.Namespace)

	// Generate universal resource UID: name-env-ver
	resourceUID := agentregistryv1alpha1.GenerateResourceUID(mcpServer.Name, environment, version)

	return agentregistryv1alpha1.MCPServerCatalog{
		ObjectMeta: metav1.ObjectMeta{
			Name: catalogName,
			Labels: map[string]string{
				discoveryLabel:                                 "true",
				sourceKindLabel:                                "MCPServer",
				sourceNameLabel:                                mcpServer.Name,
				sourceNSLabel:                                  mcpServer.Namespace,
				agentregistryv1alpha1.LabelResourceUID:         resourceUID,
				agentregistryv1alpha1.LabelResourceName:        mcpServer.Name,
				agentregistryv1alpha1.LabelResourceVersion:     version,
				agentregistryv1alpha1.LabelResourceEnvironment: environment,
				agentregistryv1alpha1.LabelResourceSource:      agentregistryv1alpha1.ResourceSourceDiscovery,
			},
		},
		Spec: agentregistryv1alpha1.MCPServerCatalogSpec{
			Name:        fmt.Sprintf("%s/%s", mcpServer.Namespace, mcpServer.Name),
			Version:     version,
			Title:       title,
			Description: description,
			SourceRef: &agentregistryv1alpha1.SourceReference{
				Kind:      "MCPServer",
				Name:      mcpServer.Name,
				Namespace: mcpServer.Namespace,
			},
			Packages: packages,
		},
	}
}

// syncStatusFromMCPServer updates catalog status from MCPServer status
func (r *MCPServerDiscoveryReconciler) syncStatusFromMCPServer(catalog *agentregistryv1alpha1.MCPServerCatalog, mcpServer *kmcpv1alpha1.MCPServer) {
	ready := false
	message := ""

	for _, cond := range mcpServer.Status.Conditions {
		if cond.Type == "Ready" {
			ready = cond.Status == "True"
			message = cond.Message
			break
		}
	}

	now := metav1.Now()
	catalog.Status.Deployment = &agentregistryv1alpha1.DeploymentRef{
		Namespace:   mcpServer.Namespace,
		ServiceName: mcpServer.Name,
		Ready:       ready,
		Message:     message,
		LastChecked: &now,
	}

	// Auto-publish discovered servers
	if !catalog.Status.Published {
		catalog.Status.Published = true
		catalog.Status.PublishedAt = &now
		catalog.Status.Status = agentregistryv1alpha1.CatalogStatusActive
	}
}

// generateCatalogName creates a valid K8s name from namespace and MCPServer name
func generateCatalogName(namespace, name string) string {
	combined := fmt.Sprintf("%s-%s", namespace, name)
	// Sanitize for K8s naming
	combined = strings.ReplaceAll(combined, "/", "-")
	combined = strings.ReplaceAll(combined, "_", "-")
	combined = strings.ToLower(combined)
	if len(combined) > 63 {
		combined = combined[:63]
	}
	// K8s names must end with alphanumeric, not hyphen
	combined = strings.TrimRight(combined, "-")
	return combined
}

// SetupWithManager sets up the controller with the Manager.
func (r *MCPServerDiscoveryReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&kmcpv1alpha1.MCPServer{}).
		Watches(
			&kmcpv1alpha1.MCPServer{},
			handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, obj client.Object) []reconcile.Request {
				return []reconcile.Request{{
					NamespacedName: types.NamespacedName{
						Name:      obj.GetName(),
						Namespace: obj.GetNamespace(),
					},
				}}
			}),
		).
		Complete(r)
}
