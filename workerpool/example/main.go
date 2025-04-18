package main

import (
	"context"
	"fmt"
	"huba/workerpool"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	// Create worker pool with 5 minimum and 20 maximum workers
	pool := workerpool.NewWorkerPool(5, 20,
		workerpool.WithName("my-service-pool"),
		workerpool.WithQueueCapacity(1000),
		workerpool.WithAutoScaling(),
		workerpool.WithDefaultTaskTimeout(10*time.Second),
	)

	// Start the worker pool
	pool.Start()

	// Handle graceful shutdown
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)

	// Start a goroutine to process results
	go func() {
		for result := range pool.Results() {
			if result.Error != nil {
				log.Printf("Task %s failed: %v", result.TaskID, result.Error)
			} else {
				log.Printf("Task %s completed successfully in %v: %v",
					result.TaskID, result.Duration, result.Value)
			}
		}
	}()

	// Periodically submit tasks
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	// Periodically print stats
	statsTicker := time.NewTicker(5 * time.Second)
	defer statsTicker.Stop()

	taskCount := 0

taskLoop:
	for {
		select {
		case <-shutdown:
			log.Println("Shutdown signal received, stopping worker pool...")
			break taskLoop

		case <-ticker.C:
			// Submit a new task
			taskID := fmt.Sprintf("task-%d", taskCount)
			taskCount++

			// Randomly create tasks with different processing times and success/failure
			duration := time.Duration(rand.Intn(2000)) * time.Millisecond
			shouldFail := rand.Float32() < 0.1 // 10% failure rate

			task := workerpool.Task{
				ID: taskID,
				Execute: func(ctx context.Context) (interface{}, error) {
					// Simulate work
					select {
					case <-ctx.Done():
						return nil, ctx.Err()
					case <-time.After(duration):
						if shouldFail {
							return nil, fmt.Errorf("simulated failure")
						}
						return fmt.Sprintf("processed %s", taskID), nil
					}
				},
			}

			if err := pool.Submit(task); err != nil {
				log.Printf("Failed to submit task: %v", err)
			}

		case <-statsTicker.C:
			stats := pool.Stats()
			fmt.Printf("Worker Pool Stats: active=%d, queue=%d/%d, completed=%d, failed=%d\n",
				stats["active_workers"], stats["queue_size"], stats["queue_capacity"],
				stats["completed_tasks"], stats["failed_tasks"])
		}
	}

	// Optionally wait for all submitted tasks to complete
	pool.StopAndWait()
	log.Println("All tasks completed, worker pool shutdown")
}
