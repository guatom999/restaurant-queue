package kafka

import (
	"context"
	"log/slog"

	"github.com/segmentio/kafka-go"
)

type KafkaPublisher struct {
	writer *kafka.Writer
}

func NewKafkaPublisher(writer *kafka.Writer) *KafkaPublisher {
	return &KafkaPublisher{writer: writer}
}

func (p *KafkaPublisher) Publish(ctx context.Context, payload []byte) error {
	slog.Info("publishing message to kafka", "topic", p.writer.Topic, "bytes", len(payload))
	return p.writer.WriteMessages(ctx, kafka.Message{Value: payload})
}
