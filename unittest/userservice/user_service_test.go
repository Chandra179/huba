package userservice

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockDatabase implements the Database interface for testing
type MockDatabase struct {
	mock.Mock
}

func (m *MockDatabase) QueryUser(ctx context.Context, id string) (*User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*User), args.Error(1)
}

func (m *MockDatabase) InsertUser(ctx context.Context, user *User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockDatabase) UpdateUser(ctx context.Context, user *User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockDatabase) DeleteUser(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockDatabase) Close() error {
	args := m.Called()
	return args.Error(0)
}

// MockLogger implements the Logger interface for testing
type MockLogger struct {
	mock.Mock
}

func (m *MockLogger) Info(message string, args ...interface{}) {
	callArgs := []interface{}{message}
	callArgs = append(callArgs, args...)
	m.Called(callArgs...)
}

func (m *MockLogger) Error(message string, args ...interface{}) {
	callArgs := []interface{}{message}
	callArgs = append(callArgs, args...)
	m.Called(callArgs...)
}

func (m *MockLogger) Debug(message string, args ...interface{}) {
	callArgs := []interface{}{message}
	callArgs = append(callArgs, args...)
	m.Called(callArgs...)
}

func TestUserService_GetUser(t *testing.T) {
	// Test table
	tests := []struct {
		name         string
		userID       string
		mockSetup    func(*MockDatabase, *MockLogger)
		expectedUser *User
		expectedErr  error
	}{
		{
			name:   "Success - User found",
			userID: "123",
			mockSetup: func(db *MockDatabase, logger *MockLogger) {
				// Setup expected call order using InOrder
				mockUser := &User{
					ID:        "123",
					Name:      "John Doe",
					Email:     "john@example.com",
					CreatedAt: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
					UpdatedAt: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
				}

				infoCall := logger.On("Info", "Getting user").Return()
				queryCall := db.On("QueryUser", mock.Anything, "123").Return(mockUser, nil)
				successCall := logger.On("Info", "User retrieved successfully").Return()

				mock.InOrder(
					infoCall,
					queryCall,
					successCall,
				)
			},
			expectedUser: &User{
				ID:        "123",
				Name:      "John Doe",
				Email:     "john@example.com",
				CreatedAt: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
				UpdatedAt: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
			},
			expectedErr: nil,
		},
		{
			name:   "Error - User not found",
			userID: "999",
			mockSetup: func(db *MockDatabase, logger *MockLogger) {
				infoCall := logger.On("Info", "Getting user").Return()
				queryCall := db.On("QueryUser", mock.Anything, "999").Return(nil, ErrUserNotFound)
				errorCall := logger.On("Error", "Failed to get user", "error", ErrUserNotFound).Return()

				mock.InOrder(
					infoCall,
					queryCall,
					errorCall,
				)
			},
			expectedUser: nil,
			expectedErr:  ErrUserNotFound,
		},
		{
			name:   "Error - Database error",
			userID: "456",
			mockSetup: func(db *MockDatabase, logger *MockLogger) {
				infoCall := logger.On("Info", "Getting user").Return()
				queryCall := db.On("QueryUser", mock.Anything, "456").Return(nil, ErrDatabaseError)
				errorCall := logger.On("Error", "Failed to get user", "error", ErrDatabaseError).Return()

				mock.InOrder(
					infoCall,
					queryCall,
					errorCall,
				)
			},
			expectedUser: nil,
			expectedErr:  ErrDatabaseError,
		},
		{
			name:   "Error - Empty user ID",
			userID: "",
			mockSetup: func(db *MockDatabase, logger *MockLogger) {
				infoCall := logger.On("Info", "Getting user").Return()
				errorCall := logger.On("Error", "Invalid user ID provided").Return()

				mock.InOrder(
					infoCall,
					errorCall,
				)
			},
			expectedUser: nil,
			expectedErr:  ErrInvalidUser,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Enable parallel test execution
			t.Parallel()

			// Create mocks
			mockDB := new(MockDatabase)
			mockLogger := new(MockLogger)

			// Setup mocks for this specific test case
			tc.mockSetup(mockDB, mockLogger)

			// Create service with mocks
			userService := NewUserService(mockDB, mockLogger)

			// Call the method being tested
			ctx := context.Background()
			user, err := userService.GetUser(ctx, tc.userID)

			// Assert expectations
			if tc.expectedErr != nil {
				assert.Error(t, err)
				assert.Equal(t, tc.expectedErr, err)
				assert.Nil(t, user)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, user)
				assert.Equal(t, tc.expectedUser.ID, user.ID)
				assert.Equal(t, tc.expectedUser.Name, user.Name)
				assert.Equal(t, tc.expectedUser.Email, user.Email)
				assert.Equal(t, tc.expectedUser.CreatedAt, user.CreatedAt)
				assert.Equal(t, tc.expectedUser.UpdatedAt, user.UpdatedAt)
			}

			// Verify all expectations were met (call counts, arguments)
			mockDB.AssertExpectations(t)
			mockLogger.AssertExpectations(t)

			// Verify specific call count if needed
			if tc.userID != "" {
				mockDB.AssertNumberOfCalls(t, "QueryUser", 1)
			} else {
				mockDB.AssertNumberOfCalls(t, "QueryUser", 0)
			}

			// Verify argument values for specific calls
			if tc.userID == "123" {
				mockDB.AssertCalled(t, "QueryUser", mock.Anything, "123")
			}
		})
	}
}

func TestUserService_CreateUser(t *testing.T) {
	// Setup
	mockDB := new(MockDatabase)
	mockLogger := new(MockLogger)

	// Register cleanup to be executed after test finishes
	t.Cleanup(func() {
		// Simulate database connection closing
		mockDB.On("Close").Return(nil).Once()
		mockDB.Close()
		mockDB.AssertExpectations(t)
	})

	// Setup expected calls in expected order
	infoCreateCall := mockLogger.On("Info", "Creating user").Return()

	// Use mock.MatchedBy to verify argument values
	insertCall := mockDB.On("InsertUser", mock.Anything, mock.MatchedBy(func(u *User) bool {
		return u.Name == "Jane Smith" &&
			u.Email == "jane@example.com" &&
			!u.CreatedAt.IsZero() &&
			!u.UpdatedAt.IsZero()
	})).Return(nil)

	infoSuccessCall := mockLogger.On("Info", "User created successfully").Return()

	// Define the order of calls
	mock.InOrder(
		infoCreateCall,
		insertCall,
		infoSuccessCall,
	)

	// Create service with mocks
	userService := NewUserService(mockDB, mockLogger)

	// Execute test
	newUser := &User{
		Name:  "Jane Smith",
		Email: "jane@example.com",
	}

	err := userService.CreateUser(context.Background(), newUser)

	// Assert results
	require.NoError(t, err)

	// Verify calls with specific argument matchers
	mockDB.AssertCalled(t, "InsertUser", mock.Anything, mock.MatchedBy(func(u *User) bool {
		return u.Name == "Jane Smith" && u.Email == "jane@example.com"
	}))

	// Verify call counts
	mockDB.AssertNumberOfCalls(t, "InsertUser", 1)
	mockLogger.AssertNumberOfCalls(t, "Info", 2) // Creating user + User created successfully
}

func TestUserService_DeleteUser_Concurrent(t *testing.T) {
	// Skip long-running tests in short mode
	if testing.Short() {
		t.Skip("Skipping concurrent test in short mode")
	}

	// Create a test function that will be run concurrently
	testDelete := func(userID string, t *testing.T) {
		t.Parallel() // Mark as parallel

		mockDB := new(MockDatabase)
		mockLogger := new(MockLogger)

		// Set up ordered call expectations
		infoDeleteCall := mockLogger.On("Info", "Deleting user").Return()
		deleteCall := mockDB.On("DeleteUser", mock.Anything, userID).Return(nil)
		infoSuccessCall := mockLogger.On("Info", "User deleted successfully").Return()

		mock.InOrder(
			infoDeleteCall,
			deleteCall,
			infoSuccessCall,
		)

		userService := NewUserService(mockDB, mockLogger)

		err := userService.DeleteUser(context.Background(), userID)
		assert.NoError(t, err)

		mockDB.AssertExpectations(t)
		mockLogger.AssertExpectations(t)
		mockDB.AssertNumberOfCalls(t, "DeleteUser", 1)
	}

	// Run multiple concurrent tests
	userIDs := []string{"user1", "user2", "user3", "user4", "user5"}
	for _, id := range userIDs {
		id := id // Capture loop variable
		t.Run("Delete_"+id, func(t *testing.T) {
			testDelete(id, t)
		})
	}
}

func TestUserService_UpdateUser(t *testing.T) {
	// Test table
	tests := []struct {
		name        string
		user        *User
		setupMocks  func(*MockDatabase, *MockLogger)
		expectedErr error
	}{
		{
			name: "Success - User updated",
			user: &User{
				ID:    "123",
				Name:  "Updated Name",
				Email: "updated@example.com",
			},
			setupMocks: func(db *MockDatabase, logger *MockLogger) {
				infoUpdateCall := logger.On("Info", "Updating user").Return()

				updateCall := db.On("UpdateUser", mock.Anything, mock.MatchedBy(func(u *User) bool {
					return u.ID == "123" &&
						u.Name == "Updated Name" &&
						u.Email == "updated@example.com" &&
						!u.UpdatedAt.IsZero()
				})).Return(nil)

				infoSuccessCall := logger.On("Info", "User updated successfully").Return()

				mock.InOrder(
					infoUpdateCall,
					updateCall,
					infoSuccessCall,
				)
			},
			expectedErr: nil,
		},
		{
			name: "Error - Invalid user",
			user: nil,
			setupMocks: func(db *MockDatabase, logger *MockLogger) {
				infoCall := logger.On("Info", "Updating user").Return()
				errorCall := logger.On("Error", "Invalid user data provided").Return()

				mock.InOrder(
					infoCall,
					errorCall,
				)
			},
			expectedErr: ErrInvalidUser,
		},
		{
			name: "Error - Empty user ID",
			user: &User{
				Name:  "No ID User",
				Email: "noid@example.com",
			},
			setupMocks: func(db *MockDatabase, logger *MockLogger) {
				infoCall := logger.On("Info", "Updating user").Return()
				errorCall := logger.On("Error", "Invalid user data provided").Return()

				mock.InOrder(
					infoCall,
					errorCall,
				)
			},
			expectedErr: ErrInvalidUser,
		},
		{
			name: "Error - Database error",
			user: &User{
				ID:    "456",
				Name:  "DB Error User",
				Email: "dberror@example.com",
			},
			setupMocks: func(db *MockDatabase, logger *MockLogger) {
				infoCall := logger.On("Info", "Updating user").Return()

				updateCall := db.On("UpdateUser", mock.Anything, mock.MatchedBy(func(u *User) bool {
					return u.ID == "456"
				})).Return(ErrDatabaseError)

				errorCall := logger.On("Error", "Failed to update user", "error", ErrDatabaseError).Return()

				mock.InOrder(
					infoCall,
					updateCall,
					errorCall,
				)
			},
			expectedErr: ErrDatabaseError,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Enable parallel test execution
			t.Parallel()

			// Create mocks
			mockDB := new(MockDatabase)
			mockLogger := new(MockLogger)

			// Setup mocks for this specific test case
			tc.setupMocks(mockDB, mockLogger)

			// Create service with mocks
			userService := NewUserService(mockDB, mockLogger)

			// Call the method being tested
			ctx := context.Background()
			err := userService.UpdateUser(ctx, tc.user)

			// Assert expectations
			if tc.expectedErr != nil {
				assert.Error(t, err)
				assert.Equal(t, tc.expectedErr, err)
			} else {
				assert.NoError(t, err)
			}

			// Verify all expectations were met
			mockDB.AssertExpectations(t)
			mockLogger.AssertExpectations(t)
		})
	}
}
