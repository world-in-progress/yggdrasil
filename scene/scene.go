package scene

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/world-in-progress/yggdrasil/component"
	"github.com/world-in-progress/yggdrasil/component/restfulcomponent"
	"github.com/world-in-progress/yggdrasil/core/logger"
	"github.com/world-in-progress/yggdrasil/core/threading"
	"github.com/world-in-progress/yggdrasil/db/mongo"
	"github.com/world-in-progress/yggdrasil/model"
	"github.com/world-in-progress/yggdrasil/node"
)

type (
	IComponent interface {
		GetID() string
		Execute(node component.INode, params map[string]any) component.ITask
		Serialize() map[string]any
	}

	IRepository interface {
		Create(ctx context.Context, table string, record any) (string, error)
		Read(ctx context.Context, table string, filter any) (map[string]any, error)
		Update(ctx context.Context, table string, filter any, update any) error
		Delete(ctx context.Context, table string, filter any) error
	}

	SceneManager struct {
		Dispatcher     *threading.WorkerPool
		Modeler        *model.ModelManager
		NodeCache      map[string]component.INode
		ComponentCache map[string]IComponent
		Repository     *mongo.MongoRepository
	}
)

func NewSceneManager(spawnWorkerNum int, maxWorkerNum int, bufferSize int) *SceneManager {
	sm := &SceneManager{
		Dispatcher:     threading.NewWorkerPool(spawnWorkerNum, maxWorkerNum, bufferSize),
		NodeCache:      make(map[string]component.INode),
		ComponentCache: make(map[string]IComponent),
		Repository:     mongo.NewMongoRepository(),
	}

	var err error
	sm.Modeler, err = model.NewModelManager()
	if err != nil {
		logger.Error("Failed to create model manager for scene manager: %v", err)
	}
	return sm
}

func (sm *SceneManager) RegisterComponent(componentInfo map[string]any) (string, error) {
	// check validation
	if err := sm.Modeler.Validate("Component", componentInfo); err != nil {
		return "", fmt.Errorf("componentInfo %v provided for registration is invalid: %v", componentInfo, err)
	}

	// TODO: check if component is existed
	// TODO: check if cache is full
	// TODO: clean cache if needed

	// instantiate the component
	var c IComponent
	if _, ok := componentInfo["rest"]; ok {
		restC := restfulcomponent.NewRestfulComponent(componentInfo)
		restC.Attributes["_id"] = uuid.New().String()
		c = restC
	}

	// insert component to repository
	if _, err := sm.Repository.Create(context.Background(), "component", c.Serialize()); err != nil {
		return "", fmt.Errorf("failed to create node %v: %v", componentInfo, err)
	}

	// add component to cache
	sm.ComponentCache[c.GetID()] = c
	return c.GetID(), nil
}

func (sm *SceneManager) RegisterNode(nodeInfo map[string]any) (string, error) {
	// check validation
	if err := sm.Modeler.Validate("Node", nodeInfo); err != nil {
		return "", fmt.Errorf("nodeInfo %v provided for registration is invalid: %v", nodeInfo, err)
	}

	// TODO: check if node is existed
	// TODO: check if cache is full
	// TODO: clean cache if needed

	// instantiate the node
	n := node.NewNode(nodeInfo)
	// n.Attributes["_id"] = uuid.New().String()

	// insert node to repository
	if _, err := sm.Repository.Create(context.Background(), "node", n.Serialize()); err != nil {
		return "", fmt.Errorf("failed to create node %v: %v", nodeInfo, err)
	}

	// add node to cache
	sm.NodeCache[n.GetID()] = n
	return n.GetID(), nil
}
