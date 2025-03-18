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
		Async:        config.AsyncProducer, // Use the configuration value
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

	// If async is enabled, use WriteMessages directly without retry handling
	// as the kafka-go library will handle retries internally for async mode
	if p.config.AsyncProducer {
		return p.writer.WriteMessages(ctx, msg)
	}

	// Synchronous mode with retries and backoff
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

// ProduceAsync sends a message to Kafka asynchronously
// This method doesn't wait for confirmation and returns immediately
func (p *Producer) ProduceAsync(ctx context.Context, key, value []byte) {
	msg := kafka.Message{
		Key:   key,
		Value: value,
		Time:  time.Now(),
	}

	// Write message asynchronously
	go func() {
		if err := p.writer.WriteMessages(ctx, msg); err != nil {
			// Log error or handle it as needed
			fmt.Printf("Error in async message production: %v\n", err)
		}
	}()
}

// ProduceBatch sends multiple messages to Kafka with retries and backoff
func (p *Producer) ProduceBatch(ctx context.Context, messages []kafka.Message) error {
	// If async is enabled, use WriteMessages directly without retry handling
	if p.config.AsyncProducer {
		return p.writer.WriteMessages(ctx, messages...)
	}

	// Synchronous mode with retries and backoff
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

// ProduceBatchAsync sends multiple messages to Kafka asynchronously
func (p *Producer) ProduceBatchAsync(ctx context.Context, messages []kafka.Message) {
	// Write messages asynchronously
	go func() {
		if err := p.writer.WriteMessages(ctx, messages...); err != nil {
			// Log error or handle it as needed
			fmt.Printf("Error in async batch production: %v\n", err)
		}
	}()
}

// Close closes the producer
func (p *Producer) Close() error {
	return p.writer.Close()
}
