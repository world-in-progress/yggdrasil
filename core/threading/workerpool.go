package threading

import (
	"fmt"
	"strconv"
	"time"
)

// ErrProcessTimeout returned by WorkerPool to indicate that there no free goroutines during some period of time.
var ErrProcessTimeout = fmt.Errorf("process error: timed out")

type (
	WorkerPool struct {
		workers chan struct{}
		tasks   chan *TaskEntry
	}
)

func NewWorkerPool(maxWorkerNum int, bufferSize int, spawnWorkerNum int) *WorkerPool {
	if spawnWorkerNum <= 0 && bufferSize > 0 {
		panic("dead queue configuration detected")
	}
	if spawnWorkerNum > maxWorkerNum {
		panic("spawn worker num larger than max worker num")
	}

	wp := &WorkerPool{
		workers: make(chan struct{}, maxWorkerNum),
		tasks:   make(chan *TaskEntry, bufferSize),
	}

	for range spawnWorkerNum {
		wp.workers <- struct{}{}
		NewWorker(strconv.Itoa(len(wp.workers)), wp.tasks, nil)
	}

	return wp
}

func (wp *WorkerPool) Submit(task Task) {
	wp.process(task, nil)
}

func (wp *WorkerPool) SubmitTimeout(timeout time.Duration, task Task) error {
	return wp.process(task, time.After(timeout))
}

func (wp *WorkerPool) process(task Task, timeout <-chan time.Time) error {
	entry := NewTaskEntry(task)

	select {
	case <-timeout:
		return ErrProcessTimeout

	case wp.tasks <- entry:
		return nil

	case wp.workers <- struct{}{}:
		NewWorker(strconv.Itoa(len(wp.workers)), wp.tasks, entry)
		return nil
	}
}
