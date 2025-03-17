package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"securedesign/kafka"
	"sync"
	"syscall"
	"time"

	kafkago "github.com/segmentio/kafka-go"
)

func main() {
	// Create a context that will be canceled on SIGINT or SIGTERM
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Set up signal handling
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-signals
		log.Println("Received shutdown signal")
		cancel()
	}()

	// Create Kafka configuration
	config := kafka.NewDefaultConfig()
	config.Topic = "example-topic"
	config.RetentionPeriod = 1 * time.Hour       // 1 hour retention
	config.RetentionSize = 1024 * 1024 * 50      // 50 MB retention
	config.MaxRetries = 5                        // 5 retries
	config.RetryBackoff = 500 * time.Millisecond // 500ms backoff
	config.EnableIdempotence = true              // Enable idempotent producer
	config.GroupID = "example-consumer-group"
	config.AutoCommit = true                // Enable auto commit
	config.CommitInterval = 5 * time.Second // Commit every 5 seconds

	// Create the topic
	log.Println("Creating topic:", config.Topic)
	if err := kafka.CreateTopic(ctx, config); err != nil {
		log.Printf("Warning: Failed to create topic: %v (topic may already exist)", err)
	}

	// Create a wait group to wait for producer and consumer to finish
	var wg sync.WaitGroup

	// Start producer
	wg.Add(1)
	go func() {
		defer wg.Done()
		runProducer(ctx, config)
	}()

	// Start consumer
	wg.Add(1)
	go func() {
		defer wg.Done()
		runConsumer(ctx, config)
	}()

	// Wait for producer and consumer to finish
	wg.Wait()
	log.Println("Example completed")
}

func runProducer(ctx context.Context, config *kafka.KafkaConfig) {
	// Create producer
	p := kafka.NewProducer(config)
	defer p.Close()

	// Produce 10 messages
	for i := 0; i < 10; i++ {
		select {
		case <-ctx.Done():
			log.Println("Producer shutting down")
			return
		default:
			// Continue
		}

		// Create message
		key := []byte(fmt.Sprintf("key-%d", i))
		value := []byte(fmt.Sprintf("message-%d", i))

		// Produce message
		log.Printf("Producing message: %s", value)
		if err := p.Produce(ctx, key, value); err != nil {
			log.Printf("Error producing message: %v", err)
			continue
		}

		// Wait a bit before sending the next message
		time.Sleep(1 * time.Second)
	}

	log.Println("Producer finished")
}

func runConsumer(ctx context.Context, config *kafka.KafkaConfig) {
	// Create consumer
	c := kafka.NewConsumer(config)
	defer c.Close()

	// Define message handler
	handler := func(msg kafkago.Message) error {
		log.Printf("Consumed message: key=%s, value=%s, partition=%d, offset=%d",
			string(msg.Key), string(msg.Value), msg.Partition, msg.Offset)
		return nil
	}

	// Start consuming
	log.Println("Consumer started")
	if err := c.Consume(ctx, handler); err != nil && err != context.Canceled {
		log.Printf("Error consuming messages: %v", err)
	}

	log.Println("Consumer finished")
}
