package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	model "github.com/duisenbekovayan/wb_l0/internal/models"
	_ "github.com/lib/pq"
)

type PG struct{ DB *sql.DB }

func New(dsn string) (*PG, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(10)
	db.SetConnMaxLifetime(time.Hour)
	return &PG{DB: db}, db.Ping()
}

func (p *PG) InsertOrder(ctx context.Context, o model.Order) error {
	tx, err := p.DB.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	var delID int
	err = tx.QueryRowContext(ctx, `
		INSERT INTO deliveries (name,phone,zip,city,address,region,email)
		VALUES ($1,$2,$3,$4,$5,$6,$7) RETURNING id`,
		o.Delivery.Name, o.Delivery.Phone, o.Delivery.Zip, o.Delivery.City,
		o.Delivery.Address, o.Delivery.Region, o.Delivery.Email).Scan(&delID)
	if err != nil {
		return err
	}

	var payID int
	err = tx.QueryRowContext(ctx, `
		INSERT INTO payments (transaction,request_id,currency,provider,amount,payment_dt,bank,delivery_cost,goods_total,custom_fee)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10) RETURNING id`,
		o.Payment.Transaction, o.Payment.RequestID, o.Payment.Currency, o.Payment.Provider,
		o.Payment.Amount, o.Payment.PaymentDT, o.Payment.Bank, o.Payment.DeliveryCost,
		o.Payment.GoodsTotal, o.Payment.CustomFee).Scan(&payID)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO orders (order_uid,track_number,entry,locale,internal_signature,customer_id,delivery_service,shardkey,sm_id,date_created,oof_shard,delivery_id,payment_id)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)
		ON CONFLICT (order_uid) DO NOTHING`,
		o.OrderUID, o.TrackNumber, o.Entry, o.Locale, o.InternalSignature, o.CustomerID,
		o.DeliveryService, o.ShardKey, o.SmID, o.DateCreated, o.OofShard, delID, payID)
	if err != nil {
		return err
	}

	for _, it := range o.Items {
		_, err = tx.ExecContext(ctx, `
			INSERT INTO items (chrt_id,track_number,price,rid,name,sale,size,total_price,nm_id,brand,status,order_uid)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`,
			it.ChrtID, it.TrackNumber, it.Price, it.RID, it.Name, it.Sale, it.Size,
			it.TotalPrice, it.NmID, it.Brand, it.Status, o.OrderUID)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (p *PG) GetOrder(ctx context.Context, uid string) (model.Order, error) {
	var o model.Order
	var delID, payID int

	err := p.DB.QueryRowContext(ctx, `
		SELECT order_uid, track_number, entry, locale, internal_signature, customer_id,
		       delivery_service, shardkey, sm_id, date_created, oof_shard, delivery_id, payment_id
		FROM orders WHERE order_uid=$1`, uid).
		Scan(&o.OrderUID, &o.TrackNumber, &o.Entry, &o.Locale, &o.InternalSignature, &o.CustomerID,
			&o.DeliveryService, &o.ShardKey, &o.SmID, &o.DateCreated, &o.OofShard, &delID, &payID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return o, err
		}
		return o, err
	}

	err = p.DB.QueryRowContext(ctx, `
		SELECT name,phone,zip,city,address,region,email FROM deliveries WHERE id=$1`, delID).
		Scan(&o.Delivery.Name, &o.Delivery.Phone, &o.Delivery.Zip, &o.Delivery.City, &o.Delivery.Address, &o.Delivery.Region, &o.Delivery.Email)
	if err != nil {
		return o, err
	}

	err = p.DB.QueryRowContext(ctx, `
		SELECT transaction,request_id,currency,provider,amount,payment_dt,bank,delivery_cost,goods_total,custom_fee
		FROM payments WHERE id=$1`, payID).
		Scan(&o.Payment.Transaction, &o.Payment.RequestID, &o.Payment.Currency, &o.Payment.Provider,
			&o.Payment.Amount, &o.Payment.PaymentDT, &o.Payment.Bank, &o.Payment.DeliveryCost, &o.Payment.GoodsTotal, &o.Payment.CustomFee)
	if err != nil {
		return o, err
	}

	rows, err := p.DB.QueryContext(ctx, `SELECT chrt_id,track_number,price,rid,name,sale,size,total_price,nm_id,brand,status FROM items WHERE order_uid=$1`, uid)
	if err != nil {
		return o, err
	}
	defer rows.Close()
	for rows.Next() {
		var it model.Item
		if err = rows.Scan(&it.ChrtID, &it.TrackNumber, &it.Price, &it.RID, &it.Name, &it.Sale, &it.Size, &it.TotalPrice, &it.NmID, &it.Brand, &it.Status); err != nil {
			return o, err
		}
		o.Items = append(o.Items, it)
	}
	return o, nil
}

// Для прогрева кэша
func (p *PG) LastOrders(ctx context.Context, limit int) ([]model.Order, error) {
	rows, err := p.DB.QueryContext(ctx, `SELECT order_uid FROM orders ORDER BY date_created DESC NULLS LAST LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var res []model.Order
	for rows.Next() {
		var uid string
		if err := rows.Scan(&uid); err != nil {
			return nil, err
		}
		o, err := p.GetOrder(ctx, uid)
		if err != nil {
			return nil, err
		}
		res = append(res, o)
	}
	return res, nil
}

func DSN(host string, port int, user, pass, db string) string {
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable", user, pass, host, port, db)
}
