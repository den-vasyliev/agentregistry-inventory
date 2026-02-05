package httpapi

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/danielgtaylor/huma/v2"
	"github.com/rs/zerolog"

	"github.com/agentregistry-dev/agentregistry/internal/config"
)

// OIDCVerifier validates JWTs and checks required group membership.
type OIDCVerifier struct {
	issuer     string
	audience   string
	groupClaim string
	adminGroup string
	verifier   *oidc.IDTokenVerifier
	tokenCache *TokenCache
	api        huma.API
	logger     zerolog.Logger
}

// NewOIDCVerifier creates a verifier using the configured issuer/audience.
func NewOIDCVerifier(ctx context.Context, api huma.API, logger zerolog.Logger) (*OIDCVerifier, error) {
	issuer := strings.TrimSpace(config.GetOIDCIssuer())
	audience := strings.TrimSpace(config.GetOIDCAudience())
	if issuer == "" || audience == "" {
		return nil, fmt.Errorf("missing OIDC issuer or audience")
	}

	provider, err := oidc.NewProvider(ctx, issuer)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize OIDC provider: %w", err)
	}

	verifier := provider.Verifier(&oidc.Config{ClientID: audience})

	return &OIDCVerifier{
		issuer:     issuer,
		audience:   audience,
		groupClaim: config.GetOIDCGroupClaim(),
		adminGroup: strings.TrimSpace(config.GetOIDCAdminGroup()),
		verifier:   verifier,
		tokenCache: NewTokenCache(),
		api:        api,
		logger:     logger.With().Str("component", "oidc").Logger(),
	}, nil
}

// RequireAdminGroup enforces the admin group (if configured).
func (v *OIDCVerifier) RequireAdminGroup(ctx huma.Context) bool {
	if v.adminGroup == "" {
		// No group configured: allow authenticated users.
		return true
	}

	claims, err := v.verifyAndClaims(ctx)
	if err != nil {
		v.writeUnauthorized(ctx, err.Error())
		return false
	}

	groups := readStringArrayClaim(claims, v.groupClaim)
	for _, g := range groups {
		if g == v.adminGroup {
			return true
		}
	}

	v.writeForbidden(ctx, "Missing required admin group")
	return false
}

func (v *OIDCVerifier) verifyAndClaims(ctx huma.Context) (map[string]interface{}, error) {
	token, err := extractBearerToken(ctx)
	if err != nil {
		return nil, err
	}

	// Check cache first
	tokenHash := HashToken(token)
	if claims, ok := v.tokenCache.Get(tokenHash); ok {
		v.logger.Debug().Msg("token cache hit")
		return claims, nil
	}

	// Cache miss - validate with OIDC provider
	v.logger.Debug().Msg("token cache miss - validating with OIDC provider")
	idToken, err := v.verifier.Verify(ctx.Context(), token)
	if err != nil {
		// Log specific error types for debugging
		if strings.Contains(err.Error(), "expired") {
			v.logger.Warn().Msg("token expired")
			return nil, fmt.Errorf("token expired")
		}
		if strings.Contains(err.Error(), "signature") {
			v.logger.Error().Err(err).Msg("invalid token signature")
			return nil, fmt.Errorf("invalid token signature")
		}
		v.logger.Error().Err(err).Msg("token verification failed")
		return nil, fmt.Errorf("invalid token")
	}

	claims := map[string]interface{}{}
	if err := idToken.Claims(&claims); err != nil {
		return nil, fmt.Errorf("invalid token claims")
	}

	// Store in cache with TTL based on token expiry
	if exp, ok := claims["exp"].(float64); ok {
		tokenExp := time.Unix(int64(exp), 0)
		safetyMargin := config.GetOIDCCacheSafetyMargin()
		v.tokenCache.Set(tokenHash, claims, tokenExp, safetyMargin)
		v.logger.Debug().
			Time("tokenExp", tokenExp).
			Dur("safetyMargin", safetyMargin).
			Msg("token cached")
	}

	return claims, nil
}

func extractBearerToken(ctx huma.Context) (string, error) {
	authHeader := ctx.Header("Authorization")
	if authHeader == "" {
		return "", fmt.Errorf("missing authorization header")
	}
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || parts[0] != "Bearer" {
		return "", fmt.Errorf("invalid authorization header format")
	}
	return parts[1], nil
}

func readStringArrayClaim(claims map[string]interface{}, name string) []string {
	raw, ok := claims[name]
	if !ok || raw == nil {
		return nil
	}

	switch v := raw.(type) {
	case []interface{}:
		out := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				out = append(out, s)
			}
		}
		return out
	case []string:
		return v
	case string:
		// Some IdPs return a single string when only one group exists.
		return []string{v}
	default:
		return nil
	}
}

func (v *OIDCVerifier) writeUnauthorized(ctx huma.Context, message string) {
	ctx.SetStatus(401)
	huma.WriteErr(v.api, ctx, 401, message)
}

func (v *OIDCVerifier) writeForbidden(ctx huma.Context, message string) {
	ctx.SetStatus(403)
	huma.WriteErr(v.api, ctx, 403, message)
}
