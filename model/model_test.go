package db

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/spf13/viper"
	db "github.com/world-in-progress/yggdrasil/db/mongo"
)

var mongoDoc = map[string]any{
	"_id":  uuid.New().String(),
	"name": "Mongo Document",
}

var addAPI = map[string]any{
	"API":     "localhost:8000/api/v0/add",
	"method":  "POST",
	"reqDesc": "Request body schema: application/json. Example: { 'a': 1, 'b': 2 }",
	"resDesc": "Response schema: application/json. Example: {'result': 3}",
}

var addComp = map[string]any{
	"_id":  uuid.New().String(),
	"name": "Add Component",
	"rest": addAPI,
}

var parentNode = map[string]any{
	"_id":  uuid.New().String(),
	"name": "Parent Node",
}

var childNode = map[string]any{
	"_id":        uuid.New().String(),
	"name":       "Child Node",
	"parent":     parentNode["_id"],
	"components": []any{addComp},
}

func TestModel(t *testing.T) {
	viper.SetConfigName("test_config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("../test")
	viper.ReadInConfig()

	// init model manager
	modelMgr, err := NewModelManager()
	if err != nil {
		t.Error(err)
	}

	// init MongoDB
	repository := db.NewMongoRepository()

	// test MongoDocument
	if mongoDocData, err := modelMgr.ToBSON("MongoDocument", mongoDoc); err != nil {
		t.Error(err)
	} else {
		fmt.Println("Mongo Docunment BSON:", mongoDocData)
	}

	// test RestfulCalling
	if addAPIData, err := modelMgr.ToBSON("RestfulCalling", addAPI); err != nil {
		t.Error(err)
	} else {
		fmt.Println("Restful Calling BSON:", addAPIData)
	}

	// test Component
	if addCompData, err := modelMgr.ToBSON("Component", addComp); err != nil {
		t.Error(err)
	} else {
		fmt.Println("Component BSON:", addCompData)
		if mongoID, err := repository.Create(context.Background(), "Component", addCompData); err != nil {
			t.Error(err)
		} else {
			fmt.Printf("Instert component %s\n", mongoID)
		}
	}

	// test Node
	if parentNodeData, err := modelMgr.ToBSON("Node", parentNode); err != nil {
		t.Error(err)
	} else {
		fmt.Println("Parent Node BSON:", parentNodeData)
		if mongoID, err := repository.Create(context.Background(), "Node", parentNodeData); err != nil {
			t.Error(err)
		} else {
			fmt.Printf("Instert node %s\n", mongoID)
		}
	}

	if childNodeData, err := modelMgr.ToBSON("Node", childNode); err != nil {
		t.Error(err)
	} else {
		fmt.Println("Child Node BSON:", childNodeData)
		if mongoID, err := repository.Create(context.Background(), "Node", childNodeData); err != nil {
			t.Error(err)
		} else {
			fmt.Printf("Instert node %s\n", mongoID)
		}
	}
}
