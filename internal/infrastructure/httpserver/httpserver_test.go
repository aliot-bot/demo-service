package httpserver

import (
	"context"
	"demo-service/internal/infrastructure/cache"
	"demo-service/internal/infrastructure/postgres"
	"demo-service/internal/model"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestHandleGetOrder(t *testing.T) {
	store, err := postgres.New(context.Background(), "postgres://demo:demo@localhost:5433/demo")
	if err != nil {
		t.Fatalf("Ошибка подключения к БД: %v", err)
	}
	defer store.Close()

	cache := cache.NewCache()
	order := makeTestOrder("test-http")
	if err := store.SaveOrder(&order, cache); err != nil {
		t.Fatalf("Ошибка сохранения: %v", err)
	}

	server := NewServer(cache, store)
	req, _ := http.NewRequest("GET", "/order/"+order.OrderUID, nil)
	rr := httptest.NewRecorder()
	server.router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Ожидался код 200, получен %d", rr.Code)
	}

	var resp model.Order
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("Ошибка декодирования: %v", err)
	}
	if resp.OrderUID != order.OrderUID {
		t.Errorf("Ожидался %s, получен %s", order.OrderUID, resp.OrderUID)
	}
}

func makeTestOrder(uid string) model.Order {
	return model.Order{
		OrderUID:    uid,
		TrackNumber: "WBILMTESTTRACK",
		Entry:       "WBIL",
		Delivery: model.Delivery{
			Name: "Test Testov", Phone: "+9720000000",
			Zip: "2639809", City: "Kiryat Mozkin",
			Address: "Ploshad Mira 15", Region: "Kraiot", Email: "test@gmail.com",
		},
		Payment: model.Payment{
			Transaction: uid, Currency: "USD", Provider: "wbpay",
			Amount: 1817, PaymentDt: 1637907727, Bank: "alpha",
			DeliveryCost: 1500, GoodsTotal: 317,
		},
		Items: []model.Item{{
			ChrtID: 9934930, TrackNumber: "WBILMTESTTRACK", Price: 453,
			Rid: "ab4219087a764ae0btest", Name: "Mascaras",
			Sale: 30, Size: "0", TotalPrice: 317, NmID: 2389212,
			Brand: "Vivienne Sabo", Status: 202,
		}},
		Locale: "en", CustomerID: "test", DeliveryService: "meest",
		Shardkey: "9", SmID: 99, DateCreated: time.Now(), OofShard: "1",
	}
}
