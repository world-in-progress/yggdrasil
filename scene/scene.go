package scene

import (
	"fmt"

	"github.com/world-in-progress/yggdrasil/component"
	"github.com/world-in-progress/yggdrasil/core/threading"
	"github.com/world-in-progress/yggdrasil/db/mongo"
	"github.com/world-in-progress/yggdrasil/node"
)

type (
	// Scene is the structure for a blackboard.
	Scene struct {
		Name       string
		Dispatcher *threading.WorkerPool
		Repo       *mongo.MongoRepository
		Tree       *node.Tree
		Components *component.ComponentManager
	}
)

func NewScene(name string, minWorkerNum, maxWorkerNum, bufferSize, cacheSize int) (*Scene, error) {
	s := &Scene{
		Name:       name,
		Repo:       mongo.NewMongoRepository(),
		Dispatcher: threading.NewWorkerPool(minWorkerNum, maxWorkerNum, bufferSize),
	}

	// create information resource tree
	tree, err := node.NewTree("Tree of "+name, s.Repo, uint(cacheSize))
	if err != nil {
		return nil, fmt.Errorf("failed to create tree for the scene %v: %v", name, err)
	}
	s.Tree = tree

	// create component manager
	cManager, err := component.NewComponentManager("Component Manager of "+name, s.Repo, uint(cacheSize))
	if err != nil {
		return nil, fmt.Errorf("failed to create component manager for the scene %v: %v", name, err)
	}
	s.Components = cManager
	return s, nil
}

func (s *Scene) RegisterNode(modelName string, nodeInfo map[string]any) (string, error) {
	if ID, err := s.Tree.RegisterNode(modelName, nodeInfo); err != nil {
		return "", fmt.Errorf("scene %v cannot register node %v: %v", s.Name, nodeInfo, err)
	} else {
		return ID, nil
	}
}

func (s *Scene) DeleteNode(ID string) error {
	if err := s.Tree.DeleteNode(ID); err != nil {
		return fmt.Errorf("scene %v cannot delete node %v: %v", s.Name, ID, err)
	} else {
		return nil
	}
}

func (s *Scene) RegisterComponent(compoType component.ComponentType, compoSchema map[string]any) (string, error) {
	if ID, err := s.Components.RegisterComponent(compoType, compoSchema); err != nil {
		return "", fmt.Errorf("scene %v cannot register component %v: %v", s.Name, compoSchema, err)
	} else {
		return ID, nil
	}
}

func (s *Scene) DeleteComponent(ID string) error {
	if err := s.Components.DeleteComponent(ID); err != nil {
		return fmt.Errorf("scene %v cannot delete componnet %v: %v", s.Name, ID, err)
	} else {
		return nil
	}
}

func (s *Scene) UpdateNodeAttribute(ID string, attributeName string, updateData any) error {
	if err := s.Tree.UpdateNodeAttribute(ID, attributeName, updateData); err != nil {
		return fmt.Errorf("scene %v cannot update attribute about %v of node %v: %v", s.Name, attributeName, ID, err)
	} else {
		return nil
	}
}

func (s *Scene) BindComponentToNode(nodeID, compoID string) error {
	var err error

	// try get node
	_, err = s.Tree.GetNode(nodeID)
	if err != nil {
		return fmt.Errorf("failed to get node by ID %v: %v", nodeID, err)
	}

	// try get component
	_, err = s.Components.GetComponent(compoID)
	if err != nil {
		return fmt.Errorf("failed to get componnet by ID %v: %v", compoID, err)
	}

	// bind component to node
	err = s.Tree.BindComponentToNode(nodeID, compoID)
	if err != nil {
		return fmt.Errorf("failed to bind component %v to node %v: %v", compoID, nodeID, err)
	}
	return nil
}

func (s *Scene) DeleteComponentFromNode(nodeID, compoID string) error {
	var err error

	// try get node
	_, err = s.Tree.GetNode(nodeID)
	if err != nil {
		return fmt.Errorf("failed to get node by ID %v: %v", nodeID, err)
	}

	// try get component
	_, err = s.Components.GetComponent(compoID)
	if err != nil {
		return fmt.Errorf("failed to get component by ID %v: %v", compoID, err)
	}

	// delete component from node
	err = s.Tree.DeleteComponentFromNode(nodeID, compoID)
	if err != nil {
		return fmt.Errorf("failed to delete component %v from node %v: %v", compoID, nodeID, err)
	}
	return nil
}

func (s *Scene) InvokeNodeComponent(nodeID, compoID string, params map[string]any, headers map[string]string) (any, error) {
	var err error

	// try get node
	node, err := s.Tree.GetNode(nodeID)
	if err != nil {
		return nil, fmt.Errorf("failed to get node by ID %v: %v", nodeID, err)
	}

	// try get component
	compo, err := s.Components.GetComponent(compoID)
	if err != nil {
		return nil, fmt.Errorf("failed to get component by ID %v: %v", compoID, err)
	}

	result, err := compo.Execute(node, params, nil, headers)
	if err != nil {
		return nil, fmt.Errorf("error executing component %v of node %v: %v", compo.GetName(), node.GetName(), err)
	}
	return result, nil
}
