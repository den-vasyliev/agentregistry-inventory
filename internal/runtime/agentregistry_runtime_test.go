package runtime

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	v1alpha2 "github.com/kagent-dev/kagent/go/api/v1alpha2"
	kmcpv1alpha1 "github.com/kagent-dev/kmcp/api/v1alpha1"
)

func setupRuntimeTestClient(t *testing.T) client.Client {
	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))
	require.NoError(t, kmcpv1alpha1.AddToScheme(scheme))
	require.NoError(t, v1alpha2.AddToScheme(scheme))
	return fake.NewClientBuilder().WithScheme(scheme).Build()
}

// ---------------------------------------------------------------------------
// applyResource tests
// ---------------------------------------------------------------------------

func TestApplyResource_ConfigMap(t *testing.T) {
	c := setupRuntimeTestClient(t)
	ctx := context.Background()

	cm := &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-config",
			Namespace: "default",
		},
		Data: map[string]string{
			"key": "value",
		},
	}

	err := applyResource(ctx, c, cm, false)
	require.NoError(t, err)

	// Verify the ConfigMap was created
	var retrieved corev1.ConfigMap
	err = c.Get(ctx, client.ObjectKey{Name: "test-config", Namespace: "default"}, &retrieved)
	require.NoError(t, err)
	assert.Equal(t, "value", retrieved.Data["key"])
}

func TestApplyResource_Update(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping test that requires proper server-side apply support")
	}

	c := setupRuntimeTestClient(t)
	ctx := context.Background()

	// Create initial ConfigMap
	cm := &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "update-config",
			Namespace: "default",
		},
		Data: map[string]string{
			"key": "original",
		},
	}

	err := applyResource(ctx, c, cm, false)
	require.NoError(t, err)

	// Note: Server-side apply updates require envtest
	// Fake client doesn't fully support SSA semantics
}

func TestApplyResource_Verbose(t *testing.T) {
	c := setupRuntimeTestClient(t)
	ctx := context.Background()

	cm := &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "verbose-config",
			Namespace: "default",
		},
		Data: map[string]string{
			"key": "value",
		},
	}

	// Test with verbose=true (should not error, just print)
	err := applyResource(ctx, c, cm, true)
	require.NoError(t, err)
}

// ---------------------------------------------------------------------------
// deleteResource tests
// ---------------------------------------------------------------------------

func TestDeleteResource_Exists(t *testing.T) {
	c := setupRuntimeTestClient(t)
	ctx := context.Background()

	// Create a ConfigMap first
	cm := &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "delete-me",
			Namespace: "default",
		},
		Data: map[string]string{
			"key": "value",
		},
	}

	err := c.Create(ctx, cm)
	require.NoError(t, err)

	// Delete it
	err = deleteResource(ctx, c, cm)
	require.NoError(t, err)

	// Verify deletion
	var retrieved corev1.ConfigMap
	err = c.Get(ctx, client.ObjectKey{Name: "delete-me", Namespace: "default"}, &retrieved)
	assert.Error(t, err)
	assert.True(t, client.IgnoreNotFound(err) == nil, "Should be NotFound error")
}

func TestDeleteResource_NotFound(t *testing.T) {
	c := setupRuntimeTestClient(t)
	ctx := context.Background()

	// Try to delete non-existent ConfigMap (should not error)
	cm := &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "does-not-exist",
			Namespace: "default",
		},
	}

	err := deleteResource(ctx, c, cm)
	require.NoError(t, err, "Deleting non-existent resource should not error")
}

// ---------------------------------------------------------------------------
// Helper function tests
// ---------------------------------------------------------------------------

func TestFieldManager(t *testing.T) {
	// Verify the field manager constant is set correctly
	assert.Equal(t, "agentregistry", fieldManager)
}

func TestSchemeContainsRequiredTypes(t *testing.T) {
	// Verify scheme includes required types
	gvk := corev1.SchemeGroupVersion.WithKind("ConfigMap")
	assert.True(t, scheme.Recognizes(gvk), "Scheme should recognize ConfigMap")

	gvk = kmcpv1alpha1.GroupVersion.WithKind("MCPServer")
	assert.True(t, scheme.Recognizes(gvk), "Scheme should recognize MCPServer")

	gvk = v1alpha2.GroupVersion.WithKind("Agent")
	assert.True(t, scheme.Recognizes(gvk), "Scheme should recognize Agent")
}
