package kafka

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"time"

	kafkago "github.com/segmentio/kafka-go"

	"github.com/duisenbekovayan/wb_l0/internal/cache"
	model "github.com/duisenbekovayan/wb_l0/internal/models"
	"github.com/duisenbekovayan/wb_l0/internal/storage"
)

type Consumer struct {
	reader *kafkago.Reader
	dlq    *kafkago.Writer // optional: dead-letter queue
	store  *storage.PG
	cache  *cache.Store
}

type Config struct {
	Brokers          []string
	Topic            string
	GroupID          string
	DLQTopic         string // "" => не использовать DLQ
	MinBytes         int
	MaxBytes         int
	MaxWait          time.Duration
	ReadErrorBackoff time.Duration
}

func NewConsumer(cfg Config, pg *storage.PG, c *cache.Store) *Consumer {
	if cfg.MinBytes == 0 {
		cfg.MinBytes = 10e3
	}
	if cfg.MaxBytes == 0 {
		cfg.MaxBytes = 10e6
	}
	if cfg.MaxWait == 0 {
		cfg.MaxWait = 2 * time.Second
	}
	if cfg.ReadErrorBackoff == 0 {
		cfg.ReadErrorBackoff = 2 * time.Second
	}

	r := kafkago.NewReader(kafkago.ReaderConfig{
		Brokers:  cfg.Brokers,
		Topic:    cfg.Topic,
		GroupID:  cfg.GroupID,
		MinBytes: cfg.MinBytes,
		MaxBytes: cfg.MaxBytes,
		MaxWait:  cfg.MaxWait,
		// ВАЖНО: отключаем фоновые автокоммиты — коммитим вручную
		CommitInterval: 0,
	})
	var w *kafkago.Writer
	if cfg.DLQTopic != "" {
		w = &kafkago.Writer{
			Addr:     kafkago.TCP(cfg.Brokers...),
			Topic:    cfg.DLQTopic,
			Balancer: &kafkago.LeastBytes{},
		}
	}

	return &Consumer{
		reader: r,
		dlq:    w,
		store:  pg,
		cache:  c,
	}
}

func (c *Consumer) Close() error {
	var err1, err2 error
	if c.reader != nil {
		err1 = c.reader.Close()
	}
	if c.dlq != nil {
		err2 = c.dlq.Close()
	}
	if err1 != nil {
		return err1
	}
	return err2
}

func (c *Consumer) Run(ctx context.Context) error {
	log.Printf("Kafka consumer started: topic=%s group=%s", c.reader.Config().Topic, c.reader.Config().GroupID)

	for {
		m, err := c.reader.FetchMessage(ctx) // без автокоммита
		if err != nil {
			// Завершаем тихо при отмене контекста
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return nil
			}
			log.Printf("kafka fetch error: %v (retry in %s)", err, 2*time.Second)
			time.Sleep(2 * time.Second)
			continue
		}

		if err := c.processMessage(ctx, m); err != nil {
			// НЕ коммитим оффсет — сообщение вернётся позже
			log.Printf("process error (offset %d): %v", m.Offset, err)
			// Небольшая пауза чтобы не крутить CPU на проблемном сообщении
			time.Sleep(500 * time.Millisecond)
			continue
		}

		// Сообщение успешно обработано → подтверждаем
		if err := c.reader.CommitMessages(ctx, m); err != nil {
			log.Printf("commit error (offset %d): %v", m.Offset, err)
		}
	}
}

func (c *Consumer) processMessage(ctx context.Context, m kafkago.Message) error {
	var o model.Order

	// 1) JSON + базовая валидация
	if err := json.Unmarshal(m.Value, &o); err != nil {
		log.Printf("invalid json at offset %d: %v", m.Offset, err)
		// переносим в DLQ, если настроен; и коммитим исходное сообщение,
		// чтобы не зациклиться на вечном мусоре
		if c.dlq != nil {
			_ = c.dlq.WriteMessages(ctx, kafkago.Message{
				Key:   m.Key,
				Value: m.Value,
				Time:  time.Now(),
			})
			// сигнал вызывающему слою: всё хорошо, можно коммитить
			return nil
		}
		// иначе — вернём ошибку и сообщение прочитаем снова (может, это временная помеха)
		return err
	}
	if o.OrderUID == "" || o.Payment.Transaction == "" {
		// бизнес-валидация
		log.Printf("invalid order at offset %d: missing order_uid/transaction", m.Offset)
		// считаем мусором → в DLQ если есть, иначе лог и OK
		if c.dlq != nil {
			_ = c.dlq.WriteMessages(ctx, kafkago.Message{
				Key:   m.Key,
				Value: m.Value,
				Time:  time.Now(),
			})
			return nil
		}
		// игнорируем и коммитим (возвращаем nil, чтобы не застревать)
		return nil
	}

	// 2) Транзакционная запись в БД
	if err := c.store.InsertOrder(ctx, o); err != nil {
		// это временная ошибка (БД недоступна/уникальный конфликт и т.п.) —
		// не подтверждаем оффсет, чтобы перечитать позже
		return err
	}

	// 3) Обновляем кэш (идемпотентно)
	c.cache.Set(o)

	log.Printf("stored order %s (offset %d, partition %d)", o.OrderUID, m.Offset, m.Partition)
	return nil
}
