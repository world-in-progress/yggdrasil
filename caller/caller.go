package caller

import "sync/atomic"

type (
	// ITask is the interface for a worker task.
	ITask interface {
		GetID() string
		Process() error
		Cancel() bool
		Complete()
		IsCanceled() bool
		IsCompleted() bool
		IsIgnoreable() bool
	}

	INode interface {
		GetID() string
		GetName() string
		GetParentID() string
		GetParam(name string) (any, error)
		Serialize() map[string]any
	}

	BaseTask struct {
		ID        string
		WorkerID  string
		done      atomic.Bool
		cancelled atomic.Bool
	}
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
