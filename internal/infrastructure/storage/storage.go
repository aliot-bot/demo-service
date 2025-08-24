package infrastructure

import (
	"context"
	"demo-service/internal/model"
)

type Storage interface {
	SaveOrder(o *model.Order) error
	GetOrder(id string) (*model.Order, error)
	LoadCache(ctx context.Context, cache *Cache) error
}
