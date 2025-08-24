package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"demo-service/internal/infrastructure/cache"
	"demo-service/internal/infrastructure/postgres"
	"demo-service/internal/model"

	"github.com/segmentio/kafka-go"
)

type KafkaConsumer struct {
	reader  *kafka.Reader
	storage *postgres.Postgres
}

func NewKafkaConsumer(brokers []string, topic, groupID string, storage *postgres.Postgres) *KafkaConsumer {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  brokers,
		Topic:    topic,
		GroupID:  groupID,
		MinBytes: 1e4,
		MaxBytes: 1e7,
	})
	return &KafkaConsumer{reader: reader, storage: storage}
}

func (c *KafkaConsumer) Consume(ctx context.Context, cacheStore *cache.Cache) error {
	for {
		select {
		case <-ctx.Done():
			log.Println("Контекст завершён, останавливаем consumer")
			return ctx.Err()
		default:
		}

		msg, err := c.reader.ReadMessage(ctx)
		if err != nil {
			log.Printf("Ошибка чтения из Kafka: %v", err)
			return fmt.Errorf("read message: %w", err)
		}
		var order model.Order
		if err := json.Unmarshal(msg.Value, &order); err != nil {
			log.Printf("Ошибка парсинга JSON: %v, данные: %s", err, string(msg.Value))
			continue
		}

		if err := c.storage.SaveOrder(&order, cacheStore); err != nil {
			log.Printf("Ошибка сохранения заказа %s: %v", order.OrderUID, err)
			continue
		}

		log.Printf("Заказ обработан: %s", order.OrderUID)

		if err := c.reader.CommitMessages(ctx, msg); err != nil {
			log.Printf("Ошибка коммита сообщения %s: %v", order.OrderUID, err)
		}
	}
}

func (c *KafkaConsumer) Close() {
	if err := c.reader.Close(); err != nil {
		log.Printf("Ошибка закрытия Kafka reader: %v", err)
	}
}
