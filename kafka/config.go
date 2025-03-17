package kafka

import (
	"time"
)

// KafkaConfig holds the configuration for Kafka broker
type KafkaConfig struct {
	// Broker addresses
	Brokers []string

	// Topic configuration
	Topic             string
	NumPartitions     int
	ReplicationFactor int

	// Retention configuration
	RetentionPeriod time.Duration // Retention period in time
	RetentionSize   int64         // Retention size in bytes

	// Producer configuration
	MaxRetries        int           // Number of retries for producer
	RetryBackoff      time.Duration // Backoff time between retries
	EnableIdempotence bool          // Enable idempotent producer
	ClientID          string        // Client ID for the producer

	// Consumer configuration
	GroupID        string        // Consumer group ID
	AutoCommit     bool          // Auto commit offsets
	CommitInterval time.Duration // Commit interval for manual commits
}

// NewDefaultConfig returns a default configuration
func NewDefaultConfig() *KafkaConfig {
	return &KafkaConfig{
		Brokers:           []string{"localhost:9092"},
		NumPartitions:     3,
		ReplicationFactor: 1,
		RetentionPeriod:   24 * time.Hour,    // 24 hours retention by default
		RetentionSize:     1024 * 1024 * 100, // 100 MB retention by default
		MaxRetries:        3,
		RetryBackoff:      time.Second * 2,
		EnableIdempotence: true,
		ClientID:          "kafka-go-producer",
		GroupID:           "default-consumer-group",
		AutoCommit:        false,
		CommitInterval:    time.Second * 5,
	}
}
