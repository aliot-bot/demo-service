package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	dsn := "postgres://demo:demo@localhost:5433/demo"
	store, err := storage.New(ctx, dsn)
	if err != nil {
		fmt.Println("Не удалось подключиться к базе:", err)
		return
	}
	defer store.Close()

	consumer := storage.NewKafkaConsumer([]string{"localhost:9092"}, "orders", "demo-group", store)
	defer consumer.Close()

	go func() {
		if err := consumer.Consume(ctx); err != nil {
			fmt.Println("Ошибка в Kafka consumer:", err)
		}
	}()

	fmt.Println("Сервис запущен")
	<-ctx.Done()
	fmt.Println("Сервис остановлен")
}
