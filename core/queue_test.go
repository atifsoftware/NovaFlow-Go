package core

import (
	"sync/atomic"
	"testing"
	"time"
)

func TestQueueDispatchAndWorkerExecution(t *testing.T) {
	q := NewQueue(2, 50)
	defer q.Shutdown(2 * time.Second)

	var counter int64

	for i := 0; i < 10; i++ {
		q.Dispatch(func() {
			atomic.AddInt64(&counter, 1)
		})
	}

	// Wait briefly for workers to finish
	time.Sleep(100 * time.Millisecond)

	val := atomic.LoadInt64(&counter)
	if val != 10 {
		t.Fatalf("expected counter to be 10, got %d", val)
	}
}

func TestQueuePanicRecovery(t *testing.T) {
	q := NewQueue(2, 10)
	defer q.Shutdown(2 * time.Second)

	var executedAfterPanic int64

	// Job 1 panics
	q.Dispatch(func() {
		panic("worker panic test")
	})

	// Job 2 should still run successfully after worker recovers
	q.Dispatch(func() {
		atomic.StoreInt64(&executedAfterPanic, 1)
	})

	time.Sleep(100 * time.Millisecond)

	if atomic.LoadInt64(&executedAfterPanic) != 1 {
		t.Fatal("expected queue worker pool to remain healthy and execute job after panic")
	}
}
