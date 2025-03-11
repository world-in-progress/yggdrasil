package component

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/spf13/viper"
	"github.com/world-in-progress/yggdrasil/db/mongo"
)

var restfulCreateNodeComponent = map[string]any{
	"method":      "POST",
	"name":        "Register Node API",
	"description": "Create nodes",
	"api":         "http://127.0.0.1:8000/api/v0/node",
	"reqSchema":   "application/json",
	"reqParams": []any{
		map[string]any{
			"name":        "name",
			"description": "Node name",
			"type":        "string",
			"kind":        "simple",
			"required":    true,
		},
	},
	"resStatuses": []any{
		map[string]any{
			"code":        200,
			"description": "Node created successfully",
			"schema":      "application/json",
			"params": []any{
				map[string]any{
					"name":        "_id",
					"description": "Node ID",
					"type":        "string",
				},
			},
		},
	},
}

// NOTE
// this test dependents on FastAPI (https://github.com/fastapi/fastapi)
// come to current directory and
// run the following command to launch the python test server:
// fastapi dev server_test.py
func TestComponentManager(t *testing.T) {
	viper.SetConfigName("test_config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("../test")

	// init component manager and its dependencies
	var err error
	var cacheSize uint = 1 // only one component can be stored in the runtime cache
	repo := mongo.NewMongoRepository()
	cManager, err := NewComponentManager("Test Component Manager", repo, cacheSize)
	if err != nil {
		t.Fatal(err)
	}

	// register component from schema in type of map
	componentID, err := cManager.RegisterComponent(Restful, restfulCreateNodeComponent)
	if err != nil {
		t.Fatal(err)
	} else {
		fmt.Printf("%v\n\n\n", componentID)
	}

	// check component schema in type of JSON string
	bytes, err := json.Marshal(restfulCreateNodeComponent)
	if err != nil {
		t.Fatalf("Failed to create schema in type of JSON string: %v", err)
	}
	schemaString := string(bytes)
	fmt.Printf("Schema in type of JSON string is: %v\n\n\n", schemaString)

	// register component from json string
	// cache size reaches the shrinking point
	// and component from map schema will be auto-deactivated
	// for its last calltime is earlier than component from json string
	jsonComponentID, err := cManager.RegisterComponent(Restful, schemaString)
	if err != nil {
		t.Fatal(err)
	} else {
		fmt.Printf("%v\n\n\n", jsonComponentID)
	}

	// deactivate component from json string
	cManager.deactivateComponent(jsonComponentID)

	// check if no active node
	if cManager.GetActiveComponentNum() != 0 {
		t.Fatalf("components should all be cleaned but not")
	}

	// get component
	compo, err := cManager.GetComponent(jsonComponentID)
	if err != nil {
		t.Fatalf("failed to get component: %v", err)
	}
	testParams := map[string]any{
		"name": "Root Node",
	}
	// check if component can be used to send request and get response
	result, err := compo.Execute(nil, testParams, nil, nil)
	if err != nil {
		t.Fatalf("error executing node component: %v\n\n\n", err)
	} else {
		fmt.Printf("get response: %+v\n\n\n", result)
	}

	// test delete
	err = cManager.DeleteComponent(componentID)
	if err != nil {
		t.Fatalf("failed to delete component %v: %v\n\n\n", componentID, err)
	}
	err = cManager.DeleteComponent(jsonComponentID)
	if err != nil {
		t.Fatalf("failed to delete component %v: %v\n\n\n", jsonComponentID, err)
	}

	// check if no active node
	if cManager.GetActiveComponentNum() != 0 {
		t.Fatalf("components should all be cleaned but not\n\n\n")
	}
	// check if all node records are deleted
	if recordNum, err := cManager.GetComponentRecordNum(); err != nil {
		t.Fatalf("%v", err)
	} else if recordNum != 0 {
		t.Fatalf("components should all be deleted but not\n\n\n")
	} else {
		fmt.Printf("components have all been deleted\n\n\n")
	}
}
