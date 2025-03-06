package db

import (
	"context"
)

type Repository interface {
	Create(ctx context.Context, table string, record any) (string, error)
	Read(ctx context.Context, table string, filter any) (any, error)
	Update(ctx context.Context, table string, filter any, update any) error
	Delete(ctx context.Context, table string, filter any) error
}
