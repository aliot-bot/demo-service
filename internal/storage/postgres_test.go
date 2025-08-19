package storage

import (
	"context"
	"testing"
	"time"

	"demo-service/internal/model"
)

func TestSaveOrderBase(t *testing.T) {
	ctx := context.Background()
	dsn := "postgres://demo:demo@localhost:5433/demo"
	p, err := New(ctx, dsn)
	if err != nil {
		t.Fatalf("Failed to connect to DB: %v", err)
	}
	defer p.pool.Close()

	order := &model.Order{
		OrderUID:          "b563feb7b2b84b6test",
		TrackNumber:       "WBILMTESTTRACK",
		Entry:             "WBIL",
		Locale:            "elfiyskiy",
		InternalSignature: "",
		CustomerID:        "test",
		DeliveryService:   "meest",
		Shardkey:          "9",
		SmID:              99,
		DateCreated:       time.Date(2021, 11, 26, 6, 22, 19, 0, time.UTC),
		OofShard:          "1",
	}

	tx, err := p.pool.Begin(ctx)
	if err != nil {
		t.Fatalf("Begin transaction failed: %v", err)
	}
	defer tx.Rollback(ctx)

	err = p.saveOrderBase(tx, order)
	if err != nil {
		t.Fatalf("saveOrderBase failed: %v", err)
	}

	err = tx.Commit(ctx)
	if err != nil {
		t.Fatalf("Commit failed: %v", err)
	}

	var uid string
	err = p.pool.QueryRow(ctx, `SELECT order_uid FROM orders WHERE order_uid = $1`, order.OrderUID).Scan(&uid)
	if err != nil {
		t.Fatalf("Failed to verify order: %v", err)
	}
	if uid != order.OrderUID {
		t.Errorf("Got OrderUID %s, want %s", uid, order.OrderUID)
	}
}
