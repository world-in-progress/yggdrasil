package db

import (
	"fmt"
	"testing"
)

func TestModel(t *testing.T) {
	// init model manager
	modelMgr, err := NewModelManager("demo.model.json")
	if err != nil {
		t.Error(err)
	}

	// test MongoDocument
	mongoDoc := map[string]any{
		"_id":  "0",
		"name": "Mongo Document",
	}
	if mongoDocData, err := modelMgr.ToBSON("MongoDocument", mongoDoc); err != nil {
		t.Error(err)
	} else {
		fmt.Println("Mongo Docunment BSON:", mongoDocData)
	}

	// test RestfulCalling
	addAPI := map[string]any{
		"API":     "localhost:8000/api/v0/add",
		"method":  "POST",
		"reqDesc": "Request body schema: application/json. Example: { 'a': 1, 'b': 2 }",
		"resDesc": "Response schema: application/json. Example: {'result': 3}",
	}
	if addAPIData, err := modelMgr.ToBSON("RestfulCalling", addAPI); err != nil {
		t.Error(err)
	} else {
		fmt.Println("Restful Calling BSON:", addAPIData)
	}

	// test Component
	addComp := map[string]any{
		"_id":  "0",
		"name": "Add Component",
		"rest": addAPI,
	}
	if addCompData, err := modelMgr.ToBSON("Component", addComp); err != nil {
		t.Error(err)
	} else {
		fmt.Println("Component BSON:", addCompData)
	}

	// test Node
	parentNode := map[string]any{
		"_id":  "0",
		"name": "Parent Node",
	}
	if parentNodeData, err := modelMgr.ToBSON("Node", parentNode); err != nil {
		t.Error(err)
	} else {
		fmt.Println("Parent Node BSON:", parentNodeData)
	}

	childNode := map[string]any{
		"_id":        "1",
		"name":       "Child Node",
		"parent":     "0",
		"components": []any{addComp},
	}
	if childNodeData, err := modelMgr.ToBSON("Node", childNode); err != nil {
		t.Error(err)
	} else {
		fmt.Println("Child Node BSON:", childNodeData)
	}
}
