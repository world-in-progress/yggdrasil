package component

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

	// INode is the interface for a node belong to a resource tree.
	INode interface {
		GetID() string
		GetName() string
		GetParentID() string
		GetParam(name string) (any, error)
		Serialize() map[string]any
	}
)
