package kafka

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/segmentio/kafka-go"
)

// CreateTopic creates a Kafka topic with the specified configuration
func CreateTopic(ctx context.Context, config *KafkaConfig) error {
	// Connect to the first broker to create the topic
	conn, err := kafka.DialContext(ctx, "tcp", config.Brokers[0])
	if err != nil {
		return fmt.Errorf("failed to dial leader: %w", err)
	}
	defer conn.Close()

	// Convert retention period to milliseconds for Kafka config
	retentionMs := int64(config.RetentionPeriod / time.Millisecond)
	retentionBytes := strconv.FormatInt(config.RetentionSize, 10)

	// Create the topic with specified configurations
	topicConfigs := []kafka.ConfigEntry{
		{
			ConfigName:  "retention.ms",
			ConfigValue: strconv.FormatInt(retentionMs, 10),
		},
		{
			ConfigName:  "retention.bytes",
			ConfigValue: retentionBytes,
		},
	}

	err = conn.CreateTopics(kafka.TopicConfig{
		Topic:             config.Topic,
		NumPartitions:     config.NumPartitions,
		ReplicationFactor: config.ReplicationFactor,
		ConfigEntries:     topicConfigs,
	})

	if err != nil {
		return fmt.Errorf("failed to create topic: %w", err)
	}

	return nil
}
