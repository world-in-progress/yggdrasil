package component

import (
	"container/heap"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	componentinterface "github.com/world-in-progress/yggdrasil/component/interface"
	"github.com/world-in-progress/yggdrasil/component/restfulcomponent"
)

type (
	ComponentType string

	componentEntry struct {
		index int
		id    string
		compo componentinterface.IComponent
	}

	componentHeap []*componentEntry

	ComponentManager struct {
		cacheSize      int
		componentCache sync.Map
		heap           componentHeap
		repo           componentinterface.IRepository

		mu sync.RWMutex
	}
)

const (
	GRPC    ComponentType = "GRPC"
	Local   ComponentType = "LOCAL"
	Restful ComponentType = "RESTFUL"
	Runtime ComponentType = "RUNTIME"
)

func NewComponentManager(name string, repo componentinterface.IRepository, cacheSize uint) (*ComponentManager, error) {
	t := &ComponentManager{
		repo:           repo,
		componentCache: sync.Map{},
		cacheSize:      int(cacheSize),
		heap:           make(componentHeap, 0),
	}

	heap.Init(&t.heap)
	return t, nil
}

func (c *ComponentManager) RegisterComponent(compoType ComponentType, providedSchema any) (string, error) {
	// convert compoSchema to type of map[string]any
	var schemaMap map[string]any
	switch t := providedSchema.(type) {
	case string:
		if err := json.Unmarshal([]byte(t), &schemaMap); err != nil {
			return "", fmt.Errorf("failed to parse component schema in type of json string: %v", err)
		}
	case map[string]any:
		schemaMap = providedSchema.(map[string]any)
	default:
		return "", fmt.Errorf("compoSchema can only support types of json string and map[string]any")
	}

	// validation and set default value
	var err error
	var schema map[string]any
	switch compoType {
	case Restful:
		schema, err = restfulcomponent.NewRestfulComponent(schemaMap)
		if err != nil {
			return "", fmt.Errorf("failed to build restful component schema: %v", err)
		}
		// TODO: implement other cases
	default:
		return "", fmt.Errorf("%s is not a support component type", compoType)
	}

	// add schema to repository
	ctx := context.Background()
	ID, err := c.repo.Create(ctx, "composchema", schema)
	if err != nil {
		return "", fmt.Errorf("failed to record component schema in repository: %v", err)
	}

	// active component
	if err := c.activateComponent(ID); err != nil {
		return "", fmt.Errorf("failed to active component: %v", err)
	}
	return ID, nil
}

// GetComponent gets a component interface through cache or deserializing from repository record.
func (c *ComponentManager) GetComponent(ID string) (componentinterface.IComponent, error) {
	// get component if it is active
	if val, loaded := c.componentCache.Load(ID); loaded {
		return val.(componentinterface.IComponent), nil
	}

	if err := c.activateComponent(ID); err != nil {
		return nil, fmt.Errorf("failed to get component in repository: %v", err)
	} else {
		return c.GetComponent(ID)
	}
}

// DeleteComponent deletes cache and repository record from the provided component
func (c *ComponentManager) DeleteComponent(ID string) error {
	// get component
	_, err := c.GetComponent(ID)
	if err != nil {
		return fmt.Errorf("failed to get component: %v", err)
	}

	// deactivate
	c.deactivateComponent(ID)

	// delete component record in repository
	ctx := context.Background()
	if err := c.repo.Delete(ctx, "composchema", map[string]any{"_id": ID}); err != nil {
		return fmt.Errorf("failed to delete component record: %v", err)
	}

	return nil
}

// Shrink clear the cache to half its size.
func (c *ComponentManager) Shrink() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.shrinkLocked()
}

// GetActiveComponentNum counts all active components in the cache.
func (c *ComponentManager) GetActiveComponentNum() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.heap.Len()
}

// GetComponentRecordNum counts all component records in the repository.
func (c *ComponentManager) GetComponentRecordNum() (int64, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	ctx := context.Background()
	if count, err := c.repo.Count(ctx, "node", nil); err != nil {
		return 0, fmt.Errorf("failed to count node record in repository: %v", err)
	} else {
		return count, nil
	}
}

// activateComponent activates a component from repository record to the runtime cache.
func (c *ComponentManager) activateComponent(ID string) error {
	// check if is active
	if _, loaded := c.componentCache.LoadOrStore(ID, nil); loaded {
		return nil
	}

	// find if is in repository
	var err error
	ctx := context.Background()
	schema, err := c.repo.ReadOne(ctx, "composchema", map[string]any{"_id": ID})
	if err != nil {
		c.componentCache.Delete(ID)
		return fmt.Errorf("cannot activate component not existing: %v", err)
	}

	// get component type
	var compoType ComponentType
	if infos := strings.Split(ID, "-"); len(infos) != 6 {
		return fmt.Errorf("provided ID %s is not valid", ID)
	} else {
		compoType = ComponentType(infos[0])
	}

	// activate component
	var compo componentinterface.IComponent
	switch compoType {
	case Restful:
		compo, err = restfulcomponent.NewRestfulComponentInstance(schema)
		if err != nil {
			return fmt.Errorf("cannot instantiate RESTful component from ID %v: %v", ID, err)
		}
		// TODO: implement other cases
	default:
		return fmt.Errorf("cannot instantiate component from an unknown type: %v", compoType)
	}
	c.componentCache.Store(ID, compo)
	return c.addToHeap(compo)
}

// deactivateComponent deactivates a node from the runtime cache.
func (c *ComponentManager) deactivateComponent(ID string) {
	// check if is inactive
	val, loaded := c.componentCache.LoadAndDelete(ID)
	if !loaded || val == nil {
		return
	}
	compo := val.(componentinterface.IComponent)

	// remove from heap
	c.removeFromHeap(compo)
}

func (c *ComponentManager) shrinkLocked() error {
	toSize := c.cacheSize / 2
	if toSize == 0 {
		toSize = 1
	}

	if c.heap.Len() <= c.cacheSize {
		return nil
	}

	heap.Init(&c.heap)
	for c.heap.Len() > toSize {

		// remove from heap
		entry := heap.Pop(&c.heap).(*componentEntry)

		// deactivate compo
		compo := entry.compo
		ID := compo.GetID()
		c.componentCache.Delete(ID)
	}
	return nil
}

func (c *ComponentManager) addToHeap(component componentinterface.IComponent) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	entry := &componentEntry{
		id:    component.GetID(),
		compo: component,
	}
	heap.Push(&c.heap, entry)
	return c.shrinkLocked()
}

func (c *ComponentManager) removeFromHeap(component componentinterface.IComponent) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for i, entry := range c.heap {
		if entry.compo == component {
			heap.Remove(&c.heap, i)
			break
		}
	}
}

func (h componentHeap) Len() int { return len(h) }
func (h componentHeap) Less(i, j int) bool {
	return h[i].compo.GetCallTime().Before(h[j].compo.GetCallTime())
}
func (h componentHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
	h[i].index = i
	h[j].index = j
}
func (h *componentHeap) Push(x any) {
	entry := x.(*componentEntry)
	entry.index = len(*h)
	*h = append(*h, entry)
}
func (h *componentHeap) Pop() any {
	old := *h
	n := len(old)
	entry := old[n-1]
	old[n-1] = nil
	entry.index = -1
	*h = old[0 : n-1]
	return entry
}
