package storage

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
		log.Error().Err(err).Msg("Failed to create pool")
		return nil, fmt.Errorf("connect: %w", err)
	}

	err = pool.Ping(ctx)
	if err != nil {
		pool.Close()
		log.Error().Err(err).Msg("Failed to ping DB")
		return nil, fmt.Errorf("ping: %w", err)
	}

	log.Info().Msg("Connected to Data Base")
	return &Postgres{pool: pool}, nil
}

func (p *Postgres) saveOrderBase(tx pgx.Tx, o *model.Order) error {
	_, err := tx.Exec(context.Background(),
		`INSERT INTO orders (order_uid, track_number, entry, locale, internal_signature, customer_id, delivery_service, shardkey, sm_id, date_created, oof_shard)
         VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
         ON CONFLICT (order_uid) DO UPDATE SET
            track_number = EXCLUDED.track_number,
            entry = EXCLUDED.entry,
            locale = EXCLUDED.locale,
            internal_signature = EXCLUDED.internal_signature,
            customer_id = EXCLUDED.customer_id,
            delivery_service = EXCLUDED.delivery_service,
            shardkey = EXCLUDED.shardkey,
            sm_id = EXCLUDED.sm_id,
            date_created = EXCLUDED.date_created,
            oof_shard = EXCLUDED.oof_shard`,
		o.OrderUID, o.TrackNumber, o.Entry, o.Locale, o.InternalSignature, o.CustomerID,
		o.DeliveryService, o.Shardkey, o.SmID, o.DateCreated, o.OofShard)

	if err != nil {
		log.Error().Err(err).Str("order_uid", o.OrderUID).Msg("Failed to insert order")
		return fmt.Errorf("insert order: %w", err)
	}

	return nil
}

func (p *Postgres) saveDelivery(tx pgx.Tx, o *model.Order) error {
	_, err := tx.Exec(context.Background(),
		`INSERT INTO deliveries (order_uid, name, phone, zip, city, address, region, email)
         VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
         ON CONFLICT (order_uid) DO UPDATE SET
            name = EXCLUDED.name,
            phone = EXCLUDED.phone,
            zip = EXCLUDED.zip,
            city = EXCLUDED.city,
            address = EXCLUDED.address,
            region = EXCLUDED.region,
            email = EXCLUDED.email`,
		o.OrderUID, o.Delivery.Name, o.Delivery.Phone, o.Delivery.Zip, o.Delivery.City,
		o.Delivery.Address, o.Delivery.Region, o.Delivery.Email)
	if err != nil {
		log.Error().Err(err).Str("order_uid", o.OrderUID).Msg("Failed to insert delivery")
		return fmt.Errorf("insert delivery: %w", err)
	}
	return nil
}

func (p *Postgres) savePayment(tx pgx.Tx, o *model.Order) error {
	_, err := tx.Exec(context.Background(),
		`INSERT INTO payments (order_uid, transaction, request_id, currency, provider, amount, payment_dt, bank, delivery_cost, goods_total, custom_fee)
         VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
         ON CONFLICT (order_uid) DO UPDATE SET
            transaction = EXCLUDED.transaction,
            request_id = EXCLUDED.request_id,
            currency = EXCLUDED.currency,
            provider = EXCLUDED.provider,
            amount = EXCLUDED.amount,
            payment_dt = EXCLUDED.payment_dt,
            bank = EXCLUDED.bank,
            delivery_cost = EXCLUDED.delivery_cost,
            goods_total = EXCLUDED.goods_total,
            custom_fee = EXCLUDED.custom_fee`,
		o.OrderUID, o.Payment.Transaction, o.Payment.RequestID, o.Payment.Currency, o.Payment.Provider,
		o.Payment.Amount, o.Payment.PaymentDt, o.Payment.Bank, o.Payment.DeliveryCost,
		o.Payment.GoodsTotal, o.Payment.CustomFee)
	if err != nil {
		log.Error().Err(err).Str("order_uid", o.OrderUID).Msg("Failed to insert payment")
		return fmt.Errorf("insert payment: %w", err)
	}
	return nil
}

func (p *Postgres) saveItems(tx pgx.Tx, o *model.Order) error {
	_, err := tx.Exec(context.Background(), `DELETE FROM items WHERE order_uid = $1`, o.OrderUID)
	if err != nil {
		log.Error().Err(err).Str("order_uid", o.OrderUID).Msg("Failed to delete old items")
		return fmt.Errorf("delete items: %w", err)
	}
	for _, item := range o.Items {
		_, err = tx.Exec(context.Background(),
			`INSERT INTO items (order_uid, chrt_id, track_number, price, rid, name, sale, size, total_price, nm_id, brand, status)
             VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`,
			o.OrderUID, item.ChrtID, item.TrackNumber, item.Price, item.Rid, item.Name,
			item.Sale, item.Size, item.TotalPrice, item.NmID, item.Brand, item.Status)
		if err != nil {
			log.Error().Err(err).Str("order_uid", o.OrderUID).Msg("Failed to insert item")
			return fmt.Errorf("insert item: %w", err)
		}
	}
	return nil
}

func (p *Postgres) SaveOrder(o *model.Order) error {
	tx, err := p.pool.Begin(context.Background())

	if err != nil {
		log.Error().Err(err).Str("order_uid", o.OrderUID).Msg("Failed to begin transaction")
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(context.Background())

	if err := p.saveOrderBase(tx, o); err != nil {
		return err
	}

	err = tx.Commit(context.Background())
	if err != nil {
		log.Error().Err(err).Str("order_uid", o.OrderUID).Msg("Failed to commit transaction")
		return fmt.Errorf("commit: %w", err)
	}

	log.Info().Str("order_uid", o.OrderUID).Msg("Order saved")

	return nil
}

func (p *Postgres) Close() {
	p.pool.Close()
}
