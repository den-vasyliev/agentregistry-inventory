package httpapi

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReadStringArrayClaim(t *testing.T) {
	tests := []struct {
		name      string
		claims    map[string]interface{}
		claimName string
		expected  []string
	}{
		{
			name: "slice of interfaces",
			claims: map[string]interface{}{
				"groups": []interface{}{"admin", "user", "editor"},
			},
			claimName: "groups",
			expected:  []string{"admin", "user", "editor"},
		},
		{
			name: "slice of strings",
			claims: map[string]interface{}{
				"groups": []string{"admin", "user"},
			},
			claimName: "groups",
			expected:  []string{"admin", "user"},
		},
		{
			name: "single string",
			claims: map[string]interface{}{
				"groups": "admin",
			},
			claimName: "groups",
			expected:  []string{"admin"},
		},
		{
			name: "missing claim",
			claims: map[string]interface{}{
				"other": "value",
			},
			claimName: "groups",
			expected:  nil,
		},
		{
			name: "nil claim",
			claims: map[string]interface{}{
				"groups": nil,
			},
			claimName: "groups",
			expected:  nil,
		},
		{
			name: "wrong type",
			claims: map[string]interface{}{
				"groups": 123,
			},
			claimName: "groups",
			expected:  nil,
		},
		{
			name: "mixed types in slice",
			claims: map[string]interface{}{
				"groups": []interface{}{"admin", 123, "user", nil},
			},
			claimName: "groups",
			expected:  []string{"admin", "user"},
		},
		{
			name:      "empty claims map",
			claims:    map[string]interface{}{},
			claimName: "groups",
			expected:  nil,
		},
		{
			name: "nested claim name not found",
			claims: map[string]interface{}{
				"roles": []string{"admin"},
			},
			claimName: "groups",
			expected:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := readStringArrayClaim(tt.claims, tt.claimName)
			assert.Equal(t, tt.expected, result)
		})
	}
}
