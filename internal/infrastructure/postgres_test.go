package infrastructure

import (
	"context"
	"demo-service/internal/model"
	"testing"
	"time"
)

func setupTestDB(t *testing.T) (*Postgres, context.Context, func()) {
	ctx := context.Background()
	dsn := "postgres://demo:demo@localhost:5433/demo"
	p, err := New(ctx, dsn)
	if err != nil {
		t.Fatalf("Failed to connect to DB: %v", err)
	}
	return p, ctx, func() { p.Close() }
}

func verifyInserted(t *testing.T, ctx context.Context, p *Postgres, table, orderUID string, expectedCount int) {
	var count int
	err := p.pool.QueryRow(ctx, "SELECT COUNT(*) FROM "+table+" WHERE order_uid=$1", orderUID).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to verify %s: %v", table, err)
	}
	if count != expectedCount {
		t.Errorf("Got %d rows in %s, want %d", count, table, expectedCount)
	}
}

func TestSaveOrder(t *testing.T) {
	p, ctx, cleanup := setupTestDB(t)
	defer cleanup()

	order := &model.Order{
		OrderUID:        "test-order-full",
		TrackNumber:     "TRACK123",
		Entry:           "ENT123",
		Locale:          "en",
		CustomerID:      "cust1",
		DeliveryService: "ups",
		Shardkey:        "1",
		SmID:            1,
		DateCreated:     time.Now(),
		OofShard:        "1",
		Delivery: model.Delivery{
			Name:    "John Doe",
			Phone:   "+123456789",
			Zip:     "12345",
			City:    "TestCity",
			Address: "123 Test St",
			Region:  "TestRegion",
			Email:   "john@example.com",
		},
		Payment: model.Payment{
			Transaction:  "tx123",
			Currency:     "USD",
			Provider:     "wbpay",
			Amount:       1000,
			PaymentDt:    1690000000,
			Bank:         "alpha",
			DeliveryCost: 200,
			GoodsTotal:   800,
		},
		Items: []model.Item{
			{
				ChrtID:      1,
				TrackNumber: "TRACK123",
				Price:       500,
				Rid:         "rid1",
				Name:        "Item1",
				Sale:        50,
				Size:        "M",
				TotalPrice:  450,
				NmID:        111,
				Brand:       "Brand1",
				Status:      202,
			},
			{
				ChrtID:      2,
				TrackNumber: "TRACK123",
				Price:       550,
				Rid:         "rid2",
				Name:        "Item2",
				Sale:        50,
				Size:        "L",
				TotalPrice:  500,
				NmID:        222,
				Brand:       "Brand2",
				Status:      202,
			},
		},
	}

	if err := p.SaveOrder(order); err != nil {
		t.Fatalf("SaveOrder failed: %v", err)
	}

	verifyInserted(t, ctx, p, "orders", order.OrderUID, 1)
	verifyInserted(t, ctx, p, "deliveries", order.OrderUID, 1)
	verifyInserted(t, ctx, p, "payments", order.OrderUID, 1)
	verifyInserted(t, ctx, p, "items", order.OrderUID, len(order.Items))
}

func TestGetOrder_Success(t *testing.T) {
	p, ctx, cleanup := setupTestDB(t)
	defer cleanup()

	order := &model.Order{
		OrderUID:        "test-getorder-1",
		TrackNumber:     "TRACK999",
		Entry:           "ENT999",
		Locale:          "en",
		CustomerID:      "cust2",
		DeliveryService: "fedex",
		Shardkey:        "2",
		SmID:            2,
		DateCreated:     time.Now(),
		OofShard:        "2",
		Delivery: model.Delivery{
			Name:    "Alice",
			Phone:   "+987654321",
			Zip:     "54321",
			City:    "CityTest",
			Address: "456 Test Ave",
			Region:  "RegionTest",
			Email:   "alice@example.com",
		},
		Payment: model.Payment{
			Transaction:  "tx999",
			Currency:     "EUR",
			Provider:     "paypal",
			Amount:       2000,
			PaymentDt:    1690000001,
			Bank:         "sber",
			DeliveryCost: 300,
			GoodsTotal:   1700,
		},
		Items: []model.Item{
			{
				ChrtID:      10,
				TrackNumber: "TRACK999",
				Price:       1000,
				Rid:         "rid10",
				Name:        "Item10",
				Sale:        100,
				Size:        "XL",
				TotalPrice:  900,
				NmID:        333,
				Brand:       "Brand10",
				Status:      200,
			},
		},
	}

	if err := p.SaveOrder(order); err != nil {
		t.Fatalf("SaveOrder failed: %v", err)
	}

	got, err := p.GetOrder(ctx, order.OrderUID)
	if err != nil {
		t.Fatalf("GetOrder failed: %v", err)
	}
	if got == nil {
		t.Fatal("GetOrder returned nil order")
	}

	if got.OrderUID != order.OrderUID {
		t.Errorf("expected OrderUID %s, got %s", order.OrderUID, got.OrderUID)
	}
	if got.CustomerID != order.CustomerID {
		t.Errorf("expected CustomerID %s, got %s", order.CustomerID, got.CustomerID)
	}
	if len(got.Items) != len(order.Items) {
		t.Errorf("expected %d items, got %d", len(order.Items), len(got.Items))
	}
}

func TestGetOrder_NotFound(t *testing.T) {
	p, ctx, cleanup := setupTestDB(t)
	defer cleanup()

	got, err := p.GetOrder(ctx, "non-existent-order")
	if err == nil {
		t.Errorf("expected error for non-existent order, got nil")
	}
	if got != nil {
		t.Errorf("expected nil order, got %+v", got)
	}
}
