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
		tasks   chan Task
		workers chan struct{}
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
		tasks:   make(chan Task, bufferSize),
		workers: make(chan struct{}, maxWorkerNum),
	}

	for range spawnWorkerNum {
		wp.workers <- struct{}{}
		NewWorker(strconv.Itoa(len(wp.workers)), wp.tasks, nil)
	}

	return wp
}

func (wp *WorkerPool) Submit(task Task) (TaskCancelFunc, error) {
	return wp.process(task, nil)
}

func (wp *WorkerPool) SubmitTimeout(timeout time.Duration, task Task) (TaskCancelFunc, error) {
	return wp.process(task, time.After(timeout))
}

func (wp *WorkerPool) process(task Task, timeout <-chan time.Time) (TaskCancelFunc, error) {

	select {
	case <-timeout:
		return nil, ErrProcessTimeout

	case wp.tasks <- task:
		return task.Cancel, nil

	case wp.workers <- struct{}{}:
		NewWorker(strconv.Itoa(len(wp.workers)), wp.tasks, task)
		return task.Cancel, nil
	}
}
