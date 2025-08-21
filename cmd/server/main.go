package main

import (
	"context"
	"demo-service/internal/infrastructure"
	"log"
	"os"
	"os/signal"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	dsn := "postgres://demo:demo@localhost:5433/demo"
	store, err := infrastructure.New(ctx, dsn)
	if err != nil {
		log.Fatalf("Не удалось подключиться к базе: %v", err)
	}
	defer store.Close()

	consumer := infrastructure.NewKafkaConsumer([]string{"localhost:9092"}, "orders", "demo-group", store)
	defer consumer.Close()

	go func() {
		if err := consumer.Consume(ctx); err != nil {
			log.Fatalf("Ошибка в Kafka consumer: %v", err)
		}
	}()

	log.Println("Сервис запущен")
	<-ctx.Done()
	log.Println("Сервис остановлен")
}
