package core

import (
	"sync"
	"testing"
	"time"
)

func TestEventDispatcherSyncAndAsync(t *testing.T) {
	queue := NewQueue(2, 50)
	defer queue.Shutdown(time.Second)

	events := NewEventDispatcher(queue)

	var syncCalled bool
	var syncPayload string
	events.Listen("user.registered", func(payload interface{}) {
		syncCalled = true
		syncPayload = payload.(string)
	})

	// 1. Test Synchronous Dispatch
	events.Dispatch("user.registered", "user@example.com")
	if !syncCalled || syncPayload != "user@example.com" {
		t.Errorf("sync event dispatch failed: called=%v, payload=%q", syncCalled, syncPayload)
	}

	// 2. Test Asynchronous Dispatch
	var wg sync.WaitGroup
	wg.Add(1)
	var asyncCalled bool

	events.Listen("invoice.paid", func(payload interface{}) {
		asyncCalled = true
		wg.Done()
	})

	events.DispatchAsync("invoice.paid", 5000)

	// Wait for background worker to execute
	wg.Wait()
	if !asyncCalled {
		t.Error("expected async event listener to execute")
	}

	// 3. Test Unlisten
	events.Unlisten("user.registered")
	syncCalled = false
	events.Dispatch("user.registered", "newuser@example.com")
	if syncCalled {
		t.Error("unlisten failed: callback was still executed")
	}
}
