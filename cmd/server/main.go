package main

import (
	"context"
	"demo-service/internal/infrastructure"
	"fmt"
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

	cache := infrastructure.NewCache()
	if err := store.LoadCache(ctx, cache); err != nil {
		log.Fatal("Ошибка загрузки кеша")
	} else {
		fmt.Println("КЕЕЕЕЕЕЕЕЕЕЕЕЕШ")
	}

	consumer := infrastructure.NewKafkaConsumer([]string{"localhost:9092"}, "orders", "demo-group", store)
	defer consumer.Close()

	go func() {
		if err := consumer.Consume(ctx, cache); err != nil {
			log.Fatalf("Ошибка в Kafka consumer: %v", err)
		}
	}()

	server := infrastructure.NewServer(cache, store)
	go func() {
		if err := server.Start(":8081"); err != nil {
			log.Printf("Сервер HTTP остановлен с ошибкой: %v", err)
		}
	}()

	log.Println("Сервис запущен")
	<-ctx.Done()
	log.Println("Сервис остановлен")
}
