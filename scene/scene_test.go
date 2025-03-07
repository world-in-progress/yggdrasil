package scene

import (
	"fmt"
	"testing"

	"github.com/spf13/viper"
)

var mongoDoc = map[string]any{
	"_id":  "0",
	"name": "Mongo Document",
}

var addAPI = map[string]any{
	"api":       "localhost:8000/api/v0/add",
	"method":    "POST",
	"reqSchema": "application/json",
	"resSchema": "application/json",
	"reqParams": []any{
		map[string]any{
			"param": "a",
			"desc":  "The addend, must be a number",
		},
		map[string]any{
			"param": "b",
			"desc":  "The summand, must be a number",
		},
	},
	"resParams": []any{
		map[string]any{
			"param": "result",
			"desc":  "The result, is a number",
		},
	},
}

var addComp = map[string]any{
	"_id":  "0",
	"name": "Add Component",
	"rest": addAPI,
}

var parentNode = map[string]any{
	"_id":  "0",
	"name": "Parent Node",
}

var childNode = map[string]any{
	"_id":        "0",
	"name":       "Child Node",
	"parent":     parentNode["_id"],
	"components": []any{"0"},
}

func TestSceneManager(t *testing.T) {
	viper.SetConfigName("test_config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("../test")

	sm := NewSceneManager(10, 100, 1000)

	// test register node
	parentID, err := sm.RegisterNode(parentNode)
	if err != nil {
		t.Errorf("Failed to register node %v: %v", parentNode, err)
	} else {
		fmt.Printf("Node cache of manager: %v\n", sm.NodeCache)
	}

	// test register component
	componentID, err := sm.RegisterComponent(addComp)
	if err != nil {
		t.Errorf("Failed to register component %v: %v", addComp, err)
	} else {
		fmt.Printf("Component cache of manager: %v\n", sm.ComponentCache)
	}

	// test register node with component
	childNode["parent"] = parentID
	childNode["components"].([]any)[0] = componentID
	if _, err := sm.RegisterNode(childNode); err != nil {
		t.Errorf("Failed to register node %v: %v", childNode, err)
	} else {
		fmt.Printf("Node cache of manager: %v\n", sm.NodeCache)
	}
}
