package restfulcomponent

import (
	"github.com/world-in-progress/yggdrasil/caller/restfulcaller"
	"github.com/world-in-progress/yggdrasil/component"
	"github.com/world-in-progress/yggdrasil/core/logger"
)

type (
	RestfulComponent struct {
		Attributes map[string]any
	}
)

func NewRestfulComponent(attributes map[string]any) *RestfulComponent {
	c := &RestfulComponent{
		Attributes: attributes,
	}
	return c
}

func (c *RestfulComponent) GetID() string {
	return c.Attributes["_id"].(string)
}

func (c *RestfulComponent) Execute(node component.INode, params map[string]any) component.ITask {
	calling := restfulcaller.NewRestfulCalling(c.Attributes)
	task, err := calling.Call(node, params)
	if err != nil {
		logger.Error("Failed to generate task from restful component: %s", err.Error())
	}
	return task
}

func (c *RestfulComponent) Serialize() map[string]any {
	return c.Attributes
}
