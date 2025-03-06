package db

import (
	"context"
	"time"

	"github.com/world-in-progress/yggdrasil/db"
	"github.com/world-in-progress/yggdrasil/logger"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type MongoRepository struct {
	db *mongo.Database
}

func NewMongoRepository() db.Repository {
	client := GetMongoClient()
	return &MongoRepository{db: client.Database}
}

func (r *MongoRepository) Create(ctx context.Context, collection string, document any) (string, error) {
	coll := r.db.Collection(collection)
	ctx, cancel := context.WithTimeout(ctx, time.Duration(GetMongoClient().Config.Timeout)*time.Second)
	defer cancel()

	res, err := coll.InsertOne(ctx, document)
	if err != nil {
		logger.Error("Insert failed: %v", err)
		return "", err
	}
	return res.InsertedID.(string), nil
}

func (r *MongoRepository) Read(ctx context.Context, collection string, filter bson.M) (any, error) {
	coll := r.db.Collection(collection)
	ctx, cancel := context.WithTimeout(ctx, time.Duration(GetMongoClient().Config.Timeout)*time.Second)
	defer cancel()

	var result any
	err := coll.FindOne(ctx, filter).Decode(&result)
	if err == mongo.ErrNoDocuments {
		return nil, mongo.ErrNoDocuments
	}
	if err != nil {
		logger.Error("Query failed: %v", err)
		return nil, err
	}
	return result, nil
}

func (r *MongoRepository) Update(ctx context.Context, collection string, filter bson.M, update bson.M) error {
	coll := r.db.Collection(collection)
	ctx, cancel := context.WithTimeout(ctx, time.Duration(GetMongoClient().Config.Timeout)*time.Second)
	defer cancel()

	_, err := coll.UpdateOne(ctx, filter, update)
	if err != nil {
		logger.Error("Update failed: %v", err)
		return err
	}
	return nil
}

func (r *MongoRepository) Delete(ctx context.Context, collection string, filter bson.M) error {
	coll := r.db.Collection(collection)
	ctx, cancel := context.WithTimeout(ctx, time.Duration(GetMongoClient().Config.Timeout)*time.Second)
	defer cancel()

	_, err := coll.DeleteOne(ctx, filter)
	if err != nil {
		logger.Error("Delete failed: %v", err)
		return err
	}
	return nil
}

func (r *MongoRepository) EnsureIndexes(ctx context.Context, collection string, indexes []mongo.IndexModel) error {
	coll := r.db.Collection(collection)
	ctx, cancel := context.WithTimeout(ctx, time.Duration(GetMongoClient().Config.Timeout)*time.Second)
	defer cancel()

	indexView := coll.Indexes()
	_, err := indexView.CreateMany(ctx, indexes)
	if err != nil {
		logger.Error("Index creation failed: %v", err)
		return err
	}
	logger.Info("Index created successfully for collection %s", collection)
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
