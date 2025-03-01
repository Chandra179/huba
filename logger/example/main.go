package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"securedesign/logger"
	"time"
)

// Global logger instance
var log *logger.Logger

func init() {
	// Initialize logger with multiple outputs
	
	// Console handler with text formatter for human-readable logs
	consoleFormatter := &logger.TextFormatter{
		IncludeTimestamp: true,
		TimestampFormat:  time.RFC3339,
		IncludeCaller:    true,
	}
	consoleHandler := logger.NewConsoleHandler(consoleFormatter)
	
	// File handler with JSON formatter for machine-readable logs
	jsonFormatter := &logger.JsonFormatter{
		Pretty: false,
	}
	fileHandler := logger.NewFileHandler(
		"logs",                // Directory
		"application.log",     // Filename
		10*1024*1024,          // 10MB max file size
		5,                     // Keep 5 rotated files
		jsonFormatter,
	)
	
	// HTTP handler for shipping logs to a central service
	// Note: Only enabled in production
	var httpHandler *logger.HttpHandler
	if os.Getenv("ENVIRONMENT") == "production" {
		httpHandler = logger.NewHttpHandler(
			"https://logging.example.com/ingest", // Endpoint
			100,                                  // Batch size
			3,                                    // Max retry attempts
			jsonFormatter,
		)
		
		// // Add custom headers for authentication
		// httpHandler.headers = map[string]string{
		// 	"Authorization": "Bearer " + os.Getenv("LOG_API_KEY"),
		// }
	}
	
	// Create the logger with options
	loggerOptions := []logger.LoggerOption{
		logger.WithService("user-service"),
		logger.WithLevel(logger.InfoLevel),
		logger.WithHandler(consoleHandler),
		logger.WithHandler(fileHandler),
		logger.WithTracing(), // Enable trace ID and span ID in logs
	}
	
	// Add HTTP handler in production
	if httpHandler != nil {
		loggerOptions = append(loggerOptions, logger.WithHandler(httpHandler))
	}
	
	// Initialize the global logger
	log = logger.NewLogger(loggerOptions...)
	
	// Enable debug logs in development
	if os.Getenv("ENVIRONMENT") == "development" {
		log.SetLevel(logger.DebugLevel)
	}
}

// UserService represents a simple user service
type UserService struct {
	db Database
}

// Database is a mock database interface
type Database interface {
	GetUser(ctx context.Context, id string) (User, error)
	UpdateUser(ctx context.Context, user User) error
}

// MockDB is a simple mock implementation of Database
type MockDB struct{}

// User represents a user in the system
type User struct {
	ID        string
	Username  string
	Email     string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// GetUser retrieves a user from the database
func (m *MockDB) GetUser(ctx context.Context, id string) (User, error) {
	// Simulate database query
	if id == "invalid" {
		return User{}, fmt.Errorf("user not found")
	}
	
	return User{
		ID:        id,
		Username:  "testuser",
		Email:     "test@example.com",
		CreatedAt: time.Now().Add(-24 * time.Hour),
		UpdatedAt: time.Now(),
	}, nil
}

// UpdateUser updates a user in the database
func (m *MockDB) UpdateUser(ctx context.Context, user User) error {
	// Simulate database update
	if user.ID == "invalid" {
		return fmt.Errorf("user not found")
	}
	
	return nil
}

// NewUserService creates a new user service
func NewUserService(db Database) *UserService {
	return &UserService{db: db}
}

// GetUserHandler handles GET /users/:id requests
func (s *UserService) GetUserHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	
	// Extract user ID from URL
	// In a real application, you'd use a router like gorilla/mux or chi
	userID := r.URL.Path[len("/users/"):]
	
	log.With(
		logger.F("user_id", userID),
		logger.F("request_id", r.Header.Get("X-Request-ID")),
	).Context(ctx).Info("Retrieving user")
	
	// Get user from database
	user, err := s.db.GetUser(ctx, userID)
	if err != nil {
		log.With(logger.F("error", err)).Context(ctx).Error("Failed to retrieve user")
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}
	
	// Log successful retrieval with structured data
	log.With(
		logger.F("user_id", user.ID),
		logger.F("username", user.Username),
		logger.F("created_at", user.CreatedAt),
	).Context(ctx).Debug("User retrieved successfully")
	
	// In a real application, you'd serialize the user to JSON
	fmt.Fprintf(w, "User: %s (%s)\n", user.Username, user.Email)
}

// UpdateUserHandler handles PUT /users/:id requests
func (s *UserService) UpdateUserHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	
	// Extract user ID from URL
	userID := r.URL.Path[len("/users/"):]
	
	// Get user from database
	user, err := s.db.GetUser(ctx, userID)
	if err != nil {
		log.With(logger.F("error", err)).Context(ctx).Error("Failed to retrieve user for update")
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}
	
	// In a real application, you'd parse the request body
	user.Email = "updated@example.com"
	user.UpdatedAt = time.Now()
	
	// Start a timer for the update operation
	startTime := time.Now()
	
	// Update user in database
	err = s.db.UpdateUser(ctx, user)
	if err != nil {
		log.With(
			logger.F("error", err),
			logger.F("user_id", user.ID),
		).Context(ctx).Error("Failed to update user")
		http.Error(w, "Failed to update user", http.StatusInternalServerError)
		return
	}
	
	// Calculate operation duration
	duration := time.Since(startTime)
	
	// Log successful update with structured data including timing
	log.With(
		logger.F("user_id", user.ID),
		logger.F("username", user.Username),
		logger.F("duration_ms", duration.Milliseconds()),
	).Context(ctx).Info("User updated successfully")
	
	fmt.Fprintf(w, "User updated successfully\n")
}

func main() {
	// Defer closing the logger to ensure all logs are flushed
	defer func() {
		if err := log.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "Error closing logger: %v\n", err)
		}
	}()
	
	log.Info(context.Background(), "Starting user service", logger.F("version", "1.0.0"))
	
	// Create user service with mock database
	userService := NewUserService(&MockDB{})
	
	// Set up HTTP server with logging middleware
	mux := http.NewServeMux()
	mux.HandleFunc("/users/", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			userService.GetUserHandler(w, r)
		case http.MethodPut:
			userService.UpdateUserHandler(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})
	
	// Apply logging middleware
	handler := logger.HTTPMiddleware(log)(mux)
	
	// Start server
	addr := ":8080"
	log.Info(context.Background(), "Server listening", logger.F("address", addr))
	
	server := &http.Server{
		Addr:    addr,
		Handler: handler,
	}
	
	if err := server.ListenAndServe(); err != nil {
		log.Fatal(context.Background(), "Server failed", logger.F("error", err))
	}
}