package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/joho/godotenv"

	"github.com/duisenbekovayan/wb_l0/internal/cache"
	"github.com/duisenbekovayan/wb_l0/internal/httpapi"
	"github.com/duisenbekovayan/wb_l0/internal/kafka"
	"github.com/duisenbekovayan/wb_l0/internal/storage"
)

func main() {
	_ = godotenv.Load()

	// Postgres
	port, _ := strconv.Atoi(getenv("PG_PORT", "5432"))
	dsn := storage.DSN(
		getenv("PG_HOST", "localhost"),
		port,
		getenv("PG_USER", "wb"),
		getenv("PG_PASSWORD", "wb"),
		getenv("PG_DB", "wb_orders"),
	)
	pg, err := storage.New(dsn)
	if err != nil {
		log.Fatal(err)
	}

	// Cache
	c := cache.New()

	// Warm-up cache
	last, _ := pg.LastOrders(context.Background(), 50)
	for _, o := range last {
		c.Set(o)
	}

	// HTTP
	api := httpapi.New(getenv("HTTP_ADDR", ":8080"), c, pg)
	log.Printf("Connecting to Kafka broker: %s", getenv("KAFKA_BROKER", "localhost:9092"))
	// Kafka consumer
	broker := getenv("KAFKA_BROKER", "localhost:9092")
	topic := getenv("KAFKA_TOPIC", "orders")
	group := getenv("KAFKA_GROUP", "orders-consumer")
	cons := kafka.NewConsumer(
		kafka.Config{
			Brokers:  []string{broker},
			Topic:    topic,
			GroupID:  group,
			DLQTopic: "orders_dlq", // или "" чтобы отключить DLQ
		},
		pg, c,
	)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		if err := cons.Run(ctx); err != nil {
			log.Printf("consumer stopped: %v", err)
		}
	}()
	defer func() {
		cancel()
		_ = cons.Close()
	}()
	go func() { _ = api.Start() }()

	// graceful shutdown
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
	cancel()
	time.Sleep(300 * time.Millisecond)
	_ = api.Shutdown(context.Background())
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
