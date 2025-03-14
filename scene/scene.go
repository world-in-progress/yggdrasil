package scene

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/world-in-progress/yggdrasil/component"
	componentinterface "github.com/world-in-progress/yggdrasil/component/interface"
	"github.com/world-in-progress/yggdrasil/core/threading"
	"github.com/world-in-progress/yggdrasil/db/mongo"
	"github.com/world-in-progress/yggdrasil/node"
)

type (
	TaskType string

	// ITask is the interface for a worker task.
	ITask interface {
		GetID() string
		Cancel() bool
		Process()
		Complete()
		IsCanceled() bool
		IsCompleted() bool
		IsIgnoreable() bool
	}

	// NodeTemplate is the structure for a node template.
	NodeTemplate struct {
		ID         string   `json:"_id"`
		Name       string   `json:"name"`
		Schema     string   `json:"schema"`
		Components []string `json:"components"`
	}

	// Scene is the structure for a blackboard.
	Scene struct {
		Name       string
		Dispatcher *threading.WorkerPool
		Repo       *mongo.MongoRepository
		Tree       *node.Tree
		Compos     *component.ComponentManager
	}
)

const (
	Sync   TaskType = "SYNC"
	Async  TaskType = "ASYNC"
	Socket TaskType = "SOCKET"
)

func NewScene(name string, minWorkerNum, maxWorkerNum, bufferSize, cacheSize int) (*Scene, error) {
	s := &Scene{
		Name:       name,
		Repo:       mongo.NewMongoRepository(),
		Dispatcher: threading.NewWorkerPool(minWorkerNum, maxWorkerNum, bufferSize),
	}

	// Create information resource tree
	tree, err := node.NewTree("Tree of "+name, s.Repo, uint(cacheSize))
	if err != nil {
		return nil, fmt.Errorf("failed to create tree for the scene %v: %v", name, err)
	}
	s.Tree = tree

	// Create component manager
	cManager, err := component.NewComponentManager("Component Manager of "+name, s.Repo, uint(cacheSize))
	if err != nil {
		return nil, fmt.Errorf("failed to create component manager for the scene %v: %v", name, err)
	}
	s.Compos = cManager
	return s, nil
}

func (s *Scene) RegisterNode(schemaName string, nodeInfo map[string]any) (string, error) {
	if ID, err := s.Tree.RegisterNode(schemaName, nodeInfo); err != nil {
		return "", fmt.Errorf("scene %v cannot register node %v: %v", s.Name, nodeInfo, err)
	} else {
		return ID, nil
	}
}

func (s *Scene) GetNode(ID string) (*node.Node, error) {
	if node, err := s.Tree.GetNode(ID); err != nil {
		return nil, fmt.Errorf("scene %v cannot get node %v: %v", s.Name, ID, err)
	} else {
		return node, nil
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
	if ID, err := s.Compos.RegisterComponent(compoType, compoSchema); err != nil {
		return "", fmt.Errorf("scene %v cannot register component %v: %v", s.Name, compoSchema, err)
	} else {
		return ID, nil
	}
}

func (s *Scene) GetComponnet(ID string) (componentinterface.IComponent, error) {
	if compo, err := s.Compos.GetComponent(ID); err != nil {
		return nil, fmt.Errorf("scene %v cannot get componnet %v: %v", s.Name, ID, err)
	} else {
		return compo, nil
	}
}

func (s *Scene) DeleteComponent(ID string) error {
	if err := s.Compos.DeleteComponent(ID); err != nil {
		return fmt.Errorf("scene %v cannot delete componnet %v: %v", s.Name, ID, err)
	} else {
		return nil
	}
}

func (s *Scene) RegisterNodeTemplate(templateName string, schemaID string, compoIDs []string) (string, error) {
	// Check if template name has existed
	ctx := context.Background()
	record, err := s.Repo.ReadOne(ctx, "nodetemplate", map[string]any{"name": templateName})
	if err != nil {
		if record == nil {
			return "", fmt.Errorf("error occured when read template by name '%s' in repository: %v", templateName, err)
		}
	} else {
		return record["_id"].(string), nil
	}

	// Check if schema exists
	if !s.Tree.SchemaMgr.HasSchema(schemaID) {
		return "", fmt.Errorf("no schema has ID %s", schemaID)
	}

	// Check if all component exists
	for _, compoID := range compoIDs {
		if _, err := s.Compos.GetComponent(compoID); err != nil {
			return "", fmt.Errorf("no component has ID %s", compoID)
		}
	}

	// Create templateID
	templateID := uuid.New().String()

	// Generate template.
	t := &NodeTemplate{
		ID:         templateID,
		Name:       templateName,
		Schema:     schemaID,
		Components: compoIDs,
	}

	// Store template to repository
	m, err := convertToMap(*t)
	if err != nil {
		return "", fmt.Errorf("failed to convert node template struct (name %s) to map[string]any: %v", templateName, err)
	}
	if _, err = s.Repo.Create(ctx, "nodetemplate", m); err != nil {
		return "", fmt.Errorf("failed to store node template (name %s) to repository: %v", templateName, err)
	}

	return templateID, nil
}

func (s *Scene) GetNodeTemplate(templateID string) (*NodeTemplate, error) {
	// Load template
	ctx := context.Background()
	templateInfo, err := s.Repo.ReadOne(ctx, "nodetemplate", map[string]any{"_id": templateID})
	if err != nil {
		if templateInfo != nil {
			return nil, fmt.Errorf("no template has ID %s", templateID)
		} else {
			return nil, fmt.Errorf("failed to find template hasing ID %s in repository: %v", templateID, err)
		}
	}

	// Make template instance
	if template, err := convertToStruct[*NodeTemplate](templateInfo); err != nil {
		return nil, fmt.Errorf("faild to create template instance (ID: %s): %v", templateID, err)
	} else {
		return template, nil
	}
}

func (s *Scene) DeleteNodeTemplate(templateID string) error {
	// Get template
	_, err := s.GetNodeTemplate(templateID)
	if err != nil {
		return err
	}

	// Delete template from repository
	ctx := context.Background()
	err = s.Repo.Delete(ctx, "nodetemplate", map[string]any{"_id": templateID})
	if err != nil {
		return fmt.Errorf("failed to delete node template (ID: %s) from template: %v", templateID, err)
	}
	return nil
}

func (s *Scene) RegisterNodeFromTemplate(templateID string, nodeInfo map[string]any) (string, error) {
	// Load template
	template, err := s.GetNodeTemplate(templateID)
	if err != nil {
		return "", err
	}

	// Create node
	nodeID, err := s.RegisterNode(template.Schema, nodeInfo)
	if err != nil {
		return "", fmt.Errorf("failed to register node (Info: %v) from template (ID: %s): %v", nodeInfo, templateID, err)
	}

	// Update node attribute about template
	err = s.Tree.UpdateNodeAttribute(nodeID, "template", templateID)
	if err != nil {
		return "", fmt.Errorf("failed to update node (ID: %s) attribute about template (ID: %s): %v", nodeID, templateID, err)
	}

	// Bind all components to node
	for _, compoID := range template.Components {
		err = s.Tree.BindComponentToNode(nodeID, compoID)
		if err != nil {
			return "", fmt.Errorf("failed to bind component (ID: %s) to node (ID: %s): %v", compoID, nodeID, err)
		}
	}

	return nodeID, nil
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

	// Get node
	_, err = s.Tree.GetNode(nodeID)
	if err != nil {
		return fmt.Errorf("failed to get node by ID %v: %v", nodeID, err)
	}

	// Get component
	_, err = s.Compos.GetComponent(compoID)
	if err != nil {
		return fmt.Errorf("failed to get componnet by ID %v: %v", compoID, err)
	}

	// Bind component to node
	err = s.Tree.BindComponentToNode(nodeID, compoID)
	if err != nil {
		return fmt.Errorf("failed to bind component %v to node %v: %v", compoID, nodeID, err)
	}
	return nil
}

func (s *Scene) DeleteComponentFromNode(nodeID, compoID string) error {
	var err error

	// Get node
	_, err = s.Tree.GetNode(nodeID)
	if err != nil {
		return fmt.Errorf("failed to get node by ID %v: %v", nodeID, err)
	}

	// Get component
	_, err = s.Compos.GetComponent(compoID)
	if err != nil {
		return fmt.Errorf("failed to get component by ID %v: %v", compoID, err)
	}

	// Delete component from node
	err = s.Tree.DeleteComponentFromNode(nodeID, compoID)
	if err != nil {
		return fmt.Errorf("failed to delete component %v from node %v: %v", compoID, nodeID, err)
	}
	return nil
}

func (s *Scene) InvokeNodeComponent(taskType string, nodeID, compoID string, params map[string]any, headers map[string]string) (ITask, error) {
	var err error

	if params == nil {
		params = map[string]any{}
	}

	// Get node
	node, err := s.Tree.GetNode(nodeID)
	if err != nil {
		return nil, fmt.Errorf("failed to get node by ID %v: %v", nodeID, err)
	}

	// Get component
	compo, err := s.Compos.GetComponent(compoID)
	if err != nil {
		return nil, fmt.Errorf("failed to get component by ID %v: %v", compoID, err)
	}

	// Build task
	var task ITask
	taskID := uuid.New().String()
	switch taskType {
	case string(Sync):
		task = NewSyncTask(taskID, s.Tree, node, compo, params, headers)
		if _, err := s.Dispatcher.Submit(task); err != nil {
			return nil, fmt.Errorf("failed to submit sync task: %v", err)
		}

	// TODO: implement other task type.
	default:
		return nil, fmt.Errorf("task type %v is not supported", taskType)
	}

	return task, nil
}

func convertToStruct[T any](source any) (T, error) {
	var result T

	bytes, err := json.Marshal(source)
	if err != nil {
		return result, fmt.Errorf("marshal error when transfer source to component: %v", err)
	}

	err = json.Unmarshal(bytes, &result)
	if err != nil {
		return result, fmt.Errorf("unmarshal error source to component: %v", err)
	}

	return result, nil
}

func convertToMap[T any](component T) (map[string]any, error) {
	var result map[string]any

	bytes, err := json.Marshal(component)
	if err != nil {
		return result, fmt.Errorf("marshal error when transfer component to map %v", err)
	}

	err = json.Unmarshal(bytes, &result)
	if err != nil {
		return result, fmt.Errorf("unmarshal error when transfer component to map %v", err)
	}

	return result, nil
}
