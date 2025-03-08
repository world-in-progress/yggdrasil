package node

import (
	"container/heap"
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/google/uuid"
	"github.com/world-in-progress/yggdrasil/core/logger"
	"github.com/world-in-progress/yggdrasil/model"
)

type (
	IRepository interface {
		Create(ctx context.Context, table string, record map[string]any) (string, error)
		ReadOne(ctx context.Context, table string, filter map[string]any) (map[string]any, error)
		ReadAll(ctx context.Context, table string, filter map[string]any) ([]map[string]any, error)
		Update(ctx context.Context, table string, filter map[string]any, update map[string]any) error
		Delete(ctx context.Context, table string, filter map[string]any) error
	}

	nodeEntry struct {
		index int
		node  *Node
	}

	nodeHeap []*nodeEntry

	Tree struct {
		cacheSize int
		nodeCache sync.Map
		heap      nodeHeap
		repo      IRepository
		modeler   *model.ModelManager

		mu sync.RWMutex
	}
)

func NewTree(name string, repo IRepository, cacheSize uint) *Tree {
	t := &Tree{
		repo:      repo,
		nodeCache: sync.Map{},
		cacheSize: int(cacheSize),
		heap:      make(nodeHeap, 0),
	}

	var err error
	t.modeler, err = model.NewModelManager()
	if err != nil {
		logger.Error("Failed to create model manager for Tree: %v", err)
	}
	heap.Init(&t.heap)
	return t
}

// RegisterNode records node information to repository and activates the node in the runtime cache.
func (t *Tree) RegisterNode(modelName string, nodeInfo map[string]any) (string, error) {
	// check validation
	if err := t.modeler.Validate(modelName, nodeInfo); err != nil {
		return "", fmt.Errorf("nodeInfo %v provided for node registration is invalid: %v", nodeInfo, err)
	}

	// create uuid
	ID := modelName + "-" + uuid.New().String()
	nodeInfo["_id"] = ID

	// create node info to repository
	ctx := context.Background()
	if _, err := t.repo.Create(ctx, "node", nodeInfo); err != nil {
		return "", fmt.Errorf("failed to create node %v: %v", nodeInfo, err)
	}

	// active node
	if err := t.ActivateNode(ID); err != nil {
		return "", fmt.Errorf("failed to active node: %v", err)
	}
	return ID, nil
}

func (t *Tree) GetNode(ID string) (*Node, error) {
	// get node if it is active
	if val, loaded := t.nodeCache.Load(ID); loaded {
		return val.(*Node), nil
	}

	if err := t.ActivateNode(ID); err != nil {
		return nil, fmt.Errorf("failed to get node in repository: %v", err)
	} else {
		return t.GetNode(ID)
	}
}

// ActivateNode activates a node from repository record to the runtime cache.
func (t *Tree) ActivateNode(ID string) error {
	// check if is active
	if _, loaded := t.nodeCache.LoadOrStore(ID, nil); loaded {
		return nil
	}

	// find if is in repository
	ctx := context.Background()
	nodeInfo, err := t.repo.ReadOne(ctx, "node", map[string]any{"_id": ID})
	if err != nil {
		t.nodeCache.Delete(ID)
		return fmt.Errorf("cannot activate node not existing: %v", err)
	}

	// activate node
	node := NewNode(nodeInfo)
	t.nodeCache.Store(ID, node)

	// record children ID through repository
	if childInfos, err := t.repo.ReadAll(ctx, "node", map[string]any{"parent": ID}); err != nil {
		t.nodeCache.Delete(ID)
		return fmt.Errorf("failed to find children of node: %v", err)
	} else {
		for _, childInfo := range childInfos {
			node.childrenIDs = append(node.childrenIDs, childInfo["_id"].(string))
		}
	}

	// update ChildIDs of parent node
	if parentID := node.GetParentID(); parentID != "" {
		if val, loaded := t.nodeCache.Load(parentID); loaded {
			val.(*Node).AddChild(ID)
		}
	}

	return t.addToHeap(node)
}

// DeactivateNode deactivates a node from the runtime cache and updates its repository record.
func (t *Tree) DeactivateNode(ID string) error {
	// check if is inactive
	val, loaded := t.nodeCache.LoadAndDelete(ID)
	if !loaded {
		return nil
	}
	node := val.(*Node)

	// update node record in repository if is dirty
	if node.IsDirty() {
		ctx := context.Background()
		if err := t.repo.Update(ctx, "node", map[string]any{"_id": ID}, map[string]any{"$set": node.Serialize()}); err != nil {
			t.nodeCache.Store(ID, node) // rollback
			return fmt.Errorf("failed to update node record in repository: %v", err)
		}
	}

	// remove from heap
	t.removeFromHeap(node)
	return nil
}

func (t *Tree) UpdateNodeAttribute(ID string, name string, update any) error {
	// get model name
	var modelName string
	if infos := strings.Split(ID, "-"); len(infos) != 5 {
		return fmt.Errorf("provided ID %s is not valid", ID)
	} else {
		modelName = infos[0]
		if !t.modeler.HasModel(modelName) {
			return fmt.Errorf("model name %s is not declared in model manager", modelName)
		}
	}

	// check if update data is valid
	if err := t.modeler.ValidateField(modelName, name, update); err != nil {
		return fmt.Errorf("update data is not valid: %v", err)
	}

	// update cache if node is active
	if val, ok := t.nodeCache.Load(ID); ok {
		node := val.(*Node)
		if _, err := node.UpdateAttribute(name, update); err != nil {
			return fmt.Errorf("failed to update node attribute: %v", err)
		}
		t.updateHeap(node)
		return nil
	}

	// update repository record if node is inactive
	ctx := context.Background()
	filter := map[string]any{"_id": ID}
	updateData := map[string]any{"$set": map[string]any{name: update}}
	if err := t.repo.Update(ctx, "node", filter, updateData); err != nil {
		return fmt.Errorf("failed to update node record in repository: %v", err)
	}
	return nil
}

func (t *Tree) Shrink() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.shrinkLocked()
}

func (t *Tree) GetActiveNodeNum() int {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.heap.Len()
}

func (t *Tree) shrinkLocked() error {
	toSize := t.cacheSize / 2
	if toSize == 0 {
		toSize = 1
	}

	for t.heap.Len() > toSize {

		// remove from heap
		entry := heap.Pop(&t.heap).(*nodeEntry)

		// deactivate node
		node := entry.node
		ID := node.GetID()
		t.nodeCache.Delete(ID)

		// update node record in repository if is dirty
		if node.IsDirty() {
			ctx := context.Background()
			if err := t.repo.Update(ctx, "node", map[string]any{"_id": ID}, map[string]any{"$set": node.Serialize()}); err != nil {
				t.nodeCache.Store(ID, node) // rollback
				return fmt.Errorf("failed to update node record in repository: %v", err)
			}
		}
	}
	return nil
}

func (t *Tree) addToHeap(node *Node) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	entry := &nodeEntry{node: node}
	heap.Push(&t.heap, entry)
	return t.shrinkLocked()
}

func (t *Tree) updateHeap(node *Node) {
	t.mu.Lock()
	defer t.mu.Unlock()

	for _, entry := range t.heap {
		if entry.node == node {
			heap.Fix(&t.heap, entry.index)
		}
	}
}

func (t *Tree) removeFromHeap(node *Node) {
	t.mu.Lock()
	defer t.mu.Unlock()

	for i, entry := range t.heap {
		if entry.node == node {
			heap.Remove(&t.heap, i)
			break
		}
	}
}

func (h nodeHeap) Len() int           { return len(h) }
func (h nodeHeap) Less(i, j int) bool { return h[i].node.GetCallTime().Before(h[j].node.GetCallTime()) }
func (h nodeHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
	h[i].index = i
	h[j].index = j
}
func (h *nodeHeap) Push(x any) {
	entry := x.(*nodeEntry)
	entry.index = len(*h)
	*h = append(*h, entry)
}
func (h *nodeHeap) Pop() any {
	old := *h
	n := len(old)
	entry := old[n-1]
	old[n-1] = nil
	entry.index = -1
	*h = old[0 : n-1]
	return entry
}
