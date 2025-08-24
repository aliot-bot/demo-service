package main

import (
	"context"
	"demo-service/internal/infrastructure/cache"
	"demo-service/internal/infrastructure/httpserver"
	"demo-service/internal/infrastructure/kafka"
	"demo-service/internal/infrastructure/postgres"
	"log"
	"os"
	"os/signal"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	dsn := "postgres://demo:demo@localhost:5433/demo"
	store, err := postgres.New(ctx, dsn)
	if err != nil {
		log.Fatalf("Не удалось подключиться к базе: %v", err)
	}
	defer store.Close()

	c := cache.NewCache()
	if err := store.LoadCache(ctx, c); err != nil {
		log.Fatal("Ошибка загрузки кеша")
	}

	consumer := kafka.NewKafkaConsumer([]string{"localhost:9092"}, "orders", "demo-group", store)
	defer consumer.Close()

	go func() {
		if err := consumer.Consume(ctx, c); err != nil {
			log.Fatalf("Ошибка в Kafka consumer: %v", err)
		}
	}()

	server := httpserver.NewServer(c, store)
	go func() {
		if err := server.Start(":8081"); err != nil {
			log.Printf("Сервер HTTP остановлен с ошибкой: %v", err)
		}
	}()

	log.Println("Сервис запущен")
	<-ctx.Done()
	log.Println("Сервис остановлен")
}
