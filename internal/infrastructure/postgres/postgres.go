package postgres

import (
	"context"
	"demo-service/internal/infrastructure/cache"
	"demo-service/internal/model"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Postgres struct {
	pool *pgxpool.Pool
}

// Создание нового подключения к бд
func New(ctx context.Context, dsn string) (*Postgres, error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("ошибка подключения: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("не удалось пропинговать базу: %w", err)
	}
	return &Postgres{pool: pool}, nil
}

// сохранение заказ в бд с использованием транзакции
func (p *Postgres) SaveOrder(o *model.Order, c *cache.Cache) error {
	ctx := context.Background()
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("не удалось начать транзакцию: %w", err)
	}

	// вставка или обновления заказа
	_, err = tx.Exec(ctx, `INSERT INTO orders (order_uid, track_number, entry, locale, internal_signature, customer_id, delivery_service, shardkey, sm_id, date_created, oof_shard)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
		ON CONFLICT (order_uid) DO UPDATE SET track_number=EXCLUDED.track_number, entry=EXCLUDED.entry, locale=EXCLUDED.locale,
		internal_signature=EXCLUDED.internal_signature, customer_id=EXCLUDED.customer_id,
		delivery_service=EXCLUDED.delivery_service, shardkey=EXCLUDED.shardkey, sm_id=EXCLUDED.sm_id,
		date_created=EXCLUDED.date_created, oof_shard=EXCLUDED.oof_shard`,
		o.OrderUID, o.TrackNumber, o.Entry, o.Locale, o.InternalSignature, o.CustomerID,
		o.DeliveryService, o.Shardkey, o.SmID, o.DateCreated, o.OofShard)
	if err != nil {
		tx.Rollback(ctx)
		return fmt.Errorf("ошибка добавления заказа: %w", err)
	}

	// вставка и обвноление данных доставки
	_, err = tx.Exec(ctx, `INSERT INTO deliveries (order_uid, name, phone, zip, city, address, region, email)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
		ON CONFLICT (order_uid) DO UPDATE SET name=EXCLUDED.name, phone=EXCLUDED.phone, zip=EXCLUDED.zip, city=EXCLUDED.city,
		address=EXCLUDED.address, region=EXCLUDED.region, email=EXCLUDED.email`,
		o.OrderUID, o.Delivery.Name, o.Delivery.Phone, o.Delivery.Zip, o.Delivery.City,
		o.Delivery.Address, o.Delivery.Region, o.Delivery.Email)
	if err != nil {
		tx.Rollback(ctx)
		return fmt.Errorf("ошибка добавления доставки: %w", err)
	}

	// вставка или обновление данных платежа
	_, err = tx.Exec(ctx, `INSERT INTO payments (order_uid, transaction, request_id, currency, provider, amount, payment_dt, bank, delivery_cost, goods_total, custom_fee)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
		ON CONFLICT (order_uid) DO UPDATE SET transaction=EXCLUDED.transaction, request_id=EXCLUDED.request_id, currency=EXCLUDED.currency,
		provider=EXCLUDED.provider, amount=EXCLUDED.amount, payment_dt=EXCLUDED.payment_dt,
		bank=EXCLUDED.bank, delivery_cost=EXCLUDED.delivery_cost, goods_total=EXCLUDED.goods_total, custom_fee=EXCLUDED.custom_fee`,
		o.OrderUID, o.Payment.Transaction, o.Payment.RequestID, o.Payment.Currency, o.Payment.Provider,
		o.Payment.Amount, o.Payment.PaymentDt, o.Payment.Bank, o.Payment.DeliveryCost,
		o.Payment.GoodsTotal, o.Payment.CustomFee)
	if err != nil {
		tx.Rollback(ctx)
		return fmt.Errorf("ошибка добавления платежа: %w", err)
	}

	// удаление старых элементов заказа
	_, err = tx.Exec(ctx, `DELETE FROM items WHERE order_uid=$1`, o.OrderUID)
	if err != nil {
		tx.Rollback(ctx)
		return fmt.Errorf("ошибка удаления элементов: %w", err)
	}

	// вставка новых элементов заказа
	for _, it := range o.Items {
		_, err = tx.Exec(ctx, `INSERT INTO items (order_uid, chrt_id, track_number, price, rid, name, sale, size, total_price, nm_id, brand, status)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`,
			o.OrderUID, it.ChrtID, it.TrackNumber, it.Price, it.Rid, it.Name,
			it.Sale, it.Size, it.TotalPrice, it.NmID, it.Brand, it.Status)
		if err != nil {
			tx.Rollback(ctx)
			return fmt.Errorf("ошибка добавления элемента: %w", err)
		}
	}

	// фиксация транзакции
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("ошибка коммита транзакции: %w", err)
	}

	return nil
}

// заказы из бд в кэщ
func (p *Postgres) LoadCache(ctx context.Context, c *cache.Cache) error {
	rows, err := p.pool.Query(ctx, `
		SELECT o.order_uid, o.track_number, o.entry, o.locale, o.internal_signature, o.customer_id,
		       o.delivery_service, o.shardkey, o.sm_id, o.date_created, o.oof_shard,
		       d.name, d.phone, d.zip, d.city, d.address, d.region, d.email,
		       p.transaction, p.request_id, p.currency, p.provider, p.amount, p.payment_dt,
		       p.bank, p.delivery_cost, p.goods_total, p.custom_fee
		FROM orders o
		LEFT JOIN deliveries d ON o.order_uid=d.order_uid
		LEFT JOIN payments p ON o.order_uid=p.order_uid`)
	if err != nil {
		return fmt.Errorf("ошибка загрузки кеша: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var o model.Order
		// считывание данных заказа
		err := rows.Scan(
			&o.OrderUID, &o.TrackNumber, &o.Entry, &o.Locale, &o.InternalSignature,
			&o.CustomerID, &o.DeliveryService, &o.Shardkey, &o.SmID, &o.DateCreated,
			&o.OofShard, &o.Delivery.Name, &o.Delivery.Phone, &o.Delivery.Zip,
			&o.Delivery.City, &o.Delivery.Address, &o.Delivery.Region, &o.Delivery.Email,
			&o.Payment.Transaction, &o.Payment.RequestID, &o.Payment.Currency,
			&o.Payment.Provider, &o.Payment.Amount, &o.Payment.PaymentDt, &o.Payment.Bank,
			&o.Payment.DeliveryCost, &o.Payment.GoodsTotal, &o.Payment.CustomFee)
		if err != nil {
			log.Println("не удалось отсканировать заказ для кеша:", err)
			continue
		}

		itemRows, err := p.pool.Query(ctx, "SELECT chrt_id, track_number, price, rid, name, sale, size, total_price, nm_id, brand, status FROM items WHERE order_uid=$1", o.OrderUID)
		if err != nil {
			log.Println("не удалось загрузить элементы заказа для", o.OrderUID, ":", err)
			continue
		}

		for itemRows.Next() {
			var it model.Item
			if err := itemRows.Scan(&it.ChrtID, &it.TrackNumber, &it.Price, &it.Rid, &it.Name, &it.Sale, &it.Size, &it.TotalPrice, &it.NmID, &it.Brand, &it.Status); err != nil {
				log.Println("не удалось отсканировать элемент для", o.OrderUID, ":", err)
				continue
			}
			o.Items = append(o.Items, it)
		}
		itemRows.Close()
		// сохранение заказа в кэщ
		c.Set(&o)
	}
	return nil
}

// получить заказ из бд по orderUID
func (p *Postgres) GetOrder(ctx context.Context, orderUID string) (*model.Order, error) {
	o := &model.Order{}

	row := p.pool.QueryRow(ctx, "SELECT order_uid, track_number, entry, locale, internal_signature, customer_id, delivery_service, shardkey, sm_id, date_created, oof_shard FROM orders WHERE order_uid=$1", orderUID)
	err := row.Scan(&o.OrderUID, &o.TrackNumber, &o.Entry, &o.Locale, &o.InternalSignature, &o.CustomerID, &o.DeliveryService, &o.Shardkey, &o.SmID, &o.DateCreated, &o.OofShard)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("заказ не найден")
		}
		return nil, fmt.Errorf("ошибка чтения заказа: %w", err)
	}

	row = p.pool.QueryRow(ctx, "SELECT name, phone, zip, city, address, region, email FROM deliveries WHERE order_uid=$1", orderUID)
	err = row.Scan(&o.Delivery.Name, &o.Delivery.Phone, &o.Delivery.Zip, &o.Delivery.City, &o.Delivery.Address, &o.Delivery.Region, &o.Delivery.Email)
	if err != nil && err != pgx.ErrNoRows {
		return nil, fmt.Errorf("ошибка чтения доставки: %w", err)
	}

	row = p.pool.QueryRow(ctx, "SELECT transaction, request_id, currency, provider, amount, payment_dt, bank, delivery_cost, goods_total, custom_fee FROM payments WHERE order_uid=$1", orderUID)
	err = row.Scan(&o.Payment.Transaction, &o.Payment.RequestID, &o.Payment.Currency, &o.Payment.Provider, &o.Payment.Amount, &o.Payment.PaymentDt, &o.Payment.Bank, &o.Payment.DeliveryCost, &o.Payment.GoodsTotal, &o.Payment.CustomFee)
	if err != nil && err != pgx.ErrNoRows {
		return nil, fmt.Errorf("ошибка чтения платежа: %w", err)
	}

	rows, err := p.pool.Query(ctx, "SELECT chrt_id, track_number, price, rid, name, sale, size, total_price, nm_id, brand, status FROM items WHERE order_uid=$1", orderUID)
	if err != nil {
		return nil, fmt.Errorf("ошибка чтения элементов заказа: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var it model.Item
		if err := rows.Scan(&it.ChrtID, &it.TrackNumber, &it.Price, &it.Rid, &it.Name, &it.Sale, &it.Size, &it.TotalPrice, &it.NmID, &it.Brand, &it.Status); err != nil {
			log.Println("не удалось отсканировать элемент:", err)
			continue
		}
		o.Items = append(o.Items, it)
	}

	return o, nil
}

func (p *Postgres) Close() {
	p.pool.Close()
}
