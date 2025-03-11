package node

import (
	"fmt"
	"sync/atomic"
	"time"
)

type (
	Node struct {
		childrenIDs []string
		callTime    time.Time
		dirty       atomic.Bool
		attributes  map[string]any
	}
)

func NewNode(attributes map[string]any) *Node {
	n := &Node{
		callTime:    time.Now(),
		attributes:  attributes,
		childrenIDs: make([]string, 0),
	}

	if _, ok := n.attributes["components"]; !ok {
		n.attributes["components"] = make([]string, 0)
	}

	return n
}

func (n *Node) GetCallTime() time.Time {
	return n.callTime
}

func (n *Node) IsDirty() bool {
	return n.dirty.Load()
}

func (n *Node) MakeDirty() {
	n.dirty.Store(false)
}

func (n *Node) GetID() string {
	n.callTime = time.Now()
	return n.attributes["_id"].(string)
}

func (n *Node) GetName() string {
	n.callTime = time.Now()
	return n.attributes["name"].(string)
}

func (n *Node) GetParentID() string {
	n.callTime = time.Now()
	if parentID, ok := n.attributes["parent"]; ok {
		return parentID.(string)
	} else {
		return ""
	}
}

func (n *Node) GetChildIDs() []string {
	n.callTime = time.Now()
	return n.childrenIDs
}

func (n *Node) GetParam(name string) any {
	n.callTime = time.Now()
	if param, ok := n.attributes[name]; ok {
		return param
	} else {
		return nil
	}
}

func (n *Node) AddChild(childID string) {
	// Do not update calltime because AddChild is not called for functional using by outside.
	n.childrenIDs = append(n.childrenIDs, childID)
}

func (n *Node) RemoveChild(childID string) {
	// Do not update calltime because AddChild is not called for functional using by outside.
	for i, ID := range n.childrenIDs {
		if ID == childID {
			n.childrenIDs[i], n.childrenIDs[len(n.childrenIDs)-1] = n.childrenIDs[len(n.childrenIDs)-1], n.childrenIDs[i]
			n.childrenIDs = n.childrenIDs[:len(n.childrenIDs)-1]
		}
	}
}

func (n *Node) UpdateAttribute(name string, update any) (any, error) {
	n.dirty.Store(true)
	n.callTime = time.Now()
	old, ok := n.attributes[name]
	if !ok {
		return nil, fmt.Errorf("node (ID: %s, Name: %s) does not hanve attribute named %s", n.attributes["_id"], n.attributes["nam"], name)
	}
	n.attributes[name] = update
	return old, nil
}

func (n *Node) AddComponent(compoID string) bool {
	// If component exists, then return
	components, _ := n.attributes["components"].([]string)
	for _, id := range components {
		if id == compoID {
			return false
		}
	}

	n.dirty.Store(true)
	n.callTime = time.Now()
	n.attributes["components"] = append(n.attributes["components"].([]string), compoID)
	return true
}

func (n *Node) DeleteComponent(compoID string) bool {
	components, _ := n.attributes["components"].([]string)
	for i, id := range components {
		if id == compoID {
			n.dirty.Store(true)
			n.callTime = time.Now()

			n.attributes["components"] = append(components[:i], components[i+1:]...)
			return true
		}
	}

	return false
}

func (n *Node) Serialize() map[string]any {
	return n.attributes
}
