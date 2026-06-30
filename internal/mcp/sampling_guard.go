package mcp

import (
	"fmt"
	"os"
	"strconv"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// samplingMaxTokens is the per-call MaxTokens cap sent to the client model.
const samplingMaxTokens = 4096

// samplingGuard throttles the LLM sampling path to bound cost and prevent abuse.
//
// It enforces two independent limits:
//   - a token-bucket rate limit on the number of sampling calls (calls/sec with
//     a small burst), and
//   - an aggregate token budget: a running ceiling on the total MaxTokens
//     requested across all sampling calls, refilled over time.
//
// Limits are server-wide (not per-caller) because mcp-go does not plumb a
// stable caller identity into tool handlers; a server-wide budget is the
// conservative choice and still caps total exposure of the connected model.
// Both are tunable via env vars.
type samplingGuard struct {
	limiter *rate.Limiter

	mu          sync.Mutex
	tokenBudget *rate.Limiter // tokens treated as "events" in a token bucket
}

// newSamplingGuard builds a guard from env configuration:
//
//	AGENTREGISTRY_SAMPLING_RPS         calls per second   (default 1)
//	AGENTREGISTRY_SAMPLING_BURST       call burst          (default 5)
//	AGENTREGISTRY_SAMPLING_TOKENS_PER_SEC  token refill/sec (default 4096)
//	AGENTREGISTRY_SAMPLING_TOKENS_BURST    max token burst  (default 65536)
func newSamplingGuard() *samplingGuard {
	rps := envFloat("AGENTREGISTRY_SAMPLING_RPS", 1)
	burst := envInt("AGENTREGISTRY_SAMPLING_BURST", 5)
	tps := envFloat("AGENTREGISTRY_SAMPLING_TOKENS_PER_SEC", 4096)
	tburst := envInt("AGENTREGISTRY_SAMPLING_TOKENS_BURST", 65536)

	return &samplingGuard{
		limiter:     rate.NewLimiter(rate.Limit(rps), burst),
		tokenBudget: rate.NewLimiter(rate.Limit(tps), tburst),
	}
}

// allow reserves capacity for one sampling call requesting up to maxTokens.
// It returns an error (suitable for surfacing to the caller) when either the
// call-rate limit or the token budget is exhausted. Nothing is consumed unless
// both checks pass.
func (g *samplingGuard) allow(maxTokens int) error {
	if g == nil {
		return nil
	}
	if maxTokens <= 0 {
		maxTokens = samplingMaxTokens
	}

	g.mu.Lock()
	defer g.mu.Unlock()

	now := time.Now()

	// Reserve both limits before committing to either, so a failure in one does
	// not consume capacity from the other. Cancel(now) immediately returns a
	// reservation's capacity to the bucket.
	callRes := g.limiter.ReserveN(now, 1)
	tokenRes := g.tokenBudget.ReserveN(now, maxTokens)

	callOK := callRes.OK() && callRes.DelayFrom(now) == 0
	tokenOK := tokenRes.OK() && tokenRes.DelayFrom(now) == 0

	if !callOK || !tokenOK {
		callRes.CancelAt(now)
		tokenRes.CancelAt(now)
		if !tokenOK {
			return fmt.Errorf("sampling token budget exhausted; try again shortly")
		}
		return fmt.Errorf("sampling rate limit exceeded; try again shortly")
	}

	return nil
}

func envFloat(key string, def float64) float64 {
	if v := os.Getenv(key); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil && f > 0 {
			return f
		}
	}
	return def
}

func envInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return def
}
