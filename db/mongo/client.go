package db

import (
	"context"
	"sync"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/world-in-progress/yggdrasil/config"
	"github.com/world-in-progress/yggdrasil/core/logger"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongoClient struct {
	Client   *mongo.Client
	Database *mongo.Database
	Config   config.MongoConfig
}

var (
	instance *MongoClient
	once     sync.Once
)

// GetMongoClient gets the instance of Mongo client
func GetMongoClient() *MongoClient {
	once.Do(func() {
		cfg := config.LoadMongoConfig()
		clientOptions := options.Client().
			ApplyURI(cfg.URI).
			SetConnectTimeout(time.Duration(cfg.Timeout) * time.Second).
			SetMaxPoolSize(100)

		var client *mongo.Client
		var err error

		// exponential backoff retry connection
		retry := backoff.NewExponentialBackOff()
		retry.MaxElapsedTime = 30 * time.Second
		err = backoff.Retry(func() error {
			client, err = mongo.Connect(context.Background(), clientOptions)
			if err != nil {
				logger.Fatal("Failed to connect MongoDB: %v", err)
				return err
			}
			return client.Ping(context.Background(), nil)
		}, retry)

		if err != nil {
			logger.Fatal("MongoDB reconnection failed: %v", err)
		}

		instance = &MongoClient{
			Client:   client,
			Database: client.Database(cfg.Database),
			Config:   cfg,
		}
		logger.Info("MongoDB connection successful: %s", cfg.URI)
	})

	return instance
}

func (m *MongoClient) Close() {
	if m.Client != nil {
		if err := m.Client.Disconnect(context.Background()); err != nil {
			logger.Error("Failed to close MongoDB connection: %v", err)
		}
	}
}
