package userservice

import (
	"context"
	"errors"
	"time"
)

// User represents a user in the system
type User struct {
	ID        string
	Name      string
	Email     string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Common errors
var (
	ErrUserNotFound = errors.New("user not found")
	ErrInvalidUser  = errors.New("invalid user data")
	ErrDatabaseError = errors.New("database error")
)

// Database defines the interface for data storage operations
type Database interface {
	QueryUser(ctx context.Context, id string) (*User, error)
	InsertUser(ctx context.Context, user *User) error
	UpdateUser(ctx context.Context, user *User) error
	DeleteUser(ctx context.Context, id string) error
	Close() error
}

// Logger defines the interface for logging operations
type Logger interface {
	Info(message string, args ...interface{})
	Error(message string, args ...interface{})
	Debug(message string, args ...interface{})
}

// UserService handles user-related operations
type UserService struct {
	db     Database
	logger Logger
}

// NewUserService creates a new UserService instance
func NewUserService(db Database, logger Logger) *UserService {
	return &UserService{
		db:     db,
		logger: logger,
	}
}

// GetUser retrieves a user by ID
func (s *UserService) GetUser(ctx context.Context, id string) (*User, error) {
	s.logger.Info("Getting user")
	
	if id == "" {
		s.logger.Error("Invalid user ID provided")
		return nil, ErrInvalidUser
	}
	
	user, err := s.db.QueryUser(ctx, id)
	if err != nil {
		s.logger.Error("Failed to get user", "error", err)
		return nil, err
	}
	
	s.logger.Info("User retrieved successfully")
	return user, nil
}

// CreateUser creates a new user
func (s *UserService) CreateUser(ctx context.Context, user *User) error {
	s.logger.Info("Creating user")
	
	if user == nil || user.Name == "" || user.Email == "" {
		s.logger.Error("Invalid user data provided")
		return ErrInvalidUser
	}
	
	// Set timestamps
	now := time.Now()
	user.CreatedAt = now
	user.UpdatedAt = now
	
	err := s.db.InsertUser(ctx, user)
	if err != nil {
		s.logger.Error("Failed to create user", "error", err)
		return err
	}
	
	s.logger.Info("User created successfully")
	return nil
}

// UpdateUser updates an existing user
func (s *UserService) UpdateUser(ctx context.Context, user *User) error {
	s.logger.Info("Updating user")
	
	if user == nil || user.ID == "" {
		s.logger.Error("Invalid user data provided")
		return ErrInvalidUser
	}
	
	// Update timestamp
	user.UpdatedAt = time.Now()
	
	err := s.db.UpdateUser(ctx, user)
	if err != nil {
		s.logger.Error("Failed to update user", "error", err)
		return err
	}
	
	s.logger.Info("User updated successfully")
	return nil
}

// DeleteUser deletes a user by ID
func (s *UserService) DeleteUser(ctx context.Context, id string) error {
	s.logger.Info("Deleting user")
	
	if id == "" {
		s.logger.Error("Invalid user ID provided")
		return ErrInvalidUser
	}
	
	err := s.db.DeleteUser(ctx, id)
	if err != nil {
		s.logger.Error("Failed to delete user", "error", err)
		return err
	}
	
	s.logger.Info("User deleted successfully")
	return nil
}