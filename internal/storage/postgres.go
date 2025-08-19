package storage

import (
	"context"
	"demo-service/internal/model"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Postgres struct {
	pool *pgxpool.Pool
}

func New(ctx context.Context, dsn string) (*Postgres, error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("connect: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping: %w", err)
	}

	return &Postgres{pool: pool}, nil
}

func (p *Postgres) SaveOrder(o *model.Order) error {
	tx, err := p.pool.Begin(context.Background())
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(context.Background())

	_, err = tx.Exec(context.Background(),
		`INSERT INTO orders (order_uid, track_number, entry, locale, internal_signature, customer_id, delivery_service, shardkey, sm_id, date_created, oof_shard)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
		 ON CONFLICT (order_uid) DO UPDATE SET
		 track_number=EXCLUDED.track_number, entry=EXCLUDED.entry, locale=EXCLUDED.locale,
		 internal_signature=EXCLUDED.internal_signature, customer_id=EXCLUDED.customer_id,
		 delivery_service=EXCLUDED.delivery_service, shardkey=EXCLUDED.shardkey, sm_id=EXCLUDED.sm_id,
		 date_created=EXCLUDED.date_created, oof_shard=EXCLUDED.oof_shard`,
		o.OrderUID, o.TrackNumber, o.Entry, o.Locale, o.InternalSignature, o.CustomerID,
		o.DeliveryService, o.Shardkey, o.SmID, o.DateCreated, o.OofShard)
	if err != nil {
		return fmt.Errorf("insert order: %w", err)
	}

	_, err = tx.Exec(context.Background(),
		`INSERT INTO deliveries (order_uid, name, phone, zip, city, address, region, email)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
		 ON CONFLICT (order_uid) DO UPDATE SET
		 name=EXCLUDED.name, phone=EXCLUDED.phone, zip=EXCLUDED.zip, city=EXCLUDED.city,
		 address=EXCLUDED.address, region=EXCLUDED.region, email=EXCLUDED.email`,
		o.OrderUID, o.Delivery.Name, o.Delivery.Phone, o.Delivery.Zip, o.Delivery.City,
		o.Delivery.Address, o.Delivery.Region, o.Delivery.Email)
	if err != nil {
		return fmt.Errorf("insert delivery: %w", err)
	}

	_, err = tx.Exec(context.Background(),
		`INSERT INTO payments (order_uid, transaction, request_id, currency, provider, amount, payment_dt, bank, delivery_cost, goods_total, custom_fee)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
		 ON CONFLICT (order_uid) DO UPDATE SET
		 transaction=EXCLUDED.transaction, request_id=EXCLUDED.request_id, currency=EXCLUDED.currency,
		 provider=EXCLUDED.provider, amount=EXCLUDED.amount, payment_dt=EXCLUDED.payment_dt,
		 bank=EXCLUDED.bank, delivery_cost=EXCLUDED.delivery_cost, goods_total=EXCLUDED.goods_total,
		 custom_fee=EXCLUDED.custom_fee`,
		o.OrderUID, o.Payment.Transaction, o.Payment.RequestID, o.Payment.Currency, o.Payment.Provider,
		o.Payment.Amount, o.Payment.PaymentDt, o.Payment.Bank, o.Payment.DeliveryCost,
		o.Payment.GoodsTotal, o.Payment.CustomFee)
	if err != nil {
		return fmt.Errorf("insert payment: %w", err)
	}

	_, err = tx.Exec(context.Background(), `DELETE FROM items WHERE order_uid=$1`, o.OrderUID)
	if err != nil {
		return fmt.Errorf("delete items: %w", err)
	}
	for _, item := range o.Items {
		_, err = tx.Exec(context.Background(),
			`INSERT INTO items (order_uid,chrt_id,track_number,price,rid,name,sale,size,total_price,nm_id,brand,status)
			 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`,
			o.OrderUID, item.ChrtID, item.TrackNumber, item.Price, item.Rid, item.Name,
			item.Sale, item.Size, item.TotalPrice, item.NmID, item.Brand, item.Status)
		if err != nil {
			return fmt.Errorf("insert item: %w", err)
		}
	}

	return tx.Commit(context.Background())
}

func (p *Postgres) Close() {
	p.pool.Close()
}

func (p *Postgres) GetOrder(ctx context.Context, orderUID string) (*model.Order, error) {
	order := &model.Order{}

	row := p.pool.QueryRow(ctx, `
		SELECT order_uid, track_number, entry, locale, internal_signature, customer_id,
		       delivery_service, shardkey, sm_id, date_created, oof_shard
		FROM orders WHERE order_uid = $1`, orderUID)

	err := row.Scan(&order.OrderUID, &order.TrackNumber, &order.Entry, &order.Locale,
		&order.InternalSignature, &order.CustomerID, &order.DeliveryService,
		&order.Shardkey, &order.SmID, &order.DateCreated, &order.OofShard)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("order not found")
		}
		return nil, err
	}

	row = p.pool.QueryRow(ctx, `
		SELECT name, phone, zip, city, address, region, email
		FROM deliveries WHERE order_uid = $1`, orderUID)

	err = row.Scan(&order.Delivery.Name, &order.Delivery.Phone, &order.Delivery.Zip,
		&order.Delivery.City, &order.Delivery.Address, &order.Delivery.Region, &order.Delivery.Email)
	if err != nil && err != pgx.ErrNoRows {
		return nil, err
	}

	row = p.pool.QueryRow(ctx, `
		SELECT transaction, request_id, currency, provider, amount, payment_dt, bank, 
		       delivery_cost, goods_total, custom_fee
		FROM payments WHERE order_uid = $1`, orderUID)

	err = row.Scan(&order.Payment.Transaction, &order.Payment.RequestID, &order.Payment.Currency,
		&order.Payment.Provider, &order.Payment.Amount, &order.Payment.PaymentDt,
		&order.Payment.Bank, &order.Payment.DeliveryCost, &order.Payment.GoodsTotal,
		&order.Payment.CustomFee)
	if err != nil && err != pgx.ErrNoRows {
		return nil, err
	}

	rows, err := p.pool.Query(ctx, `
		SELECT chrt_id, track_number, price, rid, name, sale, size, 
		       total_price, nm_id, brand, status
		FROM items WHERE order_uid = $1`, orderUID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var item model.Item
		err := rows.Scan(&item.ChrtID, &item.TrackNumber, &item.Price, &item.Rid, &item.Name,
			&item.Sale, &item.Size, &item.TotalPrice, &item.NmID, &item.Brand, &item.Status)
		if err != nil {
			return nil, err
		}
		order.Items = append(order.Items, item)
	}

	return order, nil
}
