package node

import (
	"container/heap"
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/google/uuid"
	nodemodel "github.com/world-in-progress/yggdrasil/node/model"
)

type (
	IRepository interface {
		Create(ctx context.Context, table string, record map[string]any) (string, error)
		ReadOne(ctx context.Context, table string, filter map[string]any) (map[string]any, error)
		ReadAll(ctx context.Context, table string, filter map[string]any) ([]map[string]any, error)
		Update(ctx context.Context, table string, filter map[string]any, update map[string]any) error
		Delete(ctx context.Context, table string, filter map[string]any) error
		Count(ctx context.Context, table string, filter map[string]any) (int64, error)
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
		modeler   *nodemodel.ModelManager

		mu sync.RWMutex
	}
)

func NewTree(name string, repo IRepository, cacheSize uint) (*Tree, error) {
	t := &Tree{
		repo:      repo,
		nodeCache: sync.Map{},
		cacheSize: int(cacheSize),
		heap:      make(nodeHeap, 0),
	}

	// add node model manager
	modeler, err := nodemodel.NewModelManager()
	if err != nil {
		return nil, fmt.Errorf("failed to create model manager for Tree: %v", err)
	}
	t.modeler = modeler

	heap.Init(&t.heap)
	return t, nil
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
	if err := t.activateNode(ID); err != nil {
		return "", fmt.Errorf("failed to active node: %v", err)
	}
	return ID, nil
}

// GetNode gets a node pointer through cache or deserializing from repository record.
func (t *Tree) GetNode(ID string) (*Node, error) {
	// get node if it is active
	if val, loaded := t.nodeCache.Load(ID); loaded {
		return val.(*Node), nil
	}

	if err := t.activateNode(ID); err != nil {
		return nil, fmt.Errorf("failed to get node in repository: %v", err)
	} else {
		return t.GetNode(ID)
	}
}

// DeleteNode recursively deletes cache and repository record from the provided node
func (t *Tree) DeleteNode(ID string) error {
	// get node
	node, err := t.GetNode(ID)
	if err != nil {
		return fmt.Errorf("failed to get node: %v", err)
	}

	// remove node from parent if parent is active
	if val, loaded := t.nodeCache.Load(node.GetParentID()); loaded {
		val.(*Node).RemoveChild(ID)
	}

	// recursively remove children
	for _, childID := range node.GetChildIDs() {
		if err := t.DeleteNode(childID); err != nil {
			return fmt.Errorf("failed to recursively remove children: %v", err)
		}
	}

	// deactivate
	if err := t.deactivateNode(ID); err != nil {
		return fmt.Errorf("failed to deactivate node: %v", err)
	}

	// delete node record in repository
	ctx := context.Background()
	if err := t.repo.Delete(ctx, "node", map[string]any{"_id": ID}); err != nil {
		return fmt.Errorf("failed to delete node record: %v", err)
	}

	return nil
}

func (t *Tree) UpdateNodeAttribute(ID string, name string, update any) error {
	// get model name
	var modelName string
	if infos := strings.Split(ID, "-"); len(infos) != 6 {
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

// Shrink clear the cache to half its size.
func (t *Tree) Shrink() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.shrinkLocked()
}

// GetActiveNodeNum counts all active nodes in the cache.
func (t *Tree) GetActiveNodeNum() int {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.heap.Len()
}

// GetNodeRecordNum counts all node records in the repository.
func (t *Tree) GetNodeRecordNum() (int64, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	ctx := context.Background()
	if count, err := t.repo.Count(ctx, "node", nil); err != nil {
		return 0, fmt.Errorf("failed to count node record in repository: %v", err)
	} else {
		return count, nil
	}
}

// activateNode activates a node from repository record to the runtime cache.
func (t *Tree) activateNode(ID string) error {
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

// deactivateNode deactivates a node from the runtime cache and updates its repository record.
func (t *Tree) deactivateNode(ID string) error {
	// check if is inactive
	val, loaded := t.nodeCache.LoadAndDelete(ID)
	if !loaded || val == nil {
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

func (t *Tree) shrinkLocked() error {
	toSize := t.cacheSize / 2
	if toSize == 0 {
		toSize = 1
	}

	if t.heap.Len() <= t.cacheSize {
		return nil
	}

	heap.Init(&t.heap)
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
