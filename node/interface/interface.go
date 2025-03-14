package nodeinterface

import "context"

type (
	IRepository interface {
		Create(ctx context.Context, table string, record map[string]any) (string, error)
		ReadOne(ctx context.Context, table string, filter map[string]any) (map[string]any, error)
		ReadAll(ctx context.Context, table string, filter map[string]any) ([]map[string]any, error)
		Update(ctx context.Context, table string, filter map[string]any, update map[string]any) error
		Delete(ctx context.Context, table string, filter map[string]any) error
		Count(ctx context.Context, table string, filter map[string]any) (int64, error)
	}
)
