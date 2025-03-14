package threading

import (
	"fmt"
	"sync"
	"time"
)

// ErrProcessTimeout returned by WorkerPool to indicate that there no free goroutines during some period of time.
var ErrProcessTimeout = fmt.Errorf("process error: timed out")

type (
	// ITask is the interface for a worker task.
	ITask interface {
		GetID() string
		Cancel() bool
		Process()
		Complete()
		IsCanceled() bool
		IsCompleted() bool
		IsIgnoreable() bool
	}

	WorkerPool struct {
		minWorkerNum int
		tasks        chan ITask
		tokens       chan struct{}
		mu           sync.RWMutex
	}
)

func NewWorkerPool(minWorkerNum int, maxWorkerNum int, bufferSize int) *WorkerPool {

	wp := &WorkerPool{
		minWorkerNum: minWorkerNum,
		tasks:        make(chan ITask, bufferSize),
		tokens:       make(chan struct{}, maxWorkerNum),
	}

	for range minWorkerNum {
		wp.tokens <- struct{}{}
		NewWorker(wp.tasks, wp.tokens, nil)
	}
	return wp
}

func (wp *WorkerPool) Shutdown() {
	close(wp.tasks)

	for task := range wp.tasks {
		task.Cancel()
	}

	close(wp.tokens)
}

func (wp *WorkerPool) GetWorkerCount() int {
	wp.mu.RLocker().Lock()
	defer wp.mu.RUnlock()
	return len(wp.tokens)
}

func (wp *WorkerPool) Submit(task ITask) (TaskCancelFunc, error) {
	return wp.dispatch(task, nil)
}

func (wp *WorkerPool) SubmitTimeout(timeout time.Duration, task ITask) (TaskCancelFunc, error) {
	return wp.dispatch(task, time.After(timeout))
}

func (wp *WorkerPool) dispatch(task ITask, timeout <-chan time.Time) (TaskCancelFunc, error) {
	if timeout == nil {
		timeout = make(chan time.Time)
	}

	select {
	case <-timeout:
		return nil, ErrProcessTimeout
	case wp.tasks <- task:
		return task.Cancel, nil
	case wp.tokens <- struct{}{}:
		NewWorker(wp.tasks, wp.tokens, task)
		return task.Cancel, nil
	}
}
