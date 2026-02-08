package masteragent

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
)

// InfraEvent represents a normalized infrastructure event
type InfraEvent struct {
	// ID is a unique event identifier
	ID string `json:"id"`
	// Source identifies the origin (e.g., "k8s/pod/namespace/name")
	Source string `json:"source"`
	// Type classifies the event (e.g., "pod-crash", "node-pressure", "webhook")
	Type string `json:"type"`
	// Severity is the event severity: "info", "warning", "critical"
	Severity string `json:"severity"`
	// Message is a human-readable event description
	Message string `json:"message"`
	// Timestamp is when the event occurred
	Timestamp time.Time `json:"timestamp"`
	// Raw contains optional raw event data
	Raw map[string]any `json:"raw,omitempty"`
}

// String returns a human-readable representation of the event
func (e InfraEvent) String() string {
	return e.Severity + " [" + e.Source + "] " + e.Type + ": " + e.Message
}

// EventHub is a bounded channel-based event queue with a ring buffer for recent events
type EventHub struct {
	queue     chan InfraEvent
	recent    []InfraEvent
	recentMax int
	mu        sync.RWMutex
	total     int64
}

// NewEventHub creates an EventHub with the given queue buffer size and recent event capacity
func NewEventHub(bufferSize, recentMax int) *EventHub {
	if bufferSize <= 0 {
		bufferSize = 1000
	}
	if recentMax <= 0 {
		recentMax = 100
	}
	return &EventHub{
		queue:     make(chan InfraEvent, bufferSize),
		recent:    make([]InfraEvent, 0, recentMax),
		recentMax: recentMax,
	}
}

// Push adds an event to the queue. Returns false if the queue is full (non-blocking).
func (h *EventHub) Push(event InfraEvent) bool {
	if event.ID == "" {
		event.ID = uuid.New().String()
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	select {
	case h.queue <- event:
		h.mu.Lock()
		h.total++
		if len(h.recent) >= h.recentMax {
			// Shift ring buffer
			copy(h.recent, h.recent[1:])
			h.recent[len(h.recent)-1] = event
		} else {
			h.recent = append(h.recent, event)
		}
		h.mu.Unlock()
		return true
	default:
		return false
	}
}

// Pop blocks until an event is available or the context is cancelled
func (h *EventHub) Pop(ctx context.Context) (InfraEvent, bool) {
	select {
	case event := <-h.queue:
		return event, true
	case <-ctx.Done():
		return InfraEvent{}, false
	}
}

// QueueDepth returns the current number of events in the queue
func (h *EventHub) QueueDepth() int {
	return len(h.queue)
}

// TotalProcessed returns the total number of events pushed
func (h *EventHub) TotalProcessed() int64 {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.total
}

// Recent returns the last n events (or all if n > available)
func (h *EventHub) Recent(n int) []InfraEvent {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if n <= 0 || n > len(h.recent) {
		n = len(h.recent)
	}
	result := make([]InfraEvent, n)
	copy(result, h.recent[len(h.recent)-n:])
	return result
}
