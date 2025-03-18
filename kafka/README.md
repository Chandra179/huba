# Kafka Message Broker

A simple message broker implementation using Kafka and Golang with the following features:

## Features

1. **Retention Configuration**
   - Configurable retention period (time-based)
   - Configurable retention size (bytes-based)

2. **Producer Features**
   - Retry mechanism with configurable retry count
   - Exponential backoff between retries
   - Idempotent message delivery to prevent duplicates
   - Synchronous and asynchronous message production

3. **Consumer Features**
   - Manual and automatic offset commit (acknowledgment)
   - Configurable commit interval for auto-commit mode
   - Synchronous and asynchronous message consumption
   - Configurable concurrency for parallel message processing

## Usage

### Configuration

```go
// Create Kafka configuration
config := kafka.NewDefaultConfig()
config.Topic = "my-topic"
config.RetentionPeriod = 24 * time.Hour    // 24 hour retention
config.RetentionSize = 1024 * 1024 * 100   // 100 MB retention
config.MaxRetries = 3                      // 3 retries
config.RetryBackoff = 1 * time.Second      // 1s backoff
config.EnableIdempotence = true            // Enable idempotent producer
config.GroupID = "my-consumer-group"
config.AutoCommit = true                   // Enable auto commit
config.CommitInterval = 5 * time.Second    // Commit every 5 seconds

// Async configuration
config.AsyncProducer = true                // Enable async producer
config.AsyncConsumer = true                // Enable async consumer
config.ConsumerConcurrency = 5             // 5 concurrent message processors
```

### Creating a Topic

```go
// Create the topic with retention settings
if err := kafka.CreateTopic(ctx, config); err != nil {
    log.Fatalf("Failed to create topic: %v", err)
}
```

### Synchronous Producer Example

```go
// Create producer
p := kafka.NewProducer(config)
defer p.Close()

// Produce a message synchronously
key := []byte("message-key")
value := []byte("message-value")
if err := p.Produce(ctx, key, value); err != nil {
    log.Printf("Error producing message: %v", err)
}
```

### Asynchronous Producer Example

```go
// Create producer
p := kafka.NewProducer(config)
defer p.Close()

// Produce a message asynchronously
key := []byte("message-key")
value := []byte("message-value")
p.ProduceAsync(ctx, key, value)

// Or produce batch asynchronously
messages := []kafka.Message{
    {Key: []byte("key1"), Value: []byte("value1")},
    {Key: []byte("key2"), Value: []byte("value2")},
}
p.ProduceBatchAsync(ctx, messages)
```

### Synchronous Consumer Example

```go
// Create consumer
c := kafka.NewConsumer(config)
defer c.Close()

// Define message handler
handler := func(msg kafka.Message) error {
    log.Printf("Consumed message: %s", string(msg.Value))
    return nil
}

// Start consuming synchronously
if err := c.Consume(ctx, handler); err != nil {
    log.Printf("Error consuming messages: %v", err)
}
```

### Asynchronous Consumer Example

```go
// Create consumer
c := kafka.NewConsumer(config)
defer c.Close()

// Define message handler
handler := func(msg kafka.Message) error {
    log.Printf("Consumed message: %s", string(msg.Value))
    return nil
}

// Start consuming asynchronously with 5 concurrent workers
if err := c.ConsumeAsync(ctx, handler, 5); err != nil {
    log.Printf("Error starting async consumer: %v", err)
}

// Stop consuming when done
// This will be called automatically when c.Close() is called
c.StopConsumeAsync()
```

## Running the Example

The example demonstrates a simple producer and consumer working together:

```bash
go run kafka/example/main.go
```

## Requirements

- Go 1.18 or higher
- Kafka server running (default: localhost:9092)
- github.com/segmentio/kafka-go package 

## Running with Docker Compose

A Docker Compose configuration is provided to easily run Kafka locally:

```bash
# Start Kafka and ZooKeeper
docker-compose up -d

# Check the status
docker-compose ps

# Stop the services
docker-compose down
```

The Docker Compose setup includes:
- ZooKeeper (accessible at localhost:2181)
- Kafka broker (accessible at localhost:9092)
- Kafka UI (accessible at http://localhost:8080)

The Kafka UI provides a web interface to manage topics, view messages, and monitor the Kafka cluster. 