package componentinterface

import (
	"context"
	"net/http"
	"time"
)

type (
	// INode is the interface for a node belong to a resource tree.
	INode interface {
		GetID() string
		GetName() string
		GetParentID() string
		GetParam(name string) any
	}

	// IComponent is the interface for a component.
	IComponent interface {
		GetID() string
		GetName() string
		GetCallTime() time.Time
		Execute(node INode, params map[string]any, client *http.Client, headers map[string]string) (map[string]any, error)
	}

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

	// IRepository is the interface for CRUD operations of some repository.
	IRepository interface {
		Create(ctx context.Context, table string, record map[string]any) (string, error)
		ReadOne(ctx context.Context, table string, filter map[string]any) (map[string]any, error)
		ReadAll(ctx context.Context, table string, filter map[string]any) ([]map[string]any, error)
		Update(ctx context.Context, table string, filter map[string]any, update map[string]any) error
		Delete(ctx context.Context, table string, filter map[string]any) error
		Count(ctx context.Context, table string, filter map[string]any) (int64, error)
	}
)
