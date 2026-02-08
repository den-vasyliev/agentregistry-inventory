package masteragent

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInfraEvent_String(t *testing.T) {
	event := InfraEvent{
		Source:   "k8s/pod/default/nginx",
		Type:     "pod-crash",
		Severity: "critical",
		Message:  "CrashLoopBackOff",
	}
	assert.Equal(t, "critical [k8s/pod/default/nginx] pod-crash: CrashLoopBackOff", event.String())
}

func TestNewEventHub(t *testing.T) {
	tests := []struct {
		name       string
		bufferSize int
		recentMax  int
	}{
		{"positive values", 50, 10},
		{"zero buffer defaults to 1000", 0, 10},
		{"negative buffer defaults to 1000", -1, 10},
		{"zero recentMax defaults to 100", 50, 0},
		{"negative recentMax defaults to 100", 50, -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hub := NewEventHub(tt.bufferSize, tt.recentMax)
			require.NotNil(t, hub)
			assert.Equal(t, 0, hub.QueueDepth())
			assert.Equal(t, int64(0), hub.TotalProcessed())
		})
	}
}

func TestEventHub_PushPop(t *testing.T) {
	hub := NewEventHub(10, 5)

	ok := hub.Push(InfraEvent{
		Source:   "test",
		Type:     "test-event",
		Severity: "info",
		Message:  "hello",
	})
	require.True(t, ok)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	event, ok := hub.Pop(ctx)
	require.True(t, ok)

	assert.NotEmpty(t, event.ID, "ID should be auto-generated")
	assert.False(t, event.Timestamp.IsZero(), "Timestamp should be set")
	assert.Equal(t, "test", event.Source)
	assert.Equal(t, "test-event", event.Type)
	assert.Equal(t, "info", event.Severity)
	assert.Equal(t, "hello", event.Message)
}

func TestEventHub_PushPreservesExistingID(t *testing.T) {
	hub := NewEventHub(10, 5)

	ok := hub.Push(InfraEvent{
		ID:      "my-custom-id",
		Message: "test",
	})
	require.True(t, ok)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	event, ok := hub.Pop(ctx)
	require.True(t, ok)
	assert.Equal(t, "my-custom-id", event.ID)
}

func TestEventHub_PushPreservesExistingTimestamp(t *testing.T) {
	hub := NewEventHub(10, 5)

	ts := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	ok := hub.Push(InfraEvent{
		Timestamp: ts,
		Message:   "test",
	})
	require.True(t, ok)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	event, ok := hub.Pop(ctx)
	require.True(t, ok)
	assert.Equal(t, ts, event.Timestamp)
}

func TestEventHub_QueueFull(t *testing.T) {
	hub := NewEventHub(2, 5)

	assert.True(t, hub.Push(InfraEvent{Message: "1"}))
	assert.True(t, hub.Push(InfraEvent{Message: "2"}))
	assert.False(t, hub.Push(InfraEvent{Message: "3"}), "should return false when queue is full")
}

func TestEventHub_PopCancelled(t *testing.T) {
	hub := NewEventHub(10, 5)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	_, ok := hub.Pop(ctx)
	assert.False(t, ok)
}

func TestEventHub_QueueDepth(t *testing.T) {
	hub := NewEventHub(10, 5)

	assert.Equal(t, 0, hub.QueueDepth())

	hub.Push(InfraEvent{Message: "1"})
	hub.Push(InfraEvent{Message: "2"})
	hub.Push(InfraEvent{Message: "3"})
	assert.Equal(t, 3, hub.QueueDepth())

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	hub.Pop(ctx)
	assert.Equal(t, 2, hub.QueueDepth())
}

func TestEventHub_TotalProcessed(t *testing.T) {
	hub := NewEventHub(10, 5)

	assert.Equal(t, int64(0), hub.TotalProcessed())

	hub.Push(InfraEvent{Message: "1"})
	hub.Push(InfraEvent{Message: "2"})
	assert.Equal(t, int64(2), hub.TotalProcessed())

	// Popping does not change total (total tracks pushes)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	hub.Pop(ctx)
	assert.Equal(t, int64(2), hub.TotalProcessed())
}

func TestEventHub_Recent(t *testing.T) {
	hub := NewEventHub(10, 5)

	for i := range 3 {
		hub.Push(InfraEvent{ID: string(rune('a' + i)), Message: string(rune('a' + i))})
	}

	// Get all recent
	recent := hub.Recent(0)
	assert.Len(t, recent, 3)

	// Get last 2
	recent = hub.Recent(2)
	assert.Len(t, recent, 2)
	assert.Equal(t, "b", recent[0].Message)
	assert.Equal(t, "c", recent[1].Message)

	// Request more than available
	recent = hub.Recent(10)
	assert.Len(t, recent, 3)
}

func TestEventHub_RecentRingBuffer(t *testing.T) {
	hub := NewEventHub(20, 3) // recentMax=3

	// Push 5 events; only last 3 should be kept
	for i := range 5 {
		hub.Push(InfraEvent{Message: string(rune('a' + i))})
	}

	recent := hub.Recent(0)
	require.Len(t, recent, 3)
	assert.Equal(t, "c", recent[0].Message)
	assert.Equal(t, "d", recent[1].Message)
	assert.Equal(t, "e", recent[2].Message)
}

func TestEventHub_ConcurrentPushPop(t *testing.T) {
	hub := NewEventHub(100, 50)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	const pushCount = 50
	var wg sync.WaitGroup

	// Pushers
	for range 5 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range pushCount {
				hub.Push(InfraEvent{Message: "concurrent"})
			}
		}()
	}

	// Poppers
	var popped int64
	var popMu sync.Mutex
	for range 3 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				_, ok := hub.Pop(ctx)
				if !ok {
					return
				}
				popMu.Lock()
				popped++
				popMu.Unlock()
			}
		}()
	}

	// Wait for pushers to finish, then drain
	time.Sleep(100 * time.Millisecond)
	// Cancel to stop poppers once queue is drained
	// Give poppers time to drain
	time.Sleep(50 * time.Millisecond)
	cancel()
	wg.Wait()

	// Total pushed should be 5 * pushCount = 250
	assert.Equal(t, int64(5*pushCount), hub.TotalProcessed())
	// All events should have been popped (some may remain in queue if cancelled early)
	popMu.Lock()
	totalPopped := popped
	popMu.Unlock()
	remaining := hub.QueueDepth()
	assert.Equal(t, int64(5*pushCount), totalPopped+int64(remaining))
}
