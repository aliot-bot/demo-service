package storage

import (
	"context"
	"demo-service/internal/model"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

func setupTestDB(t *testing.T) (*Postgres, context.Context, func()) {
	ctx := context.Background()
	dsn := "postgres://demo:demo@localhost:5433/demo"
	p, err := New(ctx, dsn)
	if err != nil {
		t.Fatalf("Failed to connect to DB: %v", err)
	}
	return p, ctx, func() { p.pool.Close() }
}

func runInTransaction(t *testing.T, ctx context.Context, p *Postgres, f func(tx pgx.Tx) error) {
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		t.Fatalf("Begin transaction failed: %v", err)
	}
	defer tx.Rollback(ctx)

	if err := f(tx); err != nil {
		t.Fatalf("Transaction operation failed: %v", err)
	}

	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("Commit failed: %v", err)
	}
}

func verifyInserted(t *testing.T, ctx context.Context, p *Postgres, table, orderUID string) {
	var uid string
	err := p.pool.QueryRow(ctx, `SELECT order_uid FROM `+table+` WHERE order_uid = $1`, orderUID).Scan(&uid)
	if err != nil {
		t.Fatalf("Failed to verify %s: %v", table, err)
	}
	if uid != orderUID {
		t.Errorf("Got OrderUID %s, want %s", uid, orderUID)
	}
}

func TestSaveOrderBase(t *testing.T) {
	p, ctx, cleanup := setupTestDB(t)
	defer cleanup()

	order := &model.Order{
		OrderUID:        "test-order-base",
		TrackNumber:     "TRACK123",
		Entry:           "ENT123",
		Locale:          "en",
		CustomerID:      "cust1",
		DeliveryService: "ups",
		Shardkey:        "1",
		SmID:            1,
		DateCreated:     time.Now(),
		OofShard:        "1",
	}

	runInTransaction(t, ctx, p, func(tx pgx.Tx) error {
		return p.saveOrderBase(tx, order)
	})

	verifyInserted(t, ctx, p, "orders", order.OrderUID)
}

func TestSaveDelivery(t *testing.T) {
	p, ctx, cleanup := setupTestDB(t)
	defer cleanup()

	order := &model.Order{
		OrderUID:        "test-delivery",
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
	}

	runInTransaction(t, ctx, p, func(tx pgx.Tx) error {
		if err := p.saveOrderBase(tx, order); err != nil {
			return err
		}
		return p.saveDelivery(tx, order)
	})

	verifyInserted(t, ctx, p, "deliveries", order.OrderUID)
}

func TestSavePayment(t *testing.T) {
	p, ctx, cleanup := setupTestDB(t)
	defer cleanup()

	order := &model.Order{
		OrderUID:        "b563feb7b2b84b6test",
		TrackNumber:     "WBILMTESTTRACK",
		Entry:           "WBIL",
		Locale:          "en",
		CustomerID:      "test",
		DeliveryService: "meest",
		Shardkey:        "9",
		SmID:            99,
		DateCreated:     time.Now(),
		OofShard:        "1",
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
	}

	runInTransaction(t, ctx, p, func(tx pgx.Tx) error {
		if err := p.saveOrderBase(tx, order); err != nil {
			return err
		}
		return p.savePayment(tx, order)
	})

	verifyInserted(t, ctx, p, "payments", order.OrderUID)
}

func TestSaveItems(t *testing.T) {
	p, ctx, cleanup := setupTestDB(t)
	defer cleanup()

	order := &model.Order{
		OrderUID:        "b563feb7b2b84b6test",
		TrackNumber:     "WBILMTESTTRACK",
		Entry:           "WBIL",
		Locale:          "en",
		CustomerID:      "test",
		DeliveryService: "meest",
		Shardkey:        "9",
		SmID:            99,
		DateCreated:     time.Now(),
		OofShard:        "1",
		Items: []model.Item{
			{
				ChrtID:      12345,
				TrackNumber: "WBILMTESTTRACK",
				Price:       100,
				Rid:         "ab4219087a764ae0btest",
				Name:        "Test Item 1",
				Sale:        10,
				Size:        "M",
				TotalPrice:  90,
				NmID:        123456,
				Brand:       "Test Brand",
				Status:      202,
			},
			{
				ChrtID:      67890,
				TrackNumber: "WBILMTESTTRACK2",
				Price:       200,
				Rid:         "cd4219087a764ae0btest",
				Name:        "Test Item 2",
				Sale:        20,
				Size:        "L",
				TotalPrice:  180,
				NmID:        789012,
				Brand:       "Test Brand 2",
				Status:      203,
			},
		},
	}

	runInTransaction(t, ctx, p, func(tx pgx.Tx) error {
		if err := p.saveOrderBase(tx, order); err != nil {
			return err
		}
		return p.saveItems(tx, order)
	})

	var count int
	err := p.pool.QueryRow(ctx, "SELECT COUNT(*) FROM items WHERE order_uid = $1", order.OrderUID).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to count items: %v", err)
	}
	if count != 2 {
		t.Errorf("Got %d items, want 2", count)
	}
}
