package nodemodel

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
	"components": []any{addComp["_id"]},
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
	if err := modelMgr.Validate("MongoDocument", mongoDoc); err != nil {
		t.Error(err)
	} else {
		fmt.Println("Mongo Docunment BSON:", mongoDoc)
	}

	// test RestfulCalling
	if err := modelMgr.Validate("RestfulCalling", addAPI); err != nil {
		t.Error(err)
	}

	// test Component
	if err := modelMgr.Validate("Component", addComp); err != nil {
		t.Error(err)
	} else {
		if mongoID, err := repository.Create(context.Background(), "component", addComp); err != nil {
			t.Error(err)
		} else {
			fmt.Printf("Instert component %s\n", mongoID)
		}
	}

	// test Node
	if err := modelMgr.Validate("Node", parentNode); err != nil {
		t.Error(err)
	} else {
		if mongoID, err := repository.Create(context.Background(), "node", parentNode); err != nil {
			t.Error(err)
		} else {
			fmt.Printf("Instert node %s\n", mongoID)
		}
	}

	if err := modelMgr.Validate("Node", childNode); err != nil {
		t.Error(err)
	} else {
		if mongoID, err := repository.Create(context.Background(), "node", childNode); err != nil {
			t.Error(err)
		} else {
			fmt.Printf("Instert node %s\n", mongoID)
		}
	}
}
