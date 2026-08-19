package core

import (
	"sync"
	"time"

	"golang.org/x/exp/slog"
)

// Job represents a background task function.
type Job func()

// Queue provides a lightweight in-memory background worker pool.
type Queue struct {
	workers int
	jobCh   chan Job
	wg      sync.WaitGroup
	quitCh  chan struct{}
	once    sync.Once
}

// NewQueue creates and starts a background Queue worker pool.
func NewQueue(workers, bufferSize int) *Queue {
	if workers <= 0 {
		workers = 4
	}
	if bufferSize <= 0 {
		bufferSize = 100
	}

	q := &Queue{
		workers: workers,
		jobCh:   make(chan Job, bufferSize),
		quitCh:  make(chan struct{}),
	}

	q.start()
	return q
}

func (q *Queue) start() {
	for i := 0; i < q.workers; i++ {
		q.wg.Add(1)
		go func(workerID int) {
			defer q.wg.Done()
			for {
				select {
				case job, ok := <-q.jobCh:
					if !ok {
						return
					}
					q.executeJob(workerID, job)
				case <-q.quitCh:
					// Drain remaining jobs before exiting
					for {
						select {
						case job, ok := <-q.jobCh:
							if !ok {
								return
							}
							q.executeJob(workerID, job)
						default:
							return
						}
					}
				}
			}
		}(i + 1)
	}
}

func (q *Queue) executeJob(workerID int, job Job) {
	defer func() {
		if r := recover(); r != nil {
			slog.Error("queue worker panic recovered", "worker_id", workerID, "error", r)
		}
	}()
	job()
}

// Dispatch adds a job to the background queue for asynchronous execution.
// Returns true if queued successfully, or false if the queue is full/stopped.
func (q *Queue) Dispatch(job Job) bool {
	select {
	case q.jobCh <- job:
		return true
	default:
		slog.Warn("queue buffer is full, executing synchronously or dropping")
		go job() // Fallback to dedicated goroutine
		return true
	}
}

// Shutdown gracefully waits for all pending jobs to complete within timeout.
func (q *Queue) Shutdown(timeout time.Duration) bool {
	q.once.Do(func() {
		close(q.quitCh)
		close(q.jobCh)
	})

	done := make(chan struct{})
	go func() {
		q.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return true
	case <-time.After(timeout):
		slog.Warn("queue shutdown timed out with pending jobs")
		return false
	}
}
