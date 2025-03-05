package threading

import (
	"sync/atomic"
)

type (
	// Task is the interface for a worker task.
	Task interface {
		Handler() error
	}

	// BaseTask is the basic structure for a worker task.
	BaseTask struct {
		ID       string
		WorkerID string
	}

	// TaskEntry is the structure to hold a task and its context.
	TaskEntry struct {
		task      Task
		cancelled atomic.Bool
	}
)

func NewTaskEntry(task Task) *TaskEntry {
	return &TaskEntry{task: task}
}

func (te *TaskEntry) Cancel() {
	te.cancelled.Store(true)
}

func (te *TaskEntry) IsCancelled() bool {
	return te.cancelled.Load()
}
