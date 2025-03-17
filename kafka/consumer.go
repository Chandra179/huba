package kafka

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/segmentio/kafka-go"
)

// MessageHandler is a function that processes a Kafka message
type MessageHandler func(msg kafka.Message) error

// Consumer represents a Kafka consumer
type Consumer struct {
	reader        *kafka.Reader
	config        *KafkaConfig
	commitMutex   sync.Mutex
	uncommitted   []kafka.Message
	lastCommit    time.Time
	stopCommit    chan struct{}
	commitWg      sync.WaitGroup
	autoCommitter bool
}

// NewConsumer creates a new Kafka consumer with the given configuration
func NewConsumer(config *KafkaConfig) *Consumer {
	// Configure the reader
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:     config.Brokers,
		Topic:       config.Topic,
		GroupID:     config.GroupID,
		MinBytes:    10e3, // 10KB
		MaxBytes:    10e6, // 10MB
		StartOffset: kafka.FirstOffset,
		// Disable auto commit, we'll handle it manually
		CommitInterval: 0,
	})

	consumer := &Consumer{
		reader:        reader,
		config:        config,
		uncommitted:   make([]kafka.Message, 0),
		lastCommit:    time.Now(),
		stopCommit:    make(chan struct{}),
		autoCommitter: config.AutoCommit,
	}

	// Start auto-commit goroutine if enabled
	if config.AutoCommit {
		consumer.commitWg.Add(1)
		go consumer.autoCommitLoop()
	}

	return consumer
}

// autoCommitLoop periodically commits offsets if auto-commit is enabled
func (c *Consumer) autoCommitLoop() {
	defer c.commitWg.Done()
	ticker := time.NewTicker(c.config.CommitInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.commitOffsets(context.Background())
		case <-c.stopCommit:
			return
		}
	}
}

// Consume reads and processes messages from Kafka
func (c *Consumer) Consume(ctx context.Context, handler MessageHandler) error {
	for {
		// Check if context is done
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			// Continue processing
		}

		// Read message
		msg, err := c.reader.FetchMessage(ctx)
		if err != nil {
			return fmt.Errorf("error fetching message: %w", err)
		}

		// Process message with handler
		err = handler(msg)
		if err != nil {
			return fmt.Errorf("error handling message: %w", err)
		}

		// Add to uncommitted messages
		c.commitMutex.Lock()
		c.uncommitted = append(c.uncommitted, msg)
		c.commitMutex.Unlock()

		// If not using auto-commit, commit immediately
		if !c.autoCommitter {
			if err := c.commitOffsets(ctx); err != nil {
				return fmt.Errorf("error committing offsets: %w", err)
			}
		}
	}
}

// commitOffsets commits the current offsets to Kafka
func (c *Consumer) commitOffsets(ctx context.Context) error {
	c.commitMutex.Lock()
	defer c.commitMutex.Unlock()

	// If no uncommitted messages, return
	if len(c.uncommitted) == 0 {
		return nil
	}

	// Commit all uncommitted messages
	if err := c.reader.CommitMessages(ctx, c.uncommitted...); err != nil {
		return err
	}

	// Clear uncommitted messages and update last commit time
	c.uncommitted = make([]kafka.Message, 0)
	c.lastCommit = time.Now()
	return nil
}

// Close stops the consumer and commits any remaining offsets
func (c *Consumer) Close() error {
	// Stop auto-commit goroutine if running
	if c.autoCommitter {
		close(c.stopCommit)
		c.commitWg.Wait()
	}

	// Commit any remaining offsets
	if err := c.commitOffsets(context.Background()); err != nil {
		return fmt.Errorf("error committing final offsets: %w", err)
	}

	// Close the reader
	return c.reader.Close()
}
