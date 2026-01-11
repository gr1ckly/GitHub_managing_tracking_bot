package kafka

import (
	"context"
	"encoding/json"
	"rep_tracker/pkg/dto"
	"time"

	"github.com/segmentio/kafka-go"
)

type KafkaNotificationWriterConfig struct {
	Addr         []string
	Topic        string
	MaxAttempts  int
	BatchSize    int
	BatchTimeout time.Duration
	WriteTimeout time.Duration
}

type KafkaNotificationWriter struct {
	writer *kafka.Writer
}

func NewKafkaNotificationWriter(config KafkaNotificationWriterConfig) (*KafkaNotificationWriter, error) {
	writer := &kafka.Writer{
		Addr:         kafka.TCP(config.Addr...),
		Balancer:     &kafka.LeastBytes{},
		Topic:        config.Topic,
		RequiredAcks: kafka.RequireAll,
		MaxAttempts:  config.MaxAttempts,
		BatchSize:    config.BatchSize,
		BatchTimeout: config.BatchTimeout,
		WriteTimeout: config.WriteTimeout,
	}

	return &KafkaNotificationWriter{writer}, nil
}

func (kw *KafkaNotificationWriter) WriteNotification(ctx context.Context, chatId string, dto *dto.ChangingDTO) error {
	dtoBytes, err := json.Marshal(dto)
	if err != nil {
		return err
	}
	return kw.writer.WriteMessages(ctx, kafka.Message{
		Key:   []byte(chatId),
		Value: dtoBytes,
	})
}

func (kw *KafkaNotificationWriter) Close() error {
	return kw.writer.Close()
}
