package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"time"

	"demo-service/internal/model"

	"github.com/segmentio/kafka-go"
)

func generateOrder() model.Order {

	orderID := fmt.Sprintf("%d", rand.Int63()) + "test"

	delivery := model.Delivery{
		Name:    "Test User",
		Phone:   "+9721234567",
		Zip:     "1234567",
		City:    "Kiryat Mozkin",
		Address: "Ploshad Mira 1",
		Region:  "Kraiot",
		Email:   "test@gmail.com",
	}

	payment := model.Payment{
		Transaction:  orderID,
		Currency:     "USD",
		Provider:     "wbpay",
		Amount:       2000,
		PaymentDt:    time.Now().Unix(),
		Bank:         "alpha",
		DeliveryCost: 700,
		GoodsTotal:   1200,
		CustomFee:    0,
	}

	item := model.Item{
		ChrtID:      1234567,
		TrackNumber: "WBILM" + orderID,
		Price:       300,
		Rid:         "ab123test",
		Name:        "Mascaras",
		Sale:        10,
		Size:        "M",
		TotalPrice:  270,
		NmID:        7654321,
		Brand:       "Vivienne Sabo",
		Status:      202,
	}

	order := model.Order{
		OrderUID:        orderID,
		TrackNumber:     "WBILM" + orderID,
		Entry:           "WBIL",
		Delivery:        delivery,
		Payment:         payment,
		Items:           []model.Item{item},
		Locale:          "en",
		CustomerID:      "test",
		DeliveryService: "meest",
		Shardkey:        "1",
		SmID:            1,
		DateCreated:     time.Now().UTC(),
		OofShard:        "1",
	}

	return order
}

func main() {
	writer := &kafka.Writer{
		Addr:     kafka.TCP("localhost:9092"),
		Topic:    "orders",
		Balancer: &kafka.LeastBytes{},
	}

	for i := 0; i < 5; i++ {
		order := generateOrder()
		data, err := json.Marshal(order)
		if err != nil {
			fmt.Println("Ошибка при сериализации:", err)
			continue
		}

		err = writer.WriteMessages(context.Background(),
			kafka.Message{
				Key:   []byte(order.OrderUID),
				Value: data,
			},
		)
		if err != nil {
			fmt.Println("Ошибка при отправке:", err)
		} else {
			fmt.Println("Сообщение отправлено для заказа:", order.OrderUID)
		}

		time.Sleep(time.Second)
	}

	if err := writer.Close(); err != nil {
		fmt.Println("Ошибка при закрытии продюсера:", err)
	}
}
