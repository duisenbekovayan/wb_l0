// cmd/producer/main.go
package main

import (
	"context"
	"os"
	"time"

	kafkago "github.com/segmentio/kafka-go"
)

func main() {
	data, _ := os.ReadFile("model.json")
	w := &kafkago.Writer{
		Addr:     kafkago.TCP("localhost:9092"),
		Topic:    "orders",
		Balancer: &kafkago.LeastBytes{},
	}
	defer w.Close()
	_ = w.WriteMessages(context.Background(),
		kafkago.Message{Key: []byte("k1"), Value: data, Time: time.Now()},
	)
}
