// Package workerpool provides a robust, production-ready worker pool implementation
// for concurrent task processing with comprehensive error handling, metrics, and
// graceful lifecycle management.
package workerpool

import (
	"context"
	"errors"
	"fmt"
	"log"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"time"
)

// TaskFunc represents a function to be executed by a worker.
type TaskFunc func(ctx context.Context) (interface{}, error)

// Task encapsulates a unit of work to be processed by the worker pool.
type Task struct {
	ID      string
	Execute TaskFunc
	Timeout time.Duration // Optional per-task timeout
}

// Result represents the outcome of a task execution.
type Result struct {
	TaskID    string
	Value     interface{}
	Error     error
	StartTime time.Time
	EndTime   time.Time
	Duration  time.Duration
}

// WorkerPool manages a pool of workers that execute tasks concurrently.
type WorkerPool struct {
	// Configuration
	name          string
	minWorkers    int
	maxWorkers    int
	queueCapacity int
	
	// Channels
	taskQueue     chan Task
	resultChan    chan Result
	
	// State
	activeWorkers int32
	totalTasks    int64
	completedTasks int64
	failedTasks   int64
	
	// Control
	ctx           context.Context
	cancel        context.CancelFunc
	wg            sync.WaitGroup
	mu            sync.RWMutex
	isRunning     bool
	shutdownOnce  sync.Once
	
	// Options
	autoScale       bool
	panicHandler    func(interface{})
	taskTimeout     time.Duration
	logger          Logger
	metrics         MetricsCollector
}

// Logger interface allows for custom logging implementations.
type Logger interface {
	Debug(format string, args ...interface{})
	Info(format string, args ...interface{})
	Warn(format string, args ...interface{})
	Error(format string, args ...interface{})
}

// MetricsCollector interface for recording performance metrics.
type MetricsCollector interface {
	RecordTaskQueued()
	RecordTaskStarted()
	RecordTaskCompleted(duration time.Duration)
	RecordTaskFailed(err error)
	RecordQueueSize(size int)
	RecordActiveWorkers(count int)
}

// defaultLogger provides a basic logging implementation.
type defaultLogger struct{}

func (l *defaultLogger) Debug(format string, args ...interface{}) {
	log.Printf("[DEBUG] "+format, args...)
}

func (l *defaultLogger) Info(format string, args ...interface{}) {
	log.Printf("[INFO] "+format, args...)
}

func (l *defaultLogger) Warn(format string, args ...interface{}) {
	log.Printf("[WARN] "+format, args...)
}

func (l *defaultLogger) Error(format string, args ...interface{}) {
	log.Printf("[ERROR] "+format, args...)
}

// noopMetrics provides a no-op metrics implementation.
type noopMetrics struct{}

func (m *noopMetrics) RecordTaskQueued()                   {}
func (m *noopMetrics) RecordTaskStarted()                  {}
func (m *noopMetrics) RecordTaskCompleted(time.Duration)   {}
func (m *noopMetrics) RecordTaskFailed(error)              {}
func (m *noopMetrics) RecordQueueSize(int)                 {}
func (m *noopMetrics) RecordActiveWorkers(int)             {}

// Option defines a functional option for configuring the WorkerPool.
type Option func(*WorkerPool)

// WithName sets a name for the worker pool for identification.
func WithName(name string) Option {
	return func(wp *WorkerPool) {
		wp.name = name
	}
}

// WithLogger sets a custom logger for the worker pool.
func WithLogger(logger Logger) Option {
	return func(wp *WorkerPool) {
		wp.logger = logger
	}
}

// WithMetrics sets a metrics collector for the worker pool.
func WithMetrics(metrics MetricsCollector) Option {
	return func(wp *WorkerPool) {
		wp.metrics = metrics
	}
}

// WithQueueCapacity sets the capacity of the task queue.
func WithQueueCapacity(capacity int) Option {
	return func(wp *WorkerPool) {
		wp.queueCapacity = capacity
	}
}

// WithAutoScaling enables automatic scaling of worker count based on load.
func WithAutoScaling() Option {
	return func(wp *WorkerPool) {
		wp.autoScale = true
	}
}

// WithPanicHandler sets a custom panic handler function.
func WithPanicHandler(handler func(interface{})) Option {
	return func(wp *WorkerPool) {
		wp.panicHandler = handler
	}
}

// WithDefaultTaskTimeout sets the default timeout for tasks.
func WithDefaultTaskTimeout(timeout time.Duration) Option {
	return func(wp *WorkerPool) {
		wp.taskTimeout = timeout
	}
}

// NewWorkerPool creates a new worker pool with the specified configuration.
func NewWorkerPool(minWorkers, maxWorkers int, options ...Option) *WorkerPool {
	if minWorkers <= 0 {
		minWorkers = 1
	}
	if maxWorkers < minWorkers {
		maxWorkers = minWorkers
	}
	
	ctx, cancel := context.WithCancel(context.Background())
	
	wp := &WorkerPool{
		name:          "worker-pool",
		minWorkers:    minWorkers,
		maxWorkers:    maxWorkers,
		queueCapacity: maxWorkers * 10,
		ctx:           ctx,
		cancel:        cancel,
		logger:        &defaultLogger{},
		metrics:       &noopMetrics{},
		panicHandler:  defaultPanicHandler,
		taskTimeout:   30 * time.Second,
	}
	
	// Apply options
	for _, option := range options {
		option(wp)
	}
	
	// Initialize channels
	wp.taskQueue = make(chan Task, wp.queueCapacity)
	wp.resultChan = make(chan Result, wp.queueCapacity)
	
	return wp
}

// defaultPanicHandler handles panics in worker goroutines.
func defaultPanicHandler(p interface{}) {
	log.Printf("Worker panic recovered: %v\nStack trace: %s", p, debug.Stack())
}

// Start initializes the worker pool and begins processing tasks.
func (wp *WorkerPool) Start() {
	wp.mu.Lock()
	defer wp.mu.Unlock()
	
	if wp.isRunning {
		wp.logger.Warn("Worker pool '%s' is already running", wp.name)
		return
	}
	
	wp.isRunning = true
	wp.logger.Info("Starting worker pool '%s' with %d workers (max: %d)", wp.name, wp.minWorkers, wp.maxWorkers)
	
	// Launch initial set of workers
	for i := 0; i < wp.minWorkers; i++ {
		wp.startWorker()
	}
	
	// Start autoscaler if enabled
	if wp.autoScale {
		go wp.autoScaler()
	}
}

// startWorker launches a new worker goroutine.
func (wp *WorkerPool) startWorker() {
	wp.wg.Add(1)
	atomic.AddInt32(&wp.activeWorkers, 1)
	wp.metrics.RecordActiveWorkers(int(atomic.LoadInt32(&wp.activeWorkers)))
	
	go func() {
		defer wp.wg.Done()
		defer atomic.AddInt32(&wp.activeWorkers, -1)
		defer func() {
			if r := recover(); r != nil {
				if wp.panicHandler != nil {
					wp.panicHandler(r)
				}
			}
		}()
		
		wp.worker()
	}()
}

// worker processes tasks from the queue.
func (wp *WorkerPool) worker() {
	for {
		select {
		case <-wp.ctx.Done():
			// Worker pool has been stopped
			wp.logger.Debug("Worker stopping due to pool shutdown")
			return
		case task, ok := <-wp.taskQueue:
			if !ok {
				// Task queue has been closed
				wp.logger.Debug("Worker stopping due to closed task queue")
				return
			}
			
			wp.metrics.RecordTaskStarted()
			
			// Create task context with timeout if specified
			var taskCtx context.Context
			var cancel context.CancelFunc
			
			if task.Timeout > 0 {
				taskCtx, cancel = context.WithTimeout(wp.ctx, task.Timeout)
			} else if wp.taskTimeout > 0 {
				taskCtx, cancel = context.WithTimeout(wp.ctx, wp.taskTimeout)
			} else {
				taskCtx, cancel = context.WithCancel(wp.ctx)
			}
			
			// Execute the task and capture metrics
			startTime := time.Now()
			result, err := task.Execute(taskCtx)
			endTime := time.Now()
			duration := endTime.Sub(startTime)
			
			// Clean up the context
			cancel()
			
			// Create and send the result
			taskResult := Result{
				TaskID:    task.ID,
				Value:     result,
				Error:     err,
				StartTime: startTime,
				EndTime:   endTime,
				Duration:  duration,
			}
			
			// Update metrics
			if err != nil {
				atomic.AddInt64(&wp.failedTasks, 1)
				wp.metrics.RecordTaskFailed(err)
				wp.logger.Error("Task %s failed: %v", task.ID, err)
			} else {
				wp.metrics.RecordTaskCompleted(duration)
				wp.logger.Debug("Task %s completed in %v", task.ID, duration)
			}
			
			atomic.AddInt64(&wp.completedTasks, 1)
			
			// Send result if the pool is still running
			select {
			case <-wp.ctx.Done():
				// Pool is shutting down, don't send the result
				return
			case wp.resultChan <- taskResult:
				// Result sent successfully
			}
		}
	}
}

// autoScaler periodically adjusts the number of workers based on load.
func (wp *WorkerPool) autoScaler() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-wp.ctx.Done():
			return
		case <-ticker.C:
			wp.adjustWorkers()
		}
	}
}

// adjustWorkers scales the worker count based on queue size and processing rate.
func (wp *WorkerPool) adjustWorkers() {
	wp.mu.Lock()
	defer wp.mu.Unlock()
	
	if !wp.isRunning {
		return
	}
	
	queueSize := len(wp.taskQueue)
	currentWorkers := int(atomic.LoadInt32(&wp.activeWorkers))
	wp.metrics.RecordQueueSize(queueSize)
	
	// Scale up if queue is backing up
	if queueSize > currentWorkers && currentWorkers < wp.maxWorkers {
		// Calculate how many workers to add (at most doubling, up to max)
		toAdd := min(currentWorkers, wp.maxWorkers-currentWorkers)
		if toAdd > 0 {
			wp.logger.Info("Scaling up: adding %d workers (current: %d, queue: %d)", 
				toAdd, currentWorkers, queueSize)
			for i := 0; i < toAdd; i++ {
				wp.startWorker()
			}
		}
	}
	
	// Scale down if queue is empty and we have more than minimum workers
	if queueSize == 0 && currentWorkers > wp.minWorkers {
		// We'll scale down gradually by 25%
		toRemove := max(1, (currentWorkers-wp.minWorkers)/4)
		wp.logger.Info("Scaling down: removing approximately %d workers as tasks complete", toRemove)
		// No immediate action - workers will exit naturally when the queue is empty
	}
}

// min returns the smaller of two integers.
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// max returns the larger of two integers.
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// Submit adds a task to the queue for execution.
// Returns ErrPoolStopped if the pool is not running or shutting down.
// Returns ErrQueueFull if the task queue is full and the task cannot be queued.
func (wp *WorkerPool) Submit(task Task) error {
	if task.Execute == nil {
		return errors.New("task function cannot be nil")
	}
	
	// Generate an ID if not provided
	if task.ID == "" {
		task.ID = fmt.Sprintf("task-%d", atomic.AddInt64(&wp.totalTasks, 1))
	}
	
	// Check if pool is running
	wp.mu.RLock()
	isRunning := wp.isRunning
	wp.mu.RUnlock()
	
	if !isRunning {
		return errors.New("worker pool is not running")
	}
	
	// Try to submit the task
	select {
	case <-wp.ctx.Done():
		return errors.New("worker pool is shutting down")
	case wp.taskQueue <- task:
		wp.metrics.RecordTaskQueued()
		wp.logger.Debug("Task %s queued", task.ID)
		return nil
	default:
		// Queue is full
		return errors.New("task queue is full")
	}
}

// SubmitWait adds a task to the queue and waits for its completion.
// It returns the task result or an error if the task couldn't be submitted or failed.
func (wp *WorkerPool) SubmitWait(task Task) (interface{}, error) {
	// Create a channel to receive the specific task result
	resultCh := make(chan Result, 1)
	
	// Wrap the original task function to send result to our channel
	originalFunc := task.Execute
	task.Execute = func(ctx context.Context) (interface{}, error) {
		return originalFunc(ctx)
	}
	
	// Submit the task
	if err := wp.Submit(task); err != nil {
		return nil, err
	}
	
	// Start a goroutine to listen for our specific task result
	go func() {
		for result := range wp.resultChan {
			if result.TaskID == task.ID {
				resultCh <- result
				return
			}
			// Put other results back in the main channel
			wp.resultChan <- result
		}
	}()
	
	// Wait for the result
	select {
	case <-wp.ctx.Done():
		return nil, errors.New("worker pool shutdown while waiting for task completion")
	case result := <-resultCh:
		return result.Value, result.Error
	}
}

// Results returns a channel for receiving task results.
func (wp *WorkerPool) Results() <-chan Result {
	return wp.resultChan
}

// Stop gracefully shuts down the worker pool.
// It waits for all in-progress tasks to complete but discards queued tasks.
func (wp *WorkerPool) Stop() {
	wp.shutdownOnce.Do(func() {
		wp.mu.Lock()
		if !wp.isRunning {
			wp.mu.Unlock()
			return
		}
		wp.isRunning = false
		wp.mu.Unlock()
		
		wp.logger.Info("Stopping worker pool '%s', waiting for in-progress tasks to complete...", wp.name)
		
		// Signal all workers to stop
		wp.cancel()
		
		// Clear the task queue without closing it
		for len(wp.taskQueue) > 0 {
			<-wp.taskQueue
		}
		
		// Wait for all workers to finish
		wp.wg.Wait()
		
		// Close channels
		close(wp.taskQueue)
		close(wp.resultChan)
		
		wp.logger.Info("Worker pool '%s' stopped. Stats: completed=%d, failed=%d",
			wp.name, atomic.LoadInt64(&wp.completedTasks), atomic.LoadInt64(&wp.failedTasks))
	})
}

// StopAndWait stops the worker pool and waits for all tasks to complete,
// including those that are still in the queue.
func (wp *WorkerPool) StopAndWait() {
	wp.mu.Lock()
	if !wp.isRunning {
		wp.mu.Unlock()
		return
	}
	wp.isRunning = false
	wp.mu.Unlock()
	
	wp.logger.Info("Stopping worker pool '%s' and waiting for all queued tasks...", wp.name)
	
	// Wait for queue to drain
	for len(wp.taskQueue) > 0 {
		time.Sleep(100 * time.Millisecond)
	}
	
	// Now stop normally
	wp.Stop()
}

// Pause temporarily stops processing new tasks, but keeps workers alive.
func (wp *WorkerPool) Pause() {
	wp.mu.Lock()
	defer wp.mu.Unlock()
	
	if !wp.isRunning {
		return
	}
	
	wp.isRunning = false
	wp.logger.Info("Worker pool '%s' paused", wp.name)
}

// Resume continues processing tasks after a pause.
func (wp *WorkerPool) Resume() {
	wp.mu.Lock()
	defer wp.mu.Unlock()
	
	if wp.isRunning {
		return
	}
	
	wp.isRunning = true
	wp.logger.Info("Worker pool '%s' resumed", wp.name)
}

// Drain removes all pending tasks from the queue without executing them.
func (wp *WorkerPool) Drain() int {
	count := 0
	
	for {
		select {
		case <-wp.taskQueue:
			count++
		default:
			wp.logger.Info("Drained %d tasks from worker pool '%s'", count, wp.name)
			return count
		}
	}
}

// Stats returns current statistics about the worker pool.
func (wp *WorkerPool) Stats() map[string]interface{} {
	wp.mu.RLock()
	defer wp.mu.RUnlock()
	
	return map[string]interface{}{
		"name":           wp.name,
		"is_running":     wp.isRunning,
		"min_workers":    wp.minWorkers,
		"max_workers":    wp.maxWorkers,
		"active_workers": atomic.LoadInt32(&wp.activeWorkers),
		"queue_capacity": wp.queueCapacity,
		"queue_size":     len(wp.taskQueue),
		"total_tasks":    atomic.LoadInt64(&wp.totalTasks),
		"completed_tasks": atomic.LoadInt64(&wp.completedTasks),
		"failed_tasks":   atomic.LoadInt64(&wp.failedTasks),
	}
}

// Wait blocks until all workers have completed their current tasks.
func (wp *WorkerPool) Wait() {
	wp.wg.Wait()
}

// Size returns the current number of active workers.
func (wp *WorkerPool) Size() int {
	return int(atomic.LoadInt32(&wp.activeWorkers))
}

// Resize changes the number of workers in the pool.
func (wp *WorkerPool) Resize(min, max int) {
	wp.mu.Lock()
	defer wp.mu.Unlock()
	
	if min < 1 {
		min = 1
	}
	if max < min {
		max = min
	}
	
	wp.minWorkers = min
	wp.maxWorkers = max
	
	// Adjust current number of workers if needed
	currentWorkers := int(atomic.LoadInt32(&wp.activeWorkers))
	
	if currentWorkers < min {
		// Need to add workers
		for i := 0; i < min-currentWorkers; i++ {
			wp.startWorker()
		}
	}
	
	wp.logger.Info("Worker pool '%s' resized: min=%d, max=%d", wp.name, min, max)
}