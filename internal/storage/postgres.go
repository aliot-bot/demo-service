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
