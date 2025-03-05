package threading

import (
	"sync/atomic"
)

type (
	// Task is the interface for a worker task.
	Task interface {
		GetID() string
		Process() error
	}

	// BaseTask is the basic structure for a worker task.
	BaseTask struct {
		ID       string
		WorkerID string
	}

	// TaskEntry is the structure to hold a task and its context.
	TaskEntry struct {
		task      Task
		done      atomic.Bool
		cancelled atomic.Bool
	}

	// TaskEntryCancelFunc is used to cancel the execution of a task. Return false if task has been done.
	TaskEntryCancelFunc func() bool
)

func NewTaskEntry(task Task) *TaskEntry {
	return &TaskEntry{task: task}
}

func (te *TaskEntry) Complete() {
	te.done.Store(true)
}

func (te *TaskEntry) IsIgnoreable() bool {
	return te.cancelled.Load() || te.done.Load()
}

func (te *TaskEntry) IsCompleted() bool {
	return te.done.Load()
}

func (te *TaskEntry) Cancel() bool {
	if !te.done.Load() {
		te.done.Store(true)
		te.cancelled.Store(true)
		return true
	}
	return false
}

func (te *TaskEntry) IsCancelled() bool {
	return te.cancelled.Load()
}
