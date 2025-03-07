package node

import (
	"fmt"
	"testing"

	"github.com/spf13/viper"
	"github.com/world-in-progress/yggdrasil/db/mongo"
)

var testNode = map[string]any{
	"name": "Test Node",
}

var testChildNode = map[string]any{
	"name": "Test Child Node",
}

func TestTree(t *testing.T) {
	viper.SetConfigName("test_config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("../test")

	var cacheSize uint = 1 // only one node can be stored in the runtime cache
	repo := mongo.NewMongoRepository()
	tree := NewTree("Test tree", repo, cacheSize)

	var err error
	var nodeID string
	var childID string

	// register test node
	if nodeID, err = tree.RegisterNode(testNode); err != nil {
		t.Errorf("%v", err)
	}

	// update name of test node
	tree.UpdateNodeAttribute(nodeID, "name", "Test Node!")

	// register test child node
	// cache size reaches the shrinking point
	// and test node will be auto-deactivated
	// for its last calltime is earlier than test child node
	testChildNode["parent"] = nodeID
	if childID, err = tree.RegisterNode(testChildNode); err != nil {
		t.Errorf("%v", err)
	}

	// deactivate test child node
	if err := tree.DeactivateNode(childID); err != nil {
		t.Errorf("cannot deactivate node: %v", err)
	}

	// check if no active node
	if tree.GetActiveNodeNum() != 0 {
		t.Errorf("nodes should all be cleaned but not")
	}

	// get test node (test node will be auto-activated)
	if node, err := tree.GetNode(nodeID); err != nil {
		t.Errorf("%v", err)
	} else {
		fmt.Printf("node children IDs are %v: ", node.GetChildIDs())
	}
}
