package threading

import (
	"sync/atomic"
)

type (
	// BaseTask is the basic structure for a Task interface.
	BaseTask struct {
		ID        string
		done      atomic.Bool
		cancelled atomic.Bool
	}

	// TaskCancelFunc is used to cancel the execution of a task. Return false if task has been done.
	TaskCancelFunc func() bool
)

func (bt *BaseTask) GetID() string      { return bt.ID }
func (bt *BaseTask) Complete()          { bt.done.Store(true) }
func (bt *BaseTask) IsCompleted() bool  { return bt.done.Load() }
func (bt *BaseTask) IsCanceled() bool   { return bt.cancelled.Load() }
func (bt *BaseTask) IsIgnoreable() bool { return bt.cancelled.Load() || bt.done.Load() }
func (bt *BaseTask) Cancel() bool {
	if !bt.done.Load() {
		bt.done.Store(true)
		bt.cancelled.Store(true)
		return true
	}
	return false
}
