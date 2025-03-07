package mongo

import (
	"context"
	"fmt"
	"time"

	"github.com/world-in-progress/yggdrasil/core/logger"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type MongoRepository struct {
	db *mongo.Database
}

func NewMongoRepository() *MongoRepository {
	client := GetMongoClient()
	return &MongoRepository{db: client.Database}
}

func (r *MongoRepository) Create(ctx context.Context, table string, record map[string]any) (string, error) {

	coll := r.db.Collection(table)
	ctx, cancel := context.WithTimeout(ctx, time.Duration(GetMongoClient().Config.Timeout)*time.Second)
	defer cancel()

	res, err := coll.InsertOne(ctx, bson.M(record))
	if err != nil {
		logger.Error("Insert failed: %v", err)
		return "", err
	}
	return res.InsertedID.(string), nil
}

func (r *MongoRepository) ReadAll(ctx context.Context, table string, filter map[string]any) ([]map[string]any, error) {
	coll := r.db.Collection(table)
	ctx, cancel := context.WithTimeout(ctx, time.Duration(GetMongoClient().Config.Timeout)*time.Second)
	defer cancel()

	cursor, err := coll.Find(ctx, bson.M(filter))
	if err != nil {
		logger.Error("Query failed: %v", err)
		return nil, err
	}
	defer cursor.Close(ctx)

	var results []map[string]any
	err = cursor.All(ctx, &results)
	if err != nil {
		logger.Error("Failed to decode results: %v", err)
		return nil, err
	}

	if len(results) == 0 {
		return results, nil
	}

	return results, nil
}

func (r *MongoRepository) ReadOne(ctx context.Context, table string, filter map[string]any) (map[string]any, error) {
	coll := r.db.Collection(table)
	ctx, cancel := context.WithTimeout(ctx, time.Duration(GetMongoClient().Config.Timeout)*time.Second)
	defer cancel()

	var result map[string]any
	err := coll.FindOne(ctx, bson.M(filter)).Decode(&result)
	if err == mongo.ErrNoDocuments {
		return nil, mongo.ErrNoDocuments
	}
	if err != nil {
		logger.Error("Query failed: %v", err)
		return nil, err
	}
	return result, nil
}

func (r *MongoRepository) Update(ctx context.Context, table string, filter map[string]any, update map[string]any) error {
	coll := r.db.Collection(table)
	ctx, cancel := context.WithTimeout(ctx, time.Duration(GetMongoClient().Config.Timeout)*time.Second)
	defer cancel()

	_, err := coll.UpdateOne(ctx, bson.M(filter), update)
	if err != nil {
		logger.Error("Update failed: %v", err)
		return err
	}
	return nil
}

func (r *MongoRepository) Delete(ctx context.Context, table string, filter map[string]any) error {
	coll := r.db.Collection(table)
	ctx, cancel := context.WithTimeout(ctx, time.Duration(GetMongoClient().Config.Timeout)*time.Second)
	defer cancel()

	_, err := coll.DeleteOne(ctx, bson.M(filter))
	if err != nil {
		logger.Error("Delete failed: %v", err)
		return err
	}
	return nil
}

func (r *MongoRepository) EnsureIndexes(ctx context.Context, table string, indexes []mongo.IndexModel) error {
	coll := r.db.Collection(table)
	ctx, cancel := context.WithTimeout(ctx, time.Duration(GetMongoClient().Config.Timeout)*time.Second)
	defer cancel()

	indexView := coll.Indexes()
	_, err := indexView.CreateMany(ctx, indexes)
	if err != nil {
		logger.Error("Index creation failed: %v", err)
		return err
	}
	logger.Info("Index created successfully for collection %s", table)
	return nil
}

func (r *MongoRepository) WithTransaction(ctx context.Context, fn func(sessionContext mongo.SessionContext) error) error {
	client := GetMongoClient().Client
	session, err := client.StartSession()
	if err != nil {
		logger.Error("Failed to launch session: %v", err)
		return err
	}
	defer session.EndSession(ctx)

	err = session.StartTransaction()
	if err != nil {
		logger.Error("Failed to start transaction: %v", err)
	}

	err = mongo.WithSession(ctx, session, func(sessionContext mongo.SessionContext) error {
		if err := fn(sessionContext); err != nil {
			session.AbortTransaction(sessionContext)
			logger.Error("Failed to execute transaction: %v", err)
			return err
		}
		return session.CommitTransaction(sessionContext)
	})

	if err != nil {
		return err
	}
	logger.Info("Transaction executed successfully")
	return nil
}

func ConvertToStruct[T any](source any) (T, error) {
	var result T

	bytes, err := bson.Marshal(source)
	if err != nil {
		return result, fmt.Errorf("marshal error: %v", err)
	}

	err = bson.Unmarshal(bytes, &result)
	if err != nil {
		return result, fmt.Errorf("unmarshal error: %v", err)
	}

	return result, nil
}
