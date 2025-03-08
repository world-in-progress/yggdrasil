package mongo

import (
	"context"
	"sync"
	"time"

	"github.com/world-in-progress/yggdrasil/core/logger"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type MongoRepository struct {
	db          *mongo.Database
	timeout     time.Duration
	collections sync.Map
}

func NewMongoRepository() *MongoRepository {
	client := GetMongoClient()
	return &MongoRepository{
		db:      client.Database,
		timeout: time.Duration(client.Config.Timeout) * time.Second,
	}
}

func (r *MongoRepository) getCollection(table string) *mongo.Collection {
	if coll, ok := r.collections.Load(table); ok {
		return coll.(*mongo.Collection)
	}
	coll := r.db.Collection(table)
	r.collections.Store(table, coll)
	return coll
}

func (r *MongoRepository) withTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, r.timeout)
}

func (r *MongoRepository) Create(ctx context.Context, table string, record map[string]any) (string, error) {
	coll := r.getCollection(table)
	timeoutCtx, cancel := r.withTimeout(ctx)
	defer cancel()

	res, err := coll.InsertOne(timeoutCtx, bson.M(record))
	if err != nil {
		logger.Error("Insert failed for collection %s: %v", table, err)
		return "", err
	}
	return res.InsertedID.(string), nil
}

func (r *MongoRepository) ReadAll(ctx context.Context, table string, filter map[string]any) ([]map[string]any, error) {
	coll := r.getCollection(table)
	timeoutCtx, cancel := r.withTimeout(ctx)
	defer cancel()

	cursor, err := coll.Find(timeoutCtx, bson.M(filter))
	if err != nil {
		logger.Error("Query failed for collection %s: %v", table, err)
		return nil, err
	}
	defer cursor.Close(timeoutCtx)

	var results []map[string]any
	err = cursor.All(timeoutCtx, &results)
	if err != nil {
		logger.Error("Failed to decode results for collection %s: %v", table, err)
		return nil, err
	}

	if len(results) == 0 {
		return results, nil
	}

	return results, nil
}

func (r *MongoRepository) ReadOne(ctx context.Context, table string, filter map[string]any) (map[string]any, error) {
	coll := r.getCollection(table)
	timeoutCtx, cancel := r.withTimeout(ctx)
	defer cancel()

	var result map[string]any
	err := coll.FindOne(timeoutCtx, bson.M(filter)).Decode(&result)
	if err == mongo.ErrNoDocuments {
		return nil, mongo.ErrNoDocuments
	}
	if err != nil {
		logger.Error("Query failed for collection %s: %v", table, err)
		return nil, err
	}
	return result, nil
}

func (r *MongoRepository) Update(ctx context.Context, table string, filter map[string]any, update map[string]any) error {
	coll := r.getCollection(table)
	timeoutCtx, cancel := r.withTimeout(ctx)
	defer cancel()

	_, err := coll.UpdateOne(timeoutCtx, bson.M(filter), update)
	if err != nil {
		logger.Error("Update failed for collection %s: %v", table, err)
		return err
	}
	return nil
}

func (r *MongoRepository) Delete(ctx context.Context, table string, filter map[string]any) error {
	coll := r.getCollection(table)
	timeoutCtx, cancel := r.withTimeout(ctx)
	defer cancel()

	_, err := coll.DeleteOne(timeoutCtx, bson.M(filter))
	if err != nil {
		logger.Error("Delete failed for collection %s: %v", table, err)
		return err
	}
	return nil
}

func (r *MongoRepository) Count(ctx context.Context, table string, filter map[string]any) (int64, error) {
	coll := r.getCollection(table)
	timeoutCtx, cancel := r.withTimeout(ctx)
	defer cancel()

	var filterDoc any
	if len(filter) == 0 {
		filterDoc = bson.D{}
	} else {
		filterDoc = bson.M(filter)
	}

	count, err := coll.CountDocuments(timeoutCtx, filterDoc)
	if err != nil {
		logger.Error("Count failed for collection %s: %v", table, err)
		return 0, err
	}

	return count, nil
}

func (r *MongoRepository) EnsureIndexes(ctx context.Context, table string, indexes []mongo.IndexModel) error {
	coll := r.getCollection(table)
	timeoutCtx, cancel := r.withTimeout(ctx)
	defer cancel()

	indexView := coll.Indexes()
	_, err := indexView.CreateMany(timeoutCtx, indexes)
	if err != nil {
		logger.Error("Index creation failed for collection %s: %v", table, err)
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
