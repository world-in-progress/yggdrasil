package db

import (
	"context"
)

type IRepository interface {
	Create(ctx context.Context, table string, record any) (string, error)
	Read(ctx context.Context, table string, filter any) (map[string]any, error)
	Update(ctx context.Context, table string, filter any, update any) error
	Delete(ctx context.Context, table string, filter any) error
}
