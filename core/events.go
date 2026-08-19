package core

import (
	"sync"

	"golang.org/x/exp/slog"
)

// EventCallback defines the signature for event listener functions.
type EventCallback func(payload interface{})

// EventDispatcher manages event registrations and broadcasts.
type EventDispatcher struct {
	listeners map[string][]EventCallback
	queue     *Queue
	mu        sync.RWMutex
}

// NewEventDispatcher initializes an event dispatcher attached to an optional background queue.
func NewEventDispatcher(queue *Queue) *EventDispatcher {
	return &EventDispatcher{
		listeners: make(map[string][]EventCallback),
		queue:     queue,
	}
}

// Listen registers a callback listener for a named event.
func (ed *EventDispatcher) Listen(event string, callback EventCallback) {
	ed.mu.Lock()
	defer ed.mu.Unlock()

	ed.listeners[event] = append(ed.listeners[event], callback)
}

// Dispatch invokes all listeners registered for the event synchronously.
func (ed *EventDispatcher) Dispatch(event string, payload interface{}) {
	ed.mu.RLock()
	callbacks, exists := ed.listeners[event]
	if !exists || len(callbacks) == 0 {
		ed.mu.RUnlock()
		return
	}

	// Make a copy so we don't hold lock during execution
	targets := make([]EventCallback, len(callbacks))
	copy(targets, callbacks)
	ed.mu.RUnlock()

	for _, cb := range targets {
		ed.executeCallback(event, cb, payload)
	}
}

// DispatchAsync runs all listeners in the background queue asynchronously without blocking.
func (ed *EventDispatcher) DispatchAsync(event string, payload interface{}) {
	if ed.queue == nil {
		go ed.Dispatch(event, payload)
		return
	}

	ed.queue.Dispatch(func() {
		ed.Dispatch(event, payload)
	})
}

func (ed *EventDispatcher) executeCallback(event string, cb EventCallback, payload interface{}) {
	defer func() {
		if r := recover(); r != nil {
			slog.Error("event listener panic recovered", "event", event, "error", r)
		}
	}()
	cb(payload)
}

// Unlisten removes all listeners for a given event name.
func (ed *EventDispatcher) Unlisten(event string) {
	ed.mu.Lock()
	defer ed.mu.Unlock()
	delete(ed.listeners, event)
}
