package scene

import (
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"testing"

	"github.com/spf13/viper"
	"github.com/world-in-progress/yggdrasil/component"
)

// NOTE
// this test dependents on FastAPI (https://github.com/fastapi/fastapi)
// come to current directory and
// run the following command to launch the python test server:
// fastapi dev server_test.py
func TestScene(t *testing.T) {
	viper.SetConfigName("test_config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("../test")

	// read component schema from json
	file, err := os.Open("component_schema_test.json")
	if err != nil {
		t.Fatalf("error opening file: %v", err)
	}
	defer file.Close()

	var compoSchema map[string]any
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&compoSchema)
	if err != nil {
		t.Fatalf("error decoding json: %v", err)
	}

	fmt.Printf("component schema is: %+v\n\n\n", compoSchema)

	// init scene
	minWorkerNum := runtime.NumCPU()
	maxWorkerNum := runtime.NumCPU() * 100
	bufferSize := runtime.NumCPU() * 1000
	cacheSize := runtime.NumCPU() * 1000
	scene, err := NewScene("Test scene", minWorkerNum, maxWorkerNum, bufferSize, int(cacheSize))
	if err != nil {
		t.Fatal(err)
	}

	// register node schemas
	if _, err = scene.Tree.RegistserNodeSchemaFromJson("node_schema_test.json"); err != nil {
		t.Fatal(err)
	}

	// register component
	compoID, err := scene.RegisterComponent(component.Restful, compoSchema)
	if err != nil {
		t.Fatalf("failed to register componnet: %v\n\n\n", err)
	}

	// register node
	nodeInfo := map[string]any{
		"name":   "Test Node",
		"result": 0.0,
	}
	nodeID, err := scene.RegisterNode("SumNode", nodeInfo)
	if err != nil {
		t.Fatalf("failed to register node: %vn\n\n", err)
	}

	// bind componnet to node
	err = scene.BindComponentToNode(nodeID, compoID)
	if err != nil {
		t.Fatalf("failed to bind component to node: %v\n\n\n", err)
	}

	// invoke node component
	testParams := map[string]any{
		"a": 0.1,
		"b": 1.0,
	}
	syncTask, err := scene.InvokeNodeComponent(string(Sync), nodeID, compoID, testParams, nil)
	if err != nil {
		t.Fatalf("failed to invoke node component: %v\n\n\n", err)
	}
	// syncing task
	result, err := syncTask.(*SyncTask).Syncing()
	if err != nil {
		t.Fatalf("error happend when syncing task: %vn\n\n", err)
	}
	fmt.Printf("\n\n\nresult invoked from componnet %v of node %v is: %v\n\n\n", compoID, nodeID, result)

	// check if node attribute has been updated
	node, err := scene.GetNode(nodeID)
	if err != nil {
		t.Fatalf("err: %v\n\n\n", err)
	}
	nodeResult := node.GetParam("result")
	if nodeResult != 1.1 {
		t.Fatalf("node attribute about result is expected to be 1.1, but is %v", nodeResult)
	}
	fmt.Printf("updated node result is: %v\n\n\n", node.GetParam("result"))

	// delete node and component
	if err = scene.DeleteNode(nodeID); err != nil {
		t.Fatalf("failed to delete node: %v\n\n\n", err)
	}
	if err = scene.DeleteComponent(compoID); err != nil {
		t.Fatalf("failed to delete component: %v\n\n\n", err)
	}
}
