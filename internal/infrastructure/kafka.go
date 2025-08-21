package infrastructure

import (
	"context"
	"demo-service/internal/model"
	"encoding/json"
	"fmt"
	"log"

	"github.com/segmentio/kafka-go"
)

type KafkaConsumer struct {
	reader  *kafka.Reader
	storage *Postgres
}

func NewKafkaConsumer(brokers []string, topic, groupID string, storage *Postgres) *KafkaConsumer {
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  brokers,
		Topic:    topic,
		GroupID:  groupID,
		MinBytes: 1e3,
		MaxBytes: 10e6,
	})
	return &KafkaConsumer{reader: r, storage: storage}
}

func (c *KafkaConsumer) Consume(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			msg, err := c.reader.ReadMessage(ctx)
			if err != nil {
				log.Printf("Ошибка чтения сообщения из Kafka: %v", err)
				return fmt.Errorf("read message: %w", err)
			}
			log.Printf("Получено сообщение: %s", string(msg.Value))

			var order model.Order
			if err := json.Unmarshal(msg.Value, &order); err != nil {
				log.Printf("Ошибка парсинга сообщения: %v, данные: %s", err, string(msg.Value))
				continue
			}

			if err := c.storage.SaveOrder(&order); err != nil {
				log.Printf("Ошибка сохранения заказа %s: %v", order.OrderUID, err)
				continue
			}

			log.Printf("Заказ сохранён: %s", order.OrderUID)

			if err := c.reader.CommitMessages(ctx, msg); err != nil {
				log.Printf("Ошибка коммита сообщения для заказа %s: %v", order.OrderUID, err)
			}
		}
	}
}

func (c *KafkaConsumer) Close() {
	_ = c.reader.Close()
}
