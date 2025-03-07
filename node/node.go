package node

import (
	"fmt"
	"sync/atomic"
	"time"
)

type (
	Node struct {
		ChildrenIDs []string
		CallTime    time.Time
		dirty       atomic.Bool
		Attributes  map[string]any
	}
)

func NewNode(attributes map[string]any) *Node {
	n := &Node{
		CallTime:    time.Now(),
		Attributes:  attributes,
		ChildrenIDs: make([]string, 0),
	}
	return n
}

func (n *Node) GetCallTime() time.Time {
	return n.CallTime
}

func (n *Node) IsDirty() bool {
	return n.dirty.Load()
}

func (n *Node) MakeDirty() {
	n.dirty.Store(false)
}

func (n *Node) GetID() string {
	n.CallTime = time.Now()
	return n.Attributes["_id"].(string)
}

func (n *Node) GetName() string {
	n.CallTime = time.Now()
	return n.Attributes["name"].(string)
}

func (n *Node) GetParentID() string {
	n.CallTime = time.Now()
	if parentID, ok := n.Attributes["parent"]; ok {
		return parentID.(string)
	} else {
		return ""
	}
}

func (n *Node) GetChildIDs() []string {
	n.CallTime = time.Now()
	return n.ChildrenIDs
}

func (n *Node) GetParam(name string) (any, error) {
	n.CallTime = time.Now()
	if param, ok := n.Attributes[name]; ok {
		return param, nil
	} else {
		return nil, fmt.Errorf("node (ID: %s, Name: %s) does not have attribute named %s", n.Attributes["_id"], n.Attributes["nam"], name)
	}
}

func (n *Node) AddChild(childID string) {
	// Do not update calltime for this AddChild is not called for functional using by outside.
	n.ChildrenIDs = append(n.ChildrenIDs, childID)
}

func (n *Node) UpdateAttribute(name string, update any) (any, error) {
	n.dirty.Store(true)
	n.CallTime = time.Now()
	old, ok := n.Attributes[name]
	if !ok {
		return nil, fmt.Errorf("node (ID: %s, Name: %s) does not hanve attribute named %s", n.Attributes["_id"], n.Attributes["nam"], name)
	}
	n.Attributes[name] = update
	return old, nil
}

func (n *Node) Serialize() map[string]any {
	return n.Attributes
}
