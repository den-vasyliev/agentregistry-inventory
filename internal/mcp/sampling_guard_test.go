package mcp

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/time/rate"
)

func TestSamplingGuard_RateLimit(t *testing.T) {
	g := &samplingGuard{
		// 0 refill, burst of 2 calls — exactly 2 calls allowed, then blocked.
		limiter:     rate.NewLimiter(0, 2),
		tokenBudget: rate.NewLimiter(0, 1_000_000), // token budget not the constraint here
	}

	assert.NoError(t, g.allow(100), "1st call within burst should pass")
	assert.NoError(t, g.allow(100), "2nd call within burst should pass")
	assert.Error(t, g.allow(100), "3rd call should be rate-limited")
}

func TestSamplingGuard_TokenBudget(t *testing.T) {
	g := &samplingGuard{
		limiter:     rate.NewLimiter(rate.Inf, 1000), // rate not the constraint
		tokenBudget: rate.NewLimiter(0, 5000),        // 5000-token budget, no refill
	}

	assert.NoError(t, g.allow(4096), "first call fits in the budget")
	assert.Error(t, g.allow(4096), "second call exceeds the remaining token budget")
}

func TestSamplingGuard_NilSafe(t *testing.T) {
	var g *samplingGuard
	assert.NoError(t, g.allow(100), "nil guard must be a no-op")
}

func TestSamplingGuard_RateFailureDoesNotConsumeBudget(t *testing.T) {
	g := &samplingGuard{
		limiter:     rate.NewLimiter(0, 0),   // no calls allowed at all
		tokenBudget: rate.NewLimiter(0, 100), // small budget
	}
	// Rate check fails; the reserved token budget must be returned so it isn't leaked.
	assert.Error(t, g.allow(50))
	tokensBefore := g.tokenBudget.Tokens()
	assert.InDelta(t, 100.0, tokensBefore, 1.0, "token budget should be intact after a rate-limited call")
}
