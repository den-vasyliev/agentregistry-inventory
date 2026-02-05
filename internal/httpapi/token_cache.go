package httpapi

import (
	"crypto/sha256"
	"encoding/hex"
	"sync"
	"time"
)

const (
	// maxCacheSize is the maximum number of tokens to cache before LRU eviction
	maxCacheSize = 1000
	// cleanupInterval is how often to run the cleanup goroutine
	cleanupInterval = 1 * time.Minute
)

// TokenCache provides thread-safe caching of validated OIDC tokens
type TokenCache struct {
	mu      sync.RWMutex
	cache   map[string]*cachedToken
	stopCh  chan struct{}
	stopped bool
}

// cachedToken represents a cached token with its claims and expiry
type cachedToken struct {
	claims     map[string]interface{}
	validUntil time.Time // Cache entry expires before token expires (safety margin)
	lastAccess time.Time // For LRU eviction
}

// NewTokenCache creates a new token cache and starts the cleanup goroutine
func NewTokenCache() *TokenCache {
	tc := &TokenCache{
		cache:  make(map[string]*cachedToken),
		stopCh: make(chan struct{}),
	}
	go tc.cleanupLoop()
	return tc
}

// Get retrieves a cached token by its hash
// Returns the claims and true if found and not expired, nil and false otherwise
func (tc *TokenCache) Get(tokenHash string) (map[string]interface{}, bool) {
	tc.mu.RLock()
	defer tc.mu.RUnlock()

	cached, ok := tc.cache[tokenHash]
	if !ok {
		return nil, false
	}

	// Check if expired
	if time.Now().After(cached.validUntil) {
		return nil, false
	}

	// Update last access time (for LRU)
	cached.lastAccess = time.Now()
	return cached.claims, true
}

// Set stores validated token claims in the cache
// tokenHash: SHA-256 hash of the token
// claims: validated JWT claims
// tokenExp: token expiration time from JWT exp claim
// safetyMargin: time to expire cache entry before token expires
func (tc *TokenCache) Set(tokenHash string, claims map[string]interface{}, tokenExp time.Time, safetyMargin time.Duration) {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	// Calculate cache entry expiry (token exp - safety margin)
	validUntil := tokenExp.Add(-safetyMargin)

	// Don't cache if already expired
	if time.Now().After(validUntil) {
		return
	}

	// Evict LRU entries if cache is full
	if len(tc.cache) >= maxCacheSize {
		tc.evictLRU()
	}

	tc.cache[tokenHash] = &cachedToken{
		claims:     claims,
		validUntil: validUntil,
		lastAccess: time.Now(),
	}
}

// Invalidate removes a token from the cache (e.g., on explicit logout)
func (tc *TokenCache) Invalidate(tokenHash string) {
	tc.mu.Lock()
	defer tc.mu.Unlock()
	delete(tc.cache, tokenHash)
}

// Stop stops the cleanup goroutine
func (tc *TokenCache) Stop() {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	if !tc.stopped {
		close(tc.stopCh)
		tc.stopped = true
	}
}

// evictLRU removes the least recently accessed entry
// Must be called with write lock held
func (tc *TokenCache) evictLRU() {
	var oldestHash string
	var oldestTime time.Time
	first := true

	for hash, cached := range tc.cache {
		if first || cached.lastAccess.Before(oldestTime) {
			oldestHash = hash
			oldestTime = cached.lastAccess
			first = false
		}
	}

	if oldestHash != "" {
		delete(tc.cache, oldestHash)
	}
}

// cleanupLoop periodically removes expired entries
func (tc *TokenCache) cleanupLoop() {
	ticker := time.NewTicker(cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			tc.cleanup()
		case <-tc.stopCh:
			return
		}
	}
}

// cleanup removes expired entries from the cache
func (tc *TokenCache) cleanup() {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	now := time.Now()
	for hash, cached := range tc.cache {
		if now.After(cached.validUntil) {
			delete(tc.cache, hash)
		}
	}
}

// Size returns the current number of cached tokens
func (tc *TokenCache) Size() int {
	tc.mu.RLock()
	defer tc.mu.RUnlock()
	return len(tc.cache)
}

// HashToken creates a SHA-256 hash of a token for use as a cache key
// This prevents storing plaintext tokens in cache
func HashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}
