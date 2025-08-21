package infrastructure

import "demo-service/internal/model"

type Storage interface {
	SaveOrder(o *model.Order) error
	GetOrder(id string) (*model.Order, error)
}
