package hub

import (
	"testing"
	"time"
)

func TestHubBroadcastUnregistersSlowClient(t *testing.T) {
	t.Parallel()

	h := New()
	client := &Client{
		hub:  h,
		send: make(chan []byte, 1),
	}

	h.Register(client)
	waitUntil(t, time.Second, func() bool {
		h.mu.RLock()
		defer h.mu.RUnlock()
		return h.clients[client]
	})

	client.send <- []byte("already-full")
	h.Broadcast([]byte("next-message"))

	waitUntil(t, time.Second, func() bool {
		h.mu.RLock()
		defer h.mu.RUnlock()
		_, ok := h.clients[client]
		return !ok
	})
}

func waitUntil(t *testing.T, timeout time.Duration, predicate func() bool) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if predicate() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}

	t.Fatal("condition was not met before timeout")
}
