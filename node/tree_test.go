package node

import (
	"fmt"
	"testing"
	"time"

	"github.com/spf13/viper"
	"github.com/world-in-progress/yggdrasil/db/mongo"
	"github.com/world-in-progress/yggdrasil/node/model"
)

// instance of model BaseNode
var testNode = map[string]any{
	"name": "Test Node",
}

// instance of model ExtendNode
var testChildNode = map[string]any{
	"name": "Test Child Node",
	"time": time.Now().String(),
}

// instance of model ExtendNode
var testChildChildNode = map[string]any{
	"name": "Test Child-Child Node",
	"time": time.Now().String(),
}

func TestTree(t *testing.T) {
	viper.SetConfigName("test_config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("../test")

	// init tree and its dependencies
	var err error
	var cacheSize uint = 1 // only one node can be stored in the runtime cache
	repo := mongo.NewMongoRepository()
	modeler, err := model.NewModelManager()
	if err != nil {
		t.Fatalf("Failed to create model manager for Tree: %v", err)
	}
	tree := NewTree("Test tree", repo, modeler, cacheSize)

	// start test
	var nodeID string
	var childID string

	// register test node
	if nodeID, err = tree.RegisterNode("BaseNode", testNode); err != nil {
		t.Fatalf("%v", err)
	}

	// update name of test node
	tree.UpdateNodeAttribute(nodeID, "name", "Test Node!")

	// register test child node
	// cache size reaches the shrinking point
	// and test node will be auto-deactivated
	// for its last calltime is earlier than test child node
	testChildNode["parent"] = nodeID
	if childID, err = tree.RegisterNode("ExtendNode", testChildNode); err != nil {
		t.Fatalf("%v", err)
	}

	// deactivate test child node
	if err := tree.deactivateNode(childID); err != nil {
		t.Fatalf("cannot deactivate node: %v", err)
	}

	// check if no active node
	if tree.GetActiveNodeNum() != 0 {
		t.Fatalf("nodes should all be cleaned but not")
	}

	// register test child-child ndoe
	testChildChildNode["parent"] = childID
	if _, err = tree.RegisterNode("ExtendNode", testChildChildNode); err != nil {
		t.Fatalf("%v", err)
	}

	// check node record num (expected 3)
	if recordNum, err := tree.GetNodeRecordNum(); err != nil {
		t.Fatalf("%v", err)
	} else if recordNum != 3 {
		t.Fatalf("node record num is expected to be 3, but is %d", recordNum)
	}

	// get test node (test node will be auto-activated)
	if node, err := tree.GetNode(childID); err != nil {
		t.Fatalf("%v", err)
	} else {
		fmt.Printf("node children IDs are %v: ", node.GetChildIDs())
	}

	// delete test node (test child node and test child-child node will be recursively deleted)
	if err := tree.DeleteNode(nodeID); err != nil {
		t.Fatalf("%v", err)
	}

	// check if no active node
	if tree.GetActiveNodeNum() != 0 {
		t.Fatalf("nodes should all be deleted but not")
	}

	// check if all node records are deleted
	if recordNum, err := tree.GetNodeRecordNum(); err != nil {
		t.Fatalf("%v", err)
	} else if recordNum != 0 {
		t.Fatalf("nodes should all be deleted but not")
	}
}
