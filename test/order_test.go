package test

import (
	"demo-service/internal/model"
	"encoding/json"
	"reflect"
	"testing"
	"time"
)

func TestOrderJSON(t *testing.T) {
	order := model.Order{
		OrderUID:    "b563feb7b2b84b6test",
		TrackNumber: "WBILMTESTTRACK",
		Entry:       "WBIL",
		Delivery: model.Delivery{
			Name:    "Test Testov",
			Phone:   "+9720000000",
			Zip:     "2639809",
			City:    "Kiryat Mozkin",
			Address: "Ploshad Mira 15",
			Region:  "Kraiot",
			Email:   "test@gmail.com",
		},
		Payment: model.Payment{
			Transaction:  "b563feb7b2b84b6test",
			Currency:     "USD",
			Provider:     "wbpay",
			Amount:       1817,
			PaymentDt:    1637907727,
			Bank:         "alpha",
			DeliveryCost: 1500,
			GoodsTotal:   317,
		},
		Items: []model.Item{
			{
				ChrtID:      9934930,
				TrackNumber: "WBILMTESTTRACK",
				Price:       453,
				Rid:         "ab4219087a764ae0btest",
				Name:        "Mascaras",
				Sale:        30,
				Size:        "0",
				TotalPrice:  317,
				NmID:        2389212,
				Brand:       "Vivienne Sabo",
				Status:      202,
			},
		},
		Locale:          "en",
		CustomerID:      "test",
		DeliveryService: "meest",
		Shardkey:        "9",
		SmID:            99,
		DateCreated:     time.Date(2021, 11, 26, 6, 22, 19, 0, time.UTC),
		OofShard:        "1",
	}

	data, err := json.Marshal(order)
	if err != nil {
		t.Fatalf("Marshall failed: %v", err)
	}

	var got model.Order
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshall failed: %v", err)
	}

	if !reflect.DeepEqual(order, got) {
		t.Errorf("Got %v, want %v", got, order)
	}
}
