package component

import (
	"encoding/json"
	"fmt"
)

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
	}
)

func ConvertToStruct[T any](source any) (T, error) {
	var result T

	bytes, err := json.Marshal(source)
	if err != nil {
		return result, fmt.Errorf("marshal error: %v", err)
	}

	err = json.Unmarshal(bytes, &result)
	if err != nil {
		return result, fmt.Errorf("unmarshal error: %v", err)
	}

	return result, nil
}
