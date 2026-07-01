package controller

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/yaml"

	agentregistryv1alpha1 "github.com/agentregistry-dev/agentregistry/api/v1alpha1"
)

func TestTweetClawSkillCatalogSample(t *testing.T) {
	path := filepath.Join("..", "..", "config", "samples", "skillcatalog_tweetclaw.yaml")
	data, err := os.ReadFile(path)
	require.NoError(t, err)

	var sample agentregistryv1alpha1.SkillCatalog
	require.NoError(t, yaml.Unmarshal(data, &sample))

	assert.Equal(t, "SkillCatalog", sample.Kind)
	assert.Equal(t, "tweetclaw-openclaw", sample.Spec.Name)
	assert.Equal(t, "1.6.31", sample.Spec.Version)
	assert.Equal(t, "social-automation", sample.Spec.Category)
	require.NotNil(t, sample.Spec.Repository)
	assert.Equal(t, "https://github.com/Xquik-dev/tweetclaw", sample.Spec.Repository.URL)
	assert.Equal(t, "github", sample.Spec.Repository.Source)
	require.Len(t, sample.Spec.Packages, 1)
	assert.Equal(t, "npm", sample.Spec.Packages[0].RegistryType)
	assert.Equal(t, "@xquik/tweetclaw", sample.Spec.Packages[0].Identifier)
	assert.Equal(t, "1.6.31", sample.Spec.Packages[0].Version)
	require.NotNil(t, sample.Spec.Metadata)

	rawMetadata := string(sample.Spec.Metadata.Raw)
	assert.Contains(t, rawMetadata, "openclaw")
	assert.Contains(t, rawMetadata, "tweetclaw")
	assert.Contains(t, rawMetadata, "requires-review")
	assert.NotContains(t, strings.ToLower(string(data)), "api_key")
}
