package cache

import (
	"demo-service/internal/model"
	"log"
	"sync"
)

type Cache struct {
	orders map[string]*model.Order
	mu     sync.RWMutex
}

func NewCache() *Cache {
	return &Cache{
		orders: make(map[string]*model.Order),
	}
}

func (c *Cache) Set(order *model.Order) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, exists := c.orders[order.OrderUID]; exists {
		log.Printf("Заказ %s уже существует в кэше, пропущена перезапись", order.OrderUID)
		return false
	}

	c.orders[order.OrderUID] = order
	log.Printf("Заказ %s добавлен в кэш", order.OrderUID)
	return true
}

func (c *Cache) Get(orderUID string) (*model.Order, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	order, exists := c.orders[orderUID]
	if exists {
		log.Printf("Заказ %s найден в кэше", orderUID)
	} else {
		log.Printf("Заказ %s не найден в кэше", orderUID)
	}
	return order, exists
}
