package db

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/spf13/viper"
	"github.com/world-in-progress/yggdrasil/logger"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type TestNode struct {
	ID   string `bson:"_id"`
	Name string `bson:"name"`
}

func TestMongo(t *testing.T) {
	viper.SetConfigName("mongo_test_config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")

	repo := NewMongoRepository()

	node := TestNode{
		ID:   uuid.New().String(),
		Name: "Test Node",
	}
	id, err := repo.Create(context.Background(), "nodes", node)
	if err != nil {
		t.Errorf("Insert node failed: %v", err)
		return
	}
	logger.Info("Insert node, ID: %s", id)

	result, err := repo.Read(context.Background(), "nodes", bson.M{"_id": node.ID})
	if err != nil {
		t.Errorf("Find node failed: %v", err)
		return
	}
	if result != nil {
		logger.Info("Find node: %+v", result)
	}

	update := bson.M{"$set": bson.M{"name": "hello"}}
	if err = repo.Update(context.Background(), "nodes", bson.M{"_id": node.ID}, update); err != nil {
		t.Errorf("Update node failed: %v", err)
	}
}

func TestTransaction(t *testing.T) {
	repo := NewMongoRepository().(*MongoRepository)

	trans := func(sessionCtx mongo.SessionContext) error {
		node := TestNode{ID: uuid.New().String(), Name: "Subscene node"}
		_, err := repo.Create(sessionCtx, "nodes", node)
		if err != nil {
			return err
		}

		err = repo.Update(sessionCtx, "nodes", bson.M{"_id": node.ID}, bson.M{"$set": bson.M{"name": "transaction test"}})
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
