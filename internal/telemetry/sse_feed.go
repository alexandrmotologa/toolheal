package telemetry

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
)

// SSEFeed distributes real-time invocation events to connected web console clients.
type SSEFeed struct {
	mu      sync.RWMutex
	clients map[chan []byte]bool
}

// NewSSEFeed initializes an SSE broker.
func NewSSEFeed() *SSEFeed {
	return &SSEFeed{
		clients: make(map[chan []byte]bool),
	}
}

// Subscribe registers a new HTTP client connection for live telemetry updates.
func (f *SSEFeed) Subscribe() chan []byte {
	f.mu.Lock()
	defer f.mu.Unlock()

	ch := make(chan []byte, 32)
	f.clients[ch] = true
	return ch
}

// Unsubscribe removes a client channel.
func (f *SSEFeed) Unsubscribe(ch chan []byte) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if _, ok := f.clients[ch]; ok {
		delete(f.clients, ch)
		close(ch)
	}
}

// Broadcast sends an event payload to all currently connected listeners.
func (f *SSEFeed) Broadcast(eventType string, data any) {
	bytes, err := json.Marshal(data)
	if err != nil {
		return
	}

	payload := []byte(fmt.Sprintf("event: %s\ndata: %s\n\n", eventType, string(bytes)))

	f.mu.RLock()
	defer f.mu.RUnlock()

	for ch := range f.clients {
		select {
		case ch <- payload:
		default:
			// Non-blocking drop if client buffer is saturated
		}
	}
}

// ServeHTTP handles the SSE connection streaming loop.
func (f *SSEFeed) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	clientChan := f.Subscribe()
	defer f.Unsubscribe(clientChan)

	// Send initial ping
	fmt.Fprintf(w, ": connected to toolheal telemetry stream\n\n")
	flusher.Flush()

	notify := r.Context().Done()
	for {
		select {
		case <-notify:
			return
		case msg, ok := <-clientChan:
			if !ok {
				return
			}
			_, _ = w.Write(msg)
			flusher.Flush()
		}
	}
}
