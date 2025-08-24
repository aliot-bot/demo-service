package postgres

import (
	"context"
	"demo-service/internal/model"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
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

func (p *Postgres) SaveOrder(o *model.Order, cache *Cache) error {
	ctx := context.Background()
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}

	_, err = tx.Exec(ctx,
		`INSERT INTO orders (order_uid, track_number, entry, locale, internal_signature, customer_id, delivery_service, shardkey, sm_id, date_created, oof_shard)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		 ON CONFLICT (order_uid) DO UPDATE SET
		 track_number = EXCLUDED.track_number, entry = EXCLUDED.entry, locale = EXCLUDED.locale,
		 internal_signature = EXCLUDED.internal_signature, customer_id = EXCLUDED.customer_id,
		 delivery_service = EXCLUDED.delivery_service, shardkey = EXCLUDED.shardkey, sm_id = EXCLUDED.sm_id,
		 date_created = EXCLUDED.date_created, oof_shard = EXCLUDED.oof_shard`,
		o.OrderUID, o.TrackNumber, o.Entry, o.Locale, o.InternalSignature, o.CustomerID,
		o.DeliveryService, o.Shardkey, o.SmID, o.DateCreated, o.OofShard)
	if err != nil {
		tx.Rollback(ctx)
		return fmt.Errorf("insert order: %w", err)
	}

	_, err = tx.Exec(ctx,
		`INSERT INTO deliveries (order_uid, name, phone, zip, city, address, region, email)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		 ON CONFLICT (order_uid) DO UPDATE SET
		 name = EXCLUDED.name, phone = EXCLUDED.phone, zip = EXCLUDED.zip, city = EXCLUDED.city,
		 address = EXCLUDED.address, region = EXCLUDED.region, email = EXCLUDED.email`,
		o.OrderUID, o.Delivery.Name, o.Delivery.Phone, o.Delivery.Zip, o.Delivery.City,
		o.Delivery.Address, o.Delivery.Region, o.Delivery.Email)
	if err != nil {
		tx.Rollback(ctx)
		return fmt.Errorf("insert delivery: %w", err)
	}

	_, err = tx.Exec(ctx,
		`INSERT INTO payments (order_uid, transaction, request_id, currency, provider, amount, payment_dt, bank, delivery_cost, goods_total, custom_fee)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		 ON CONFLICT (order_uid) DO UPDATE SET
		 transaction = EXCLUDED.transaction, request_id = EXCLUDED.request_id, currency = EXCLUDED.currency,
		 provider = EXCLUDED.provider, amount = EXCLUDED.amount, payment_dt = EXCLUDED.payment_dt,
		 bank = EXCLUDED.bank, delivery_cost = EXCLUDED.delivery_cost, goods_total = EXCLUDED.goods_total,
		 custom_fee = EXCLUDED.custom_fee`,
		o.OrderUID, o.Payment.Transaction, o.Payment.RequestID, o.Payment.Currency, o.Payment.Provider,
		o.Payment.Amount, o.Payment.PaymentDt, o.Payment.Bank, o.Payment.DeliveryCost,
		o.Payment.GoodsTotal, o.Payment.CustomFee)
	if err != nil {
		tx.Rollback(ctx)
		return fmt.Errorf("insert payment: %w", err)
	}

	_, err = tx.Exec(ctx, `DELETE FROM items WHERE order_uid = $1`, o.OrderUID)
	if err != nil {
		tx.Rollback(ctx)
		return fmt.Errorf("delete items: %w", err)
	}
	for _, item := range o.Items {
		_, err = tx.Exec(ctx,
			`INSERT INTO items (order_uid, chrt_id, track_number, price, rid, name, sale, size, total_price, nm_id, brand, status)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`,
			o.OrderUID, item.ChrtID, item.TrackNumber, item.Price, item.Rid, item.Name,
			item.Sale, item.Size, item.TotalPrice, item.NmID, item.Brand, item.Status)
		if err != nil {
			tx.Rollback(ctx)
			return fmt.Errorf("insert item: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return err
	}

	if cache != nil {
		cache.Set(o)
	}

	return nil
}

func (p *Postgres) LoadCache(ctx context.Context, cache *Cache) error {
	rows, err := p.pool.Query(ctx, `
		SELECT o.order_uid, o.track_number, o.entry, o.locale, o.internal_signature, o.customer_id, 
		       o.delivery_service, o.shardkey, o.sm_id, o.date_created, o.oof_shard,
		       d.name, d.phone, d.zip, d.city, d.address, d.region, d.email,
		       p.transaction, p.request_id, p.currency, p.provider, p.amount, p.payment_dt, 
		       p.bank, p.delivery_cost, p.goods_total, p.custom_fee
		FROM orders o
		LEFT JOIN deliveries d ON o.order_uid = d.order_uid
		LEFT JOIN payments p ON o.order_uid = p.order_uid`)
	if err != nil {
		return fmt.Errorf("load cache query: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var order model.Order
		err := rows.Scan(
			&order.OrderUID, &order.TrackNumber, &order.Entry, &order.Locale, &order.InternalSignature,
			&order.CustomerID, &order.DeliveryService, &order.Shardkey, &order.SmID, &order.DateCreated,
			&order.OofShard, &order.Delivery.Name, &order.Delivery.Phone, &order.Delivery.Zip,
			&order.Delivery.City, &order.Delivery.Address, &order.Delivery.Region, &order.Delivery.Email,
			&order.Payment.Transaction, &order.Payment.RequestID, &order.Payment.Currency,
			&order.Payment.Provider, &order.Payment.Amount, &order.Payment.PaymentDt, &order.Payment.Bank,
			&order.Payment.DeliveryCost, &order.Payment.GoodsTotal, &order.Payment.CustomFee)
		if err != nil {
			log.Error().Err(err).Msg("Failed to scan order for cache")
			continue
		}
		itemRows, err := p.pool.Query(ctx, `
			SELECT chrt_id, track_number, price, rid, name, sale, size, total_price, nm_id, brand, status
			FROM items WHERE order_uid = $1`, order.OrderUID)
		if err != nil {
			log.Error().Err(err).Str("order_uid", order.OrderUID).Msg("Failed to load items for cache")
			continue
		}
		for itemRows.Next() {
			var item model.Item
			if err := itemRows.Scan(&item.ChrtID, &item.TrackNumber, &item.Price, &item.Rid, &item.Name,
				&item.Sale, &item.Size, &item.TotalPrice, &item.NmID, &item.Brand, &item.Status); err != nil {
				log.Error().Err(err).Str("order_uid", order.OrderUID).Msg("Failed to scan item")
				continue
			}
			order.Items = append(order.Items, item)
		}
		itemRows.Close()
		cache.Set(&order)
	}
	return nil
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
