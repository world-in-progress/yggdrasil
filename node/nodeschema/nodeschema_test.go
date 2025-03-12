package nodeschema

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/spf13/viper"
	"github.com/world-in-progress/yggdrasil/db/mongo"
)

var mongoDoc = map[string]any{
	"_id":  uuid.New().String(),
	"name": "Mongo Document",
}

var parentNode = map[string]any{
	"_id":  uuid.New().String(),
	"name": "Parent Node",
}

var childNode = map[string]any{
	"_id":    uuid.New().String(),
	"name":   "Child Node",
	"parent": parentNode["_id"],
}

func TestNodeSchema(t *testing.T) {
	viper.SetConfigName("test_config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("../../test")
	viper.ReadInConfig()

	// read node schema from json
	file, err := os.Open("../node_schema_test.json")
	if err != nil {
		t.Fatalf("error opening file: %v", err)
	}
	defer file.Close()

	var nodeSchemas map[string]any
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&nodeSchemas)
	if err != nil {
		t.Fatalf("error decoding json: %v", err)
	}

	// init schema manager
	repo := mongo.NewMongoRepository()
	schemaMgr := NewSchemaManager(repo)

	// register schemas
	schemasRaw, ok := nodeSchemas["schemas"]
	if !ok {
		t.Fatal("error parse schemas")
	}
	fmt.Printf("\n\n\nschemas are: %+v\n\n\n", schemasRaw)

	for _, schema := range schemasRaw.([]any) {
		if err = schemaMgr.RegisterSchema(schema.(map[string]any)); err != nil {
			t.Errorf("%v", err)
		}
	}

	// test MongoDocument
	if err := schemaMgr.Validate("MongoDocument", mongoDoc); err != nil {
		t.Fatalf("%v", err)
	}

	// test Node
	if err := schemaMgr.Validate("BaseNode", parentNode); err != nil {
		t.Fatalf("%v", err)
	}

	if err := schemaMgr.Validate("BaseNode", childNode); err != nil {
		t.Fatalf("%v", err)
	}
}
