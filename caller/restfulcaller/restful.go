package restfulcaller

import (
	"encoding/json"
	"fmt"

	"github.com/world-in-progress/yggdrasil/caller"
	"github.com/world-in-progress/yggdrasil/core/logger"
	"go.mongodb.org/mongo-driver/bson"
)

type (
	ParamDescription struct {
		Param string
		Desc  string
	}
	RestfulCalling struct {
		API       string `bson:"api"`
		Method    string
		ReqSchema string
		ResSchema string
		ReqParams []ParamDescription
		ResParams []ParamDescription
	}
)

func NewRestfulCalling(attributes map[string]any) *RestfulCalling {
	if calling, err := convertToStruct[RestfulCalling](attributes); err != nil {
		logger.Error("Failed to create restful calling from attributes: %v", err)
		return nil
	} else {
		return &calling
	}
}

func (c *RestfulCalling) Call(node caller.INode, params map[string]any) (caller.ITask, error) {
	var task caller.ITask
	var err error

	switch c.Method {
	case "POST":
		task, err = c.processPostCall(node, params)
	}
	return task, err
}

func (c *RestfulCalling) processPostCall(node caller.INode, params map[string]any) (caller.ITask, error) {
	var bodyData = make(map[string]any)

	// First process: embedding params stored in node to bodyData.
	for _, paramDesc := range c.ReqParams {
		if nodeParam, err := node.GetParam(paramDesc.Param); err != nil {
			bodyData[paramDesc.Param] = nodeParam
		}
	}

	// Second process: embedding provided params to bodyData.
	// DISMISSING provided params not declared in ReqParams.
	// DISMISSING param declared in ReqParams but not in privided params.
	// OVERWRITING node params is possible.
	for _, paramDesc := range c.ReqParams {
		if param, ok := params[paramDesc.Param]; ok {
			bodyData[paramDesc.Param] = param
		}
	}

	// Third process: convert bodyData into json body.
	jsonData, err := json.Marshal(bodyData)
	if err != nil {
		return nil, fmt.Errorf("error marshaling POST json body when processPostCall: %v", err)
	}

	// Fourth process: make post task.
	task := NewRestPostTask(c, jsonData)

	return task, nil
}

func convertToStruct[T any](source any) (T, error) {
	var result T

	bytes, err := bson.Marshal(source)
	if err != nil {
		return result, fmt.Errorf("marshal error: %v", err)
	}

	err = bson.Unmarshal(bytes, &result)
	if err != nil {
		return result, fmt.Errorf("unmarshal error: %v", err)
	}

	return result, nil
}
