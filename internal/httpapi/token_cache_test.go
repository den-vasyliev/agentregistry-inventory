package httpapi

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewTokenCache(t *testing.T) {
	t.Parallel()
	tc := NewTokenCache()
	assert.NotNil(t, tc)
	assert.NotNil(t, tc.cache)
	assert.Equal(t, 0, tc.Size())
	tc.Stop()
}

func TestTokenCache_SetAndGet(t *testing.T) {
	t.Parallel()
	tc := NewTokenCache()
	defer tc.Stop()

	tokenHash := "test-hash-1"
	claims := map[string]interface{}{
		"sub":   "user123",
		"email": "user@example.com",
	}
	tokenExp := time.Now().Add(1 * time.Hour)
	safetyMargin := 5 * time.Minute

	// Set token
	tc.Set(tokenHash, claims, tokenExp, safetyMargin)

	// Get token - should exist
	retrievedClaims, found := tc.Get(tokenHash)
	assert.True(t, found)
	assert.Equal(t, claims, retrievedClaims)
}

func TestTokenCache_Get_NotFound(t *testing.T) {
	t.Parallel()
	tc := NewTokenCache()
	defer tc.Stop()

	_, found := tc.Get("non-existent-hash")
	assert.False(t, found)
}

func TestTokenCache_Get_Expired(t *testing.T) {
	t.Parallel()
	tc := NewTokenCache()
	defer tc.Stop()

	tokenHash := "expired-token"
	claims := map[string]interface{}{"sub": "user123"}
	// Token expired 10 minutes ago
	tokenExp := time.Now().Add(-10 * time.Minute)
	safetyMargin := 5 * time.Minute

	tc.Set(tokenHash, claims, tokenExp, safetyMargin)

	// Should not find expired token
	_, found := tc.Get(tokenHash)
	assert.False(t, found)
}

func TestTokenCache_Set_AlreadyExpired(t *testing.T) {
	t.Parallel()
	tc := NewTokenCache()
	defer tc.Stop()

	tokenHash := "already-expired"
	claims := map[string]interface{}{"sub": "user123"}
	// Token that expires in 1 minute, but safety margin is 5 minutes
	// So cache entry would be already expired
	tokenExp := time.Now().Add(1 * time.Minute)
	safetyMargin := 5 * time.Minute

	tc.Set(tokenHash, claims, tokenExp, safetyMargin)

	// Should not be cached since it's already expired
	assert.Equal(t, 0, tc.Size())
}

func TestTokenCache_Invalidate(t *testing.T) {
	t.Parallel()
	tc := NewTokenCache()
	defer tc.Stop()

	tokenHash := "to-invalidate"
	claims := map[string]interface{}{"sub": "user123"}
	tokenExp := time.Now().Add(1 * time.Hour)

	tc.Set(tokenHash, claims, tokenExp, 5*time.Minute)
	assert.Equal(t, 1, tc.Size())

	tc.Invalidate(tokenHash)
	assert.Equal(t, 0, tc.Size())

	_, found := tc.Get(tokenHash)
	assert.False(t, found)
}

func TestTokenCache_LRU_Eviction(t *testing.T) {
	// Create cache with small max size for testing
	tc := NewTokenCache()
	defer tc.Stop()

	// Override maxCacheSize for testing (normally 1000)
	// Fill cache beyond capacity
	tokenExp := time.Now().Add(1 * time.Hour)

	// Add maxCacheSize + 1 tokens
	for i := 0; i < maxCacheSize+10; i++ {
		tokenHash := fmt.Sprintf("token-%d", i)
		claims := map[string]interface{}{"index": i}
		tc.Set(tokenHash, claims, tokenExp, 5*time.Minute)
		// Small delay to ensure different lastAccess times
		time.Sleep(1 * time.Millisecond)
	}

	// Cache should not exceed max size
	assert.LessOrEqual(t, tc.Size(), maxCacheSize)
}

func TestTokenCache_ConcurrentAccess(t *testing.T) {
	tc := NewTokenCache()
	defer tc.Stop()

	var wg sync.WaitGroup
	numGoroutines := 100
	operationsPerGoroutine := 100

	// Concurrent writes
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < operationsPerGoroutine; j++ {
				tokenHash := fmt.Sprintf("token-%d-%d", id, j)
				claims := map[string]interface{}{"goroutine": id, "op": j}
				tokenExp := time.Now().Add(1 * time.Hour)
				tc.Set(tokenHash, claims, tokenExp, 5*time.Minute)
			}
		}(i)
	}

	// Concurrent reads
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < operationsPerGoroutine; j++ {
				tokenHash := fmt.Sprintf("token-%d-%d", id, j)
				tc.Get(tokenHash)
			}
		}(i)
	}

	wg.Wait()

	// Should not panic or deadlock, and size should be reasonable
	assert.LessOrEqual(t, tc.Size(), maxCacheSize)
}

func TestTokenCache_Cleanup(t *testing.T) {
	tc := NewTokenCache()
	defer tc.Stop()

	// Add a token that will expire soon
	tokenHash := "soon-expired"
	claims := map[string]interface{}{"sub": "user123"}
	// Token expires in 100ms, safety margin 50ms
	// So cache entry expires in 50ms
	tokenExp := time.Now().Add(100 * time.Millisecond)
	tc.Set(tokenHash, claims, tokenExp, 50*time.Millisecond)

	assert.Equal(t, 1, tc.Size())

	// Wait for expiration (150ms should be enough)
	time.Sleep(150 * time.Millisecond)

	// Manually trigger cleanup for testing
	tc.cleanup()

	// Token should be cleaned up
	assert.Equal(t, 0, tc.Size())
}

func TestHashToken(t *testing.T) {
	t.Parallel()
	// Same token should produce same hash
	token1 := "my-test-token"
	hash1 := HashToken(token1)
	hash2 := HashToken(token1)
	assert.Equal(t, hash1, hash2)

	// Different tokens should produce different hashes
	token2 := "different-token"
	hash3 := HashToken(token2)
	assert.NotEqual(t, hash1, hash3)

	// Hash should be valid hex string
	assert.Equal(t, 64, len(hash1)) // SHA-256 produces 64 hex chars
}

func TestTokenCache_Stop(t *testing.T) {
	tc := NewTokenCache()

	// Stop should not panic
	tc.Stop()

	// Multiple stops should not panic
	tc.Stop()
}
