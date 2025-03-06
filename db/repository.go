package db

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
)

type Repository interface {
	Create(ctx context.Context, collection string, document any) (string, error)
	Read(ctx context.Context, collection string, filter bson.M) (any, error)
	Update(ctx context.Context, collection string, filter bson.M, update bson.M) error
	Delete(ctx context.Context, collection string, filter bson.M) error
}
