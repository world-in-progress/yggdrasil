package db

import (
	"context"
	"testing"

	"github.com/world-in-progress/yggdrasil/logger"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type TestNode struct {
	ID   string `bson:"_id"`
	Name string `bson:"name"`
}

func TestRepository(t *testing.T) {
	repo := NewMongoRepository()

	node := TestNode{
		ID:   "testNode2",
		Name: "Scene Root",
	}
	id, err := repo.Insert(context.Background(), "nodes", node)
	if err != nil {
		t.Errorf("Insert node failed: %v", err)
		return
	}
	logger.Info("Insert node, ID: %s", id)

	result, err := repo.FindOne(context.Background(), "nodes", bson.M{"_id": "testNode1"})
	if err != nil {
		t.Errorf("Find node failed: %v", err)
		return
	}
	if result != nil {
		logger.Info("Find node: %+v", result)
	}

	update := bson.M{"$set": bson.M{"name": "haha!"}}
	if err = repo.Update(context.Background(), "nodes", bson.M{"_id": "testNode2"}, update); err != nil {
		t.Errorf("Update node failed: %v", err)
	}
}

func TestTransaction(t *testing.T) {
	repo := NewMongoRepository()

	trans := func(sessionCtx mongo.SessionContext) error {
		node := TestNode{ID: "2", Name: "Subscene node"}
		_, err := repo.Insert(sessionCtx, "nodes", node)
		if err != nil {
			return err
		}

		err = repo.Update(sessionCtx, "nodes", bson.M{"_id": "2"}, bson.M{"$set": bson.M{"status": "active"}})
		if err != nil {
			return err
		}
		return nil
	}

	if err := repo.WithTransaction(context.Background(), trans); err != nil {
		logger.Error("Transaction failed: %v", err)
		return
	}
}
