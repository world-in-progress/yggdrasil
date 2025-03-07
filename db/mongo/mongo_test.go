package mongo

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/spf13/viper"
	"github.com/world-in-progress/yggdrasil/core/logger"
)

func TestMongo(t *testing.T) {
	viper.SetConfigName("test_config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("../../test")

	repo := NewMongoRepository()

	node := map[string]any{
		"_id":  uuid.New().String(),
		"name": "Test Node",
	}

	id, err := repo.Create(context.Background(), "nodes", node)
	if err != nil {
		t.Errorf("Insert node failed: %v", err)
		return
	}
	logger.Info("Insert node, ID: %s", id)

	result, err := repo.ReadOne(context.Background(), "nodes", map[string]any{"_id": node["_id"]})
	if err != nil {
		t.Errorf("Find node failed: %v", err)
		return
	}
	if result != nil {
		logger.Info("Find node: %+v", result)
	}

	update := map[string]any{"set": map[string]any{"name": "Hello!"}}
	if err = repo.Update(context.Background(), "nodes", map[string]any{"_id": node["_id"]}, update); err != nil {
		t.Errorf("Update node failed: %v", err)
	}
}
