package kafka

import (
	"context"
	"fmt"
	"time"

	"github.com/segmentio/kafka-go"
)

// Producer represents a Kafka producer
type Producer struct {
	writer *kafka.Writer
	config *KafkaConfig
}

// NewProducer creates a new Kafka producer with the given configuration
func NewProducer(config *KafkaConfig) *Producer {
	// Configure the writer with retry and idempotence settings
	writer := &kafka.Writer{
		Addr:         kafka.TCP(config.Brokers...),
		Topic:        config.Topic,
		Balancer:     &kafka.Hash{},
		RequiredAcks: kafka.RequireAll, // Wait for all replicas to acknowledge
		MaxAttempts:  config.MaxRetries,
		Async:        false, // Synchronous by default for reliability
	}

	// Set idempotence if enabled
	if config.EnableIdempotence {
		// Create a transport with a client ID for better idempotence support
		transport := &kafka.Transport{
			ClientID: config.ClientID, // Use the configurable client ID
		}
		writer.Transport = transport
	}

	return &Producer{
		writer: writer,
		config: config,
	}
}

// Produce sends a message to Kafka with retries and backoff
func (p *Producer) Produce(ctx context.Context, key, value []byte) error {
	msg := kafka.Message{
		Key:   key,
		Value: value,
		Time:  time.Now(),
	}

	// Attempt to write with retries and backoff
	var err error
	for attempt := 0; attempt <= p.config.MaxRetries; attempt++ {
		// Try to write the message
		err = p.writer.WriteMessages(ctx, msg)
		if err == nil {
			return nil // Success
		}

		// If this was the last attempt, return the error
		if attempt == p.config.MaxRetries {
			return fmt.Errorf("failed to write message after %d attempts: %w", p.config.MaxRetries, err)
		}

		// Wait before retrying with exponential backoff
		backoff := p.config.RetryBackoff * time.Duration(1<<attempt)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(backoff):
			// Continue to next attempt
		}
	}

	return err
}

// ProduceBatch sends multiple messages to Kafka with retries and backoff
func (p *Producer) ProduceBatch(ctx context.Context, messages []kafka.Message) error {
	// Attempt to write with retries and backoff
	var err error
	for attempt := 0; attempt <= p.config.MaxRetries; attempt++ {
		// Try to write the messages
		err = p.writer.WriteMessages(ctx, messages...)
		if err == nil {
			return nil // Success
		}

		// If this was the last attempt, return the error
		if attempt == p.config.MaxRetries {
			return fmt.Errorf("failed to write batch after %d attempts: %w", p.config.MaxRetries, err)
		}

		// Wait before retrying with exponential backoff
		backoff := p.config.RetryBackoff * time.Duration(1<<attempt)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(backoff):
			// Continue to next attempt
		}
	}

	return err
}

// Close closes the producer
func (p *Producer) Close() error {
	return p.writer.Close()
}
