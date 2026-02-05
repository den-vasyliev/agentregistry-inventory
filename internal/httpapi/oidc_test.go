package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestContext(t *testing.T, headers map[string]string) huma.Context {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	return humatest.NewContext(nil, req, w)
}

func TestExtractBearerToken(t *testing.T) {
	tests := []struct {
		name      string
		headers   map[string]string
		wantToken string
		wantErr   string
	}{
		{
			name:      "valid bearer token",
			headers:   map[string]string{"Authorization": "Bearer abc123.def456.ghi789"},
			wantToken: "abc123.def456.ghi789",
		},
		{
			name:    "missing authorization header",
			headers: map[string]string{},
			wantErr: "missing authorization header",
		},
		{
			name:    "Basic scheme rejected",
			headers: map[string]string{"Authorization": "Basic dXNlcjpwYXNz"},
			wantErr: "invalid authorization header format",
		},
		{
			name:    "no space in header value",
			headers: map[string]string{"Authorization": "Bearertoken"},
			wantErr: "invalid authorization header format",
		},
		{
			name:      "token with spaces preserved after first split",
			headers:   map[string]string{"Authorization": "Bearer token with spaces"},
			wantToken: "token with spaces",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := newTestContext(t, tt.headers)
			token, err := extractBearerToken(ctx)
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantToken, token)
			}
		})
	}
}

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
