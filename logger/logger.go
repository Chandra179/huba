// Package logger provides a production-grade centralized logging system for Go applications.
// It supports multiple output destinations, structured logging, sampling, rotation,
// and integration with common monitoring systems.
package logger

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Level represents the severity level of a log message.
type Level int

const (
	DebugLevel Level = iota
	InfoLevel
	WarnLevel
	ErrorLevel
	FatalLevel
)

// String returns the string representation of the log level.
func (l Level) String() string {
	switch l {
	case DebugLevel:
		return "DEBUG"
	case InfoLevel:
		return "INFO"
	case WarnLevel:
		return "WARN"
	case ErrorLevel:
		return "ERROR"
	case FatalLevel:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}

// Field represents a key-value pair in a structured log entry.
type Field struct {
	Key   string
	Value interface{}
}

// Entry represents a log entry with metadata.
type Entry struct {
	ID        string                 `json:"id"`
	Timestamp time.Time              `json:"timestamp"`
	Level     string                 `json:"level"`
	Message   string                 `json:"message"`
	Fields    map[string]interface{} `json:"fields,omitempty"`
	Caller    string                 `json:"caller,omitempty"`
	Service   string                 `json:"service"`
	TraceID   string                 `json:"trace_id,omitempty"`
	SpanID    string                 `json:"span_id,omitempty"`
}

// OutputHandler represents a destination for log entries.
type OutputHandler interface {
	// Handle processes a log entry
	Handle(entry Entry) error
	// Close performs any cleanup necessary
	Close() error
}

// ConsoleHandler outputs log entries to the console (stdout or stderr).
type ConsoleHandler struct {
	out       io.Writer
	formatter Formatter
	mu        sync.Mutex
}

// Handle writes the log entry to the console.
func (h *ConsoleHandler) Handle(entry Entry) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	
	bytes, err := h.formatter.Format(entry)
	if err != nil {
		return err
	}
	
	_, err = h.out.Write(bytes)
	if err != nil {
		return err
	}
	
	return nil
}

// Close implements the OutputHandler interface.
func (h *ConsoleHandler) Close() error {
	return nil // Console doesn't need to be closed
}

// NewConsoleHandler creates a new handler that outputs to the console.
func NewConsoleHandler(formatter Formatter) *ConsoleHandler {
	return &ConsoleHandler{
		out:       os.Stdout,
		formatter: formatter,
	}
}

// FileHandler outputs log entries to a file with rotation capabilities.
type FileHandler struct {
	directory     string
	filename      string
	maxFileSize   int64
	maxFiles      int
	currentFile   *os.File
	currentSize   int64
	formatter     Formatter
	mu            sync.Mutex
	fileOpenTime  time.Time
	rotateOnDaily bool
}

// Handle writes the log entry to a file, rotating if necessary.
func (h *FileHandler) Handle(entry Entry) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	
	// Check if we need to rotate based on time
	if h.rotateOnDaily && !isSameDay(h.fileOpenTime, time.Now()) {
		if err := h.rotate(); err != nil {
			return err
		}
	}
	
	// Check if file is open
	if h.currentFile == nil {
		if err := h.openOrCreateFile(); err != nil {
			return err
		}
	}
	
	// Format the entry
	bytes, err := h.formatter.Format(entry)
	if err != nil {
		return err
	}
	
	// Write to file
	_, err = h.currentFile.Write(bytes)
	if err != nil {
		return err
	}
	
	// Update size and check if rotation needed
	h.currentSize += int64(len(bytes))
	if h.maxFileSize > 0 && h.currentSize >= h.maxFileSize {
		return h.rotate()
	}
	
	return nil
}

// openOrCreateFile opens or creates the log file.
func (h *FileHandler) openOrCreateFile() error {
	// Ensure directory exists
	if err := os.MkdirAll(h.directory, 0755); err != nil {
		return err
	}
	
	// Open file
	path := filepath.Join(h.directory, h.filename)
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	
	// Get file info for size
	info, err := file.Stat()
	if err != nil {
		file.Close()
		return err
	}
	
	h.currentFile = file
	h.currentSize = info.Size()
	h.fileOpenTime = time.Now()
	
	return nil
}

// rotate performs log file rotation.
func (h *FileHandler) rotate() error {
	// Close current file if open
	if h.currentFile != nil {
		if err := h.currentFile.Close(); err != nil {
			return err
		}
		h.currentFile = nil
	}
	
	// Rename current file with timestamp
	currentPath := filepath.Join(h.directory, h.filename)
	timestamp := time.Now().Format("20060102-150405")
	newPath := filepath.Join(h.directory, fmt.Sprintf("%s.%s", h.filename, timestamp))
	
	if _, err := os.Stat(currentPath); err == nil {
		if err := os.Rename(currentPath, newPath); err != nil {
			return err
		}
	}
	
	// Delete old files if we have too many
	if h.maxFiles > 0 {
		if err := h.cleanupOldFiles(); err != nil {
			return err
		}
	}
	
	// Open new file
	return h.openOrCreateFile()
}

// cleanupOldFiles removes old log files when we exceed maxFiles.
func (h *FileHandler) cleanupOldFiles() error {
	pattern := filepath.Join(h.directory, h.filename+".*")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return err
	}
	
	if len(matches) <= h.maxFiles {
		return nil
	}
	
	// Sort files by modification time (oldest first)
	type fileInfo struct {
		path    string
		modTime time.Time
	}
	
	files := make([]fileInfo, 0, len(matches))
	for _, path := range matches {
		info, err := os.Stat(path)
		if err != nil {
			continue
		}
		files = append(files, fileInfo{path, info.ModTime()})
	}
	
	// Sort by modification time
	// (In a real implementation, use sort.Slice here)
	// For simplicity, I'm using a basic bubble sort
	for i := 0; i < len(files)-1; i++ {
		for j := i + 1; j < len(files); j++ {
			if files[i].modTime.After(files[j].modTime) {
				files[i], files[j] = files[j], files[i]
			}
		}
	}
	
	// Delete oldest files
	toDelete := len(files) - h.maxFiles
	for i := 0; i < toDelete; i++ {
		if err := os.Remove(files[i].path); err != nil {
			return err
		}
	}
	
	return nil
}

// Close implements the OutputHandler interface.
func (h *FileHandler) Close() error {
	h.mu.Lock()
	defer h.mu.Unlock()
	
	if h.currentFile != nil {
		return h.currentFile.Close()
	}
	return nil
}

// NewFileHandler creates a new handler that outputs to rotating files.
func NewFileHandler(directory, filename string, maxFileSize int64, maxFiles int, formatter Formatter) *FileHandler {
	return &FileHandler{
		directory:     directory,
		filename:      filename,
		maxFileSize:   maxFileSize,
		maxFiles:      maxFiles,
		formatter:     formatter,
		rotateOnDaily: true,
	}
}

// HttpHandler sends log entries to a remote HTTP endpoint.
type HttpHandler struct {
	endpoint  string
	client    *http.Client
	headers   map[string]string
	formatter Formatter
	batchSize int
	maxRetry  int
	buffer    []Entry
	mu        sync.Mutex
	wg        sync.WaitGroup
	flush     chan struct{}
	done      chan struct{}
}

// Handle buffers the log entry and flushes when batch size is reached.
func (h *HttpHandler) Handle(entry Entry) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	
	h.buffer = append(h.buffer, entry)
	
	if len(h.buffer) >= h.batchSize {
		h.wg.Add(1)
		go func(entries []Entry) {
			defer h.wg.Done()
			if err := h.sendEntries(entries); err != nil {
				// In a real implementation, we would have a proper error handling strategy
				fmt.Fprintf(os.Stderr, "Error sending logs: %v\n", err)
			}
		}(append([]Entry{}, h.buffer...))
		
		h.buffer = h.buffer[:0]
	}
	
	return nil
}

// sendEntries sends log entries to the HTTP endpoint.
func (h *HttpHandler) sendEntries(entries []Entry) error {
	payload, err := json.Marshal(entries)
	if err != nil {
		return err
	}
	
	for attempt := 0; attempt <= h.maxRetry; attempt++ {
		req, err := http.NewRequest("POST", h.endpoint, strings.NewReader(string(payload)))
		if err != nil {
			return err
		}
		
		req.Header.Set("Content-Type", "application/json")
		for key, value := range h.headers {
			req.Header.Set(key, value)
		}
		
		resp, err := h.client.Do(req)
		if err != nil {
			if attempt == h.maxRetry {
				return err
			}
			// Exponential backoff
			time.Sleep(time.Duration(1<<uint(attempt)) * 100 * time.Millisecond)
			continue
		}
		
		defer resp.Body.Close()
		
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return nil
		}
		
		if attempt == h.maxRetry {
			return fmt.Errorf("failed after %d attempts, last status code: %d", attempt+1, resp.StatusCode)
		}
		
		// Exponential backoff
		time.Sleep(time.Duration(1<<uint(attempt)) * 100 * time.Millisecond)
	}
	
	return errors.New("unknown error occurred while sending logs")
}

// startFlushWorker starts a background worker to periodically flush logs.
func (h *HttpHandler) startFlushWorker() {
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		
		for {
			select {
			case <-ticker.C:
				h.Flush()
			case <-h.flush:
				h.Flush()
			case <-h.done:
				h.Flush()
				return
			}
		}
	}()
}

// Flush sends any buffered log entries immediately.
func (h *HttpHandler) Flush() {
	h.mu.Lock()
	if len(h.buffer) == 0 {
		h.mu.Unlock()
		return
	}
	
	entries := append([]Entry{}, h.buffer...)
	h.buffer = h.buffer[:0]
	h.mu.Unlock()
	
	h.wg.Add(1)
	go func() {
		defer h.wg.Done()
		if err := h.sendEntries(entries); err != nil {
			fmt.Fprintf(os.Stderr, "Error flushing logs: %v\n", err)
		}
	}()
}

// Close flushes remaining entries and stops the flush worker.
func (h *HttpHandler) Close() error {
	close(h.done)
	h.wg.Wait()
	return nil
}

// NewHttpHandler creates a new handler that sends logs to an HTTP endpoint.
func NewHttpHandler(endpoint string, batchSize, maxRetry int, formatter Formatter) *HttpHandler {
	handler := &HttpHandler{
		endpoint:  endpoint,
		client:    &http.Client{Timeout: 10 * time.Second},
		headers:   make(map[string]string),
		formatter: formatter,
		batchSize: batchSize,
		maxRetry:  maxRetry,
		buffer:    make([]Entry, 0, batchSize),
		flush:     make(chan struct{}),
		done:      make(chan struct{}),
	}
	
	handler.startFlushWorker()
	return handler
}

// Formatter defines the interface for formatting log entries.
type Formatter interface {
	Format(entry Entry) ([]byte, error)
}

// JsonFormatter formats log entries as JSON.
type JsonFormatter struct {
	Pretty bool
}

// Format converts the log entry to JSON.
func (f *JsonFormatter) Format(entry Entry) ([]byte, error) {
	var bytes []byte
	var err error
	
	if f.Pretty {
		bytes, err = json.MarshalIndent(entry, "", "  ")
	} else {
		bytes, err = json.Marshal(entry)
	}
	
	if err != nil {
		return nil, err
	}
	
	return append(bytes, '\n'), nil
}

// TextFormatter formats log entries as human-readable text.
type TextFormatter struct {
	IncludeTimestamp bool
	TimestampFormat  string
	IncludeCaller    bool
}

// Format converts the log entry to text.
func (f *TextFormatter) Format(entry Entry) ([]byte, error) {
	var parts []string
	
	if f.IncludeTimestamp {
		format := f.TimestampFormat
		if format == "" {
			format = time.RFC3339
		}
		parts = append(parts, entry.Timestamp.Format(format))
	}
	
	parts = append(parts, fmt.Sprintf("[%s]", entry.Level))
	
	if entry.Service != "" {
		parts = append(parts, fmt.Sprintf("[%s]", entry.Service))
	}
	
	if f.IncludeCaller && entry.Caller != "" {
		parts = append(parts, fmt.Sprintf("(%s)", entry.Caller))
	}
	
	parts = append(parts, entry.Message)
	
	if len(entry.Fields) > 0 {
		fieldsStr := make([]string, 0, len(entry.Fields))
		for k, v := range entry.Fields {
			fieldsStr = append(fieldsStr, fmt.Sprintf("%s=%v", k, v))
		}
		parts = append(parts, fmt.Sprintf("{%s}", strings.Join(fieldsStr, ", ")))
	}
	
	if entry.TraceID != "" {
		parts = append(parts, fmt.Sprintf("trace=%s", entry.TraceID))
	}
	
	return []byte(strings.Join(parts, " ") + "\n"), nil
}

// Logger represents the main logger instance.
type Logger struct {
	handlers    []OutputHandler
	level       Level
	serviceName string
	mu          sync.RWMutex
	sampleRate  int
	sampling    bool
	callDepth   int
	tracing     bool
}

// LoggerOption defines a functional option for configuring Logger.
type LoggerOption func(*Logger)

// WithHandler adds an OutputHandler to the logger.
func WithHandler(handler OutputHandler) LoggerOption {
	return func(l *Logger) {
		l.handlers = append(l.handlers, handler)
	}
}

// WithLevel sets the minimum log level.
func WithLevel(level Level) LoggerOption {
	return func(l *Logger) {
		l.level = level
	}
}

// WithService sets the service name.
func WithService(name string) LoggerOption {
	return func(l *Logger) {
		l.serviceName = name
	}
}

// WithSampling enables sampling of log entries.
func WithSampling(rate int) LoggerOption {
	return func(l *Logger) {
		l.sampling = true
		l.sampleRate = rate
	}
}

// WithTracing enables trace ID and span ID in logs.
func WithTracing() LoggerOption {
	return func(l *Logger) {
		l.tracing = true
	}
}

// WithCallDepth sets the call depth for caller information.
func WithCallDepth(depth int) LoggerOption {
	return func(l *Logger) {
		l.callDepth = depth
	}
}

// NewLogger creates a new logger with the given options.
func NewLogger(options ...LoggerOption) *Logger {
	logger := &Logger{
		handlers:    make([]OutputHandler, 0),
		level:       InfoLevel,
		serviceName: "unknown",
		sampleRate:  100, // Log every 100th message when sampling
		callDepth:   2,   // Default call depth
	}
	
	for _, option := range options {
		option(logger)
	}
	
	return logger
}

// shouldLog determines if an entry should be logged based on level and sampling.
func (l *Logger) shouldLog(level Level) bool {
	if level < l.level {
		return false
	}
	
	if !l.sampling {
		return true
	}
	
	// Simple sampling implementation
	return time.Now().UnixNano()%int64(l.sampleRate) == 0
}

// getCaller returns the file name and line number of the caller.
func (l *Logger) getCaller() string {
	_, file, line, ok := runtime.Caller(l.callDepth)
	if !ok {
		return "unknown:0"
	}
	
	// Use short file path
	short := file
	for i := len(file) - 1; i > 0; i-- {
		if file[i] == '/' {
			short = file[i+1:]
			break
		}
	}
	
	return fmt.Sprintf("%s:%d", short, line)
}

// getTraceInfo extracts trace and span IDs from context if available.
func (l *Logger) getTraceInfo(ctx context.Context) (string, string) {
	if !l.tracing || ctx == nil {
		return "", ""
	}
	
	// This would normally extract trace/span IDs from OpenTelemetry, AWS X-Ray, etc.
	// For simplicity, we're using placeholder logic
	traceID := ctx.Value("trace_id")
	spanID := ctx.Value("span_id")
	
	traceIDStr := ""
	spanIDStr := ""
	
	if traceID != nil {
		traceIDStr = fmt.Sprintf("%v", traceID)
	}
	
	if spanID != nil {
		spanIDStr = fmt.Sprintf("%v", spanID)
	}
	
	return traceIDStr, spanIDStr
}

// With creates a new Entry with the given fields.
func (l *Logger) With(fields ...Field) *EntryBuilder {
	fieldsMap := make(map[string]interface{}, len(fields))
	for _, field := range fields {
		fieldsMap[field.Key] = field.Value
	}
	
	return &EntryBuilder{
		logger: l,
		fields: fieldsMap,
	}
}

// log logs an entry with the given level and message.
func (l *Logger) log(ctx context.Context, level Level, message string, fields ...Field) {
	if !l.shouldLog(level) {
		return
	}
	
	fieldsMap := make(map[string]interface{}, len(fields))
	for _, field := range fields {
		fieldsMap[field.Key] = field.Value
	}
	
	traceID, spanID := l.getTraceInfo(ctx)
	
	entry := Entry{
		ID:        uuid.New().String(),
		Timestamp: time.Now(),
		Level:     level.String(),
		Message:   message,
		Fields:    fieldsMap,
		Service:   l.serviceName,
		Caller:    l.getCaller(),
		TraceID:   traceID,
		SpanID:    spanID,
	}
	
	l.mu.RLock()
	handlers := l.handlers
	l.mu.RUnlock()
	
	// Send to all handlers
	for _, handler := range handlers {
		if err := handler.Handle(entry); err != nil {
			// In a real implementation, we would have a more sophisticated error handling strategy
			fmt.Fprintf(os.Stderr, "Failed to log entry: %v\n", err)
		}
	}
	
	// If it's a fatal log, exit the application
	if level == FatalLevel {
		os.Exit(1)
	}
}

// Debug logs a debug message.
func (l *Logger) Debug(ctx context.Context, message string, fields ...Field) {
	l.log(ctx, DebugLevel, message, fields...)
}

// Info logs an info message.
func (l *Logger) Info(ctx context.Context, message string, fields ...Field) {
	l.log(ctx, InfoLevel, message, fields...)
}

// Warn logs a warning message.
func (l *Logger) Warn(ctx context.Context, message string, fields ...Field) {
	l.log(ctx, WarnLevel, message, fields...)
}

// Error logs an error message.
func (l *Logger) Error(ctx context.Context, message string, fields ...Field) {
	l.log(ctx, ErrorLevel, message, fields...)
}

// Fatal logs a fatal message and exits the application.
func (l *Logger) Fatal(ctx context.Context, message string, fields ...Field) {
	l.log(ctx, FatalLevel, message, fields...)
}

// Close closes all handlers.
func (l *Logger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	
	var errs []string
	
	for _, handler := range l.handlers {
		if err := handler.Close(); err != nil {
			errs = append(errs, err.Error())
		}
	}
	
	if len(errs) > 0 {
		return fmt.Errorf("errors closing handlers: %s", strings.Join(errs, "; "))
	}
	
	return nil
}

// AddHandler adds a handler to the logger.
func (l *Logger) AddHandler(handler OutputHandler) {
	l.mu.Lock()
	defer l.mu.Unlock()
	
	l.handlers = append(l.handlers, handler)
}

// SetLevel sets the minimum log level.
func (l *Logger) SetLevel(level Level) {
	l.mu.Lock()
	defer l.mu.Unlock()
	
	l.level = level
}

// EntryBuilder helps build entries with additional context.
type EntryBuilder struct {
	logger *Logger
	fields map[string]interface{}
	ctx    context.Context
}

// Context adds a context to the entry builder.
func (b *EntryBuilder) Context(ctx context.Context) *EntryBuilder {
	b.ctx = ctx
	return b
}

// Debug logs a debug message with the accumulated fields.
func (b *EntryBuilder) Debug(message string) {
	fieldsSlice := b.fieldsToSlice()
	b.logger.Debug(b.ctx, message, fieldsSlice...)
}

// Info logs an info message with the accumulated fields.
func (b *EntryBuilder) Info(message string) {
	fieldsSlice := b.fieldsToSlice()
	b.logger.Info(b.ctx, message, fieldsSlice...)
}

// Warn logs a warning message with the accumulated fields.
func (b *EntryBuilder) Warn(message string) {
	fieldsSlice := b.fieldsToSlice()
	b.logger.Warn(b.ctx, message, fieldsSlice...)
}

// Error logs an error message with the accumulated fields.
func (b *EntryBuilder) Error(message string) {
	fieldsSlice := b.fieldsToSlice()
	b.logger.Error(b.ctx, message, fieldsSlice...)
}

// Fatal logs a fatal message with the accumulated fields and exits.
func (b *EntryBuilder) Fatal(message string) {
	fieldsSlice := b.fieldsToSlice()
	b.logger.Fatal(b.ctx, message, fieldsSlice...)
}

// fieldsToSlice converts the fields map to a slice of Field.
func (b *EntryBuilder) fieldsToSlice() []Field {
	fields := make([]Field, 0, len(b.fields))
	for k, v := range b.fields {
		fields = append(fields, Field{Key: k, Value: v})
	}
	return fields
}

// WithField adds a field to the entry builder.
func (b *EntryBuilder) WithField(key string, value interface{}) *EntryBuilder {
	b.fields[key] = value
	return b
}

// WithFields adds multiple fields to the entry builder.
func (b *EntryBuilder) WithFields(fields ...Field) *EntryBuilder {
	for _, field := range fields {
		b.fields[field.Key] = field.Value
	}
	return b
}

// WithError adds an error as a field.
func (b *EntryBuilder) WithError(err error) *EntryBuilder {
	if err != nil {
		b.fields["error"] = err.Error()
	}
	return b
}

// isSameDay checks if two times are on the same day.
func isSameDay(t1, t2 time.Time) bool {
	y1, m1, d1 := t1.Date()
	y2, m2, d2 := t2.Date()
	return y1 == y2 && m1 == m2 && d1 == d2
}

// F is a shorthand for creating a Field.
func F(key string, value interface{}) Field {
	return Field{Key: key, Value: value}
}

// DefaultLogger creates a basic logger that writes to the console.
func DefaultLogger(serviceName string) *Logger {
	formatter := &TextFormatter{
		IncludeTimestamp: true,
		IncludeCaller:    true,
	}
	
	handler := NewConsoleHandler(formatter)
	
	return NewLogger(
		WithService(serviceName),
		WithHandler(handler),
		WithLevel(InfoLevel),
	)
}

// ContextWithTraceID adds a trace ID to a context.
func ContextWithTraceID(ctx context.Context, traceID string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, "trace_id", traceID)
}

// ContextWithSpanID adds a span ID to a context.
func ContextWithSpanID(ctx context.Context, spanID string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, "span_id", spanID)
}

// ContextWithNewTrace creates a context with new trace and span IDs.
func ContextWithNewTrace(ctx context.Context) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	
	traceID := uuid.New().String()
	spanID := uuid.New().String()
	
	ctx = ContextWithTraceID(ctx, traceID)
	ctx = ContextWithSpanID(ctx, spanID)
	
	return ctx
}

// GetTraceID extracts a trace ID from a context.
func GetTraceID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	
	traceID := ctx.Value("trace_id")
	if traceID == nil {
		return ""
	}
	
	return fmt.Sprintf("%v", traceID)
}

// GetSpanID extracts a span ID from a context.
func GetSpanID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	
	spanID := ctx.Value("span_id")
	if spanID == nil {
		return ""
	}
	
	return fmt.Sprintf("%v", spanID)
}

// HTTPMiddleware creates middleware for HTTP handlers that adds trace context.
func HTTPMiddleware(log *Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract trace ID from headers or generate a new one
			traceID := r.Header.Get("X-Trace-ID")
			if traceID == "" {
				traceID = uuid.New().String()
			}
			
			// Generate a new span ID for this request
			spanID := uuid.New().String()
			
			// Create a new context with trace information
			ctx := r.Context()
			ctx = ContextWithTraceID(ctx, traceID)
			ctx = ContextWithSpanID(ctx, spanID)
			
			// Add the trace ID to the response headers
			w.Header().Set("X-Trace-ID", traceID)
			
			// Log the request
			log.With(
				F("method", r.Method),
				F("path", r.URL.Path),
				F("remote_addr", r.RemoteAddr),
				F("user_agent", r.UserAgent()),
			).Context(ctx).Info("HTTP request received")
			
			// Call the next handler with the enhanced context
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}