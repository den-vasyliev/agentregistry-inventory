package handlers

import (
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	agentregistryv1alpha1 "github.com/agentregistry-dev/agentregistry/api/v1alpha1"
	"github.com/agentregistry-dev/agentregistry/internal/validation"
)

// Response is a generic response wrapper
type Response[T any] struct {
	Body T
}

// PublisherInfoJSON holds governance trust data set by an external controller.
type PublisherInfoJSON struct {
	VerifiedPublisher    bool    `json:"verifiedPublisher,omitempty"`
	VerifiedOrganization bool    `json:"verifiedOrganization,omitempty"`
	Score                *int32  `json:"score,omitempty"`
	Grade                string  `json:"grade,omitempty"`
	GradedAt             *string `json:"gradedAt,omitempty"`
}

// DeploymentInfo contains runtime deployment information from the source resource
type DeploymentInfo struct {
	Namespace   string     `json:"namespace,omitempty"`
	ServiceName string     `json:"serviceName,omitempty"`
	URL         string     `json:"url,omitempty"`
	Ready       bool       `json:"ready"`
	Message     string     `json:"message,omitempty"`
	LastChecked *time.Time `json:"lastChecked,omitempty"`
}

// EmptyResponse represents an empty response
type EmptyResponse struct {
	Message string `json:"message,omitempty"`
}

// ListMetadata contains pagination metadata
type ListMetadata struct {
	NextCursor string `json:"nextCursor,omitempty"`
	Count      int    `json:"count"`
}

// SanitizeK8sName converts a name to a valid Kubernetes resource name
func SanitizeK8sName(name string) string {
	return validation.SanitizeName(name)
}

// GenerateCRName generates a CR name from name and version
func GenerateCRName(name, version string) string {
	sanitizedName := SanitizeK8sName(name)
	sanitizedVersion := SanitizeK8sName(version)
	return sanitizedName + "-" + sanitizedVersion
}

// convertPublisherVerification converts a CRD PublisherVerification to PublisherInfoJSON.
// Returns nil if v is nil.
func convertPublisherVerification(v *agentregistryv1alpha1.PublisherVerification) *PublisherInfoJSON {
	if v == nil {
		return nil
	}
	info := &PublisherInfoJSON{
		VerifiedPublisher:    v.VerifiedPublisher,
		VerifiedOrganization: v.VerifiedOrganization,
		Score:                v.Score,
		Grade:                string(v.Grade),
	}
	if v.GradedAt != nil {
		t := v.GradedAt.UTC().Format(time.RFC3339)
		info.GradedAt = &t
	}
	return info
}

// SetCatalogCondition sets or updates a condition in the status
func SetCatalogCondition(conditions []agentregistryv1alpha1.CatalogCondition, condType agentregistryv1alpha1.CatalogConditionType, status metav1.ConditionStatus, reason, message string) []agentregistryv1alpha1.CatalogCondition {
	now := metav1.Now()
	for i, c := range conditions {
		if c.Type == condType {
			if c.Status != status {
				conditions[i].LastTransitionTime = now
			}
			conditions[i].Status = status
			conditions[i].Reason = reason
			conditions[i].Message = message
			return conditions
		}
	}
	return append(conditions, agentregistryv1alpha1.CatalogCondition{
		Type:               condType,
		Status:             status,
		LastTransitionTime: now,
		Reason:             reason,
		Message:            message,
	})
}
