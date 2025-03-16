package webauthn

import (
	"errors"
	"net/http"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
)

// Service represents the WebAuthn service
type Service struct {
	webAuthn  *webauthn.WebAuthn
	userStore *UserStore
}

// NewService creates a new WebAuthn service
func NewService(rpID, rpOrigin, rpDisplayName string) (*Service, error) {
	// Initialize WebAuthn
	webAuthn, err := webauthn.New(&webauthn.Config{
		RPDisplayName: rpDisplayName,      // Display name for your site
		RPID:          rpID,               // Generally the domain name for your site
		RPOrigins:     []string{rpOrigin}, // The origin URLs for WebAuthn requests
	})

	if err != nil {
		return nil, err
	}

	return &Service{
		webAuthn:  webAuthn,
		userStore: NewUserStore(),
	}, nil
}

// BeginRegistration starts the registration process
func (s *Service) BeginRegistration(username, displayName string) (*protocol.CredentialCreation, *User, error) {
	// Get user or create a new one
	user, err := s.userStore.GetUser(username)
	if err != nil {
		// User doesn't exist, create a new one
		user = NewUser(username, displayName)
		s.userStore.PutUser(user)
	}

	// Begin registration
	options, sessionData, err := s.webAuthn.BeginRegistration(user)
	if err != nil {
		return nil, nil, err
	}

	// Store session data in the user
	user.RegistrationSessionData = sessionData

	return options, user, nil
}

// FinishRegistration completes the registration process
func (s *Service) FinishRegistration(username string, response *http.Request) error {
	// Get user
	user, err := s.userStore.GetUser(username)
	if err != nil {
		return err
	}

	// Get session data
	sessionData := user.RegistrationSessionData
	if sessionData == nil {
		return errors.New("no registration session data found")
	}

	// Parse response
	credential, err := s.webAuthn.FinishRegistration(user, *sessionData, response)
	if err != nil {
		return err
	}

	// Add credential to user
	user.AddCredential(*credential)

	// Clear session data
	user.RegistrationSessionData = nil

	// Update user in store
	s.userStore.PutUser(user)

	return nil
}

// BeginLogin starts the login process
func (s *Service) BeginLogin(username string) (*protocol.CredentialAssertion, error) {
	// Get user
	user, err := s.userStore.GetUser(username)
	if err != nil {
		return nil, err
	}

	// Begin login
	options, sessionData, err := s.webAuthn.BeginLogin(user)
	if err != nil {
		return nil, err
	}

	// Store session data in the user
	user.AuthenticationSessionData = sessionData

	// Update user in store
	s.userStore.PutUser(user)

	return options, nil
}

// FinishLogin completes the login process
func (s *Service) FinishLogin(username string, response *http.Request) error {
	// Get user
	user, err := s.userStore.GetUser(username)
	if err != nil {
		return err
	}

	// Get session data
	sessionData := user.AuthenticationSessionData
	if sessionData == nil {
		return errors.New("no authentication session data found")
	}

	// Parse response
	_, err = s.webAuthn.FinishLogin(user, *sessionData, response)
	if err != nil {
		return err
	}

	// Clear session data
	user.AuthenticationSessionData = nil

	// Update user in store
	s.userStore.PutUser(user)

	return nil
}
