package masteragent

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildSystemPrompt(t *testing.T) {
	tests := []struct {
		name        string
		override    string
		wantDefault bool
	}{
		{"empty override returns default", "", true},
		{"custom override returned", "Custom prompt", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildSystemPrompt(tt.override)
			if tt.wantDefault {
				assert.Equal(t, defaultSystemPrompt, got)
			} else {
				assert.Equal(t, tt.override, got)
			}
		})
	}
}
