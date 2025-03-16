package webauthn

import (
	"errors"
	"sync"
)

// UserStore is a simple in-memory store for users
type UserStore struct {
	users map[string]*User
	mu    sync.RWMutex
}

// NewUserStore creates a new UserStore
func NewUserStore() *UserStore {
	return &UserStore{
		users: make(map[string]*User),
	}
}

// GetUser returns a user by username
func (s *UserStore) GetUser(username string) (*User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	user, ok := s.users[username]
	if !ok {
		return nil, errors.New("user not found")
	}

	return user, nil
}

// PutUser adds or updates a user
func (s *UserStore) PutUser(user *User) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.users[user.Name] = user
}

// DeleteUser removes a user
func (s *UserStore) DeleteUser(username string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.users, username)
}
