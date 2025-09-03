package kafka

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/duisenbekovayan/wb_l0/internal/cache"
	model "github.com/duisenbekovayan/wb_l0/internal/models"
	"github.com/duisenbekovayan/wb_l0/internal/storage"
	kafkago "github.com/segmentio/kafka-go"
)

type Consumer struct {
	reader *kafkago.Reader
	store  *storage.PG
	cache  *cache.Store
}

func NewConsumer(brokers []string, topic, group string, pg *storage.PG, c *cache.Store) *Consumer {
	r := kafkago.NewReader(kafkago.ReaderConfig{
		Brokers:  brokers,
		GroupID:  group,
		Topic:    topic,
		MinBytes: 10e3,
		MaxBytes: 10e6,
		MaxWait:  2 * time.Second,
	})
	return &Consumer{reader: r, store: pg, cache: c}
}

func (c *Consumer) Run(ctx context.Context) error {
	for {
		m, err := c.reader.ReadMessage(ctx)
		if err != nil {
			return err
		}
		var o model.Order
		if err := json.Unmarshal(m.Value, &o); err != nil {
			log.Printf("bad message: %v", err)
			continue // игнорим некорректные
		}
		// простейшая валидация
		if o.OrderUID == "" || o.Payment.Transaction == "" {
			log.Printf("invalid data: missing order_uid/transaction")
			continue
		}
		if err := c.store.InsertOrder(ctx, o); err != nil {
			log.Printf("db error: %v", err)
			continue
		}
		c.cache.Set(o)
		log.Printf("stored order %s", o.OrderUID)
	}
}
