package infrastructure

import (
	"context"
	"demo-service/internal/model"
	"encoding/json"
	"fmt"

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
		MinBytes: 1e3,  // минимум 1KB
		MaxBytes: 10e6, // максимум 10MB
	})
	return &KafkaConsumer{reader: r, storage: storage}
}

func (c *KafkaConsumer) Consume(ctx context.Context) error {
	for {
		// проверка на завершение контекста
		if ctx.Err() != nil {
			return ctx.Err()
		}

		msg, err := c.reader.ReadMessage(ctx)
		if err != nil {
			return fmt.Errorf("read message: %w", err)
		}

		var order model.Order
		if err := json.Unmarshal(msg.Value, &order); err != nil {
			fmt.Println("Ошибка при парсинге сообщения:", err)
			continue
		}

		if err := c.storage.SaveOrder(&order); err != nil {
			fmt.Println("Ошибка при сохранении заказа:", err)
			continue
		}

		fmt.Println("Заказ сохранён:", order.OrderUID)

		// фиксируем оффсет
		if err := c.reader.CommitMessages(ctx, msg); err != nil {
			fmt.Println("Ошибка при коммите сообщения:", err)
		}
	}
}

func (c *KafkaConsumer) Close() {
	_ = c.reader.Close()
}
