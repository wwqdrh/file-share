package api

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"sync"
)

type Subscriber struct {
	ID       string
	Response http.ResponseWriter
}

var (
	subscribers []Subscriber
	subLock     sync.RWMutex
)

// RegistrySSE registers a new SSE connection
func RegistrySSE(w http.ResponseWriter, r *http.Request) {
	// Set headers for SSE
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Cache-Control", "no-cache")

	// Generate unique ID for subscriber
	subscriberID := generateUUID()
	fmt.Printf("%s Connection connected\n", subscriberID)

	// Send initial registration message
	data := map[string]interface{}{
		"type": "registry",
		"data": map[string]string{
			"id": subscriberID,
		},
	}
	jsonData, _ := json.Marshal(data)
	fmt.Fprintf(w, "%s\n", jsonData)
	w.(http.Flusher).Flush()

	// Add subscriber to list
	subLock.Lock()
	subscribers = append(subscribers, Subscriber{
		ID:       subscriberID,
		Response: w,
	})
	subLock.Unlock()

	// Remove subscriber when connection closes
	notify := w.(http.CloseNotifier).CloseNotify()
	go func() {
		<-notify
		subLock.Lock()
		for i, sub := range subscribers {
			if sub.ID == subscriberID {
				subscribers = append(subscribers[:i], subscribers[i+1:]...)
				break
			}
		}
		subLock.Unlock()
		fmt.Printf("%s Connection closed\n", subscriberID)
	}()
}

// SendEvent sends an event to all connected subscribers
func SendEvent(data interface{}) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("error marshaling event data: %v", err)
	}

	subLock.RLock()
	defer subLock.RUnlock()

	for _, sub := range subscribers {
		fmt.Fprintf(sub.Response, "%s\n", jsonData)
		sub.Response.(http.Flusher).Flush()
	}

	return nil
}

// Helper function to generate UUID
func generateUUID() string {
	// This is a simple implementation. In production, you should use a proper UUID library
	// like github.com/google/uuid
	return fmt.Sprintf("%x-%x-%x-%x-%x",
		rand.Int63(),
		rand.Int63(),
		rand.Int63(),
		rand.Int63(),
		rand.Int63())
}
