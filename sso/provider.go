package sso

import (
	"context"
	"net/http"
)

// UserProfile represents a standardized user profile across different SSO providers
type UserProfile struct {
	ID            string
	Email         string
	EmailVerified bool
	Name          string
	FirstName     string
	LastName      string
	Picture       string
	Locale        string
	Provider      string // The name of the provider (e.g., "google", "github")
	RawData       map[string]interface{}
}

// Provider defines the interface for SSO providers
type Provider interface {
	// Name returns the name of the provider
	Name() string

	// GetAuthURL returns the URL to redirect the user to for authentication
	GetAuthURL(state string) string

	// HandleCallback processes the callback from the provider
	HandleCallback(ctx context.Context, r *http.Request) (*UserProfile, error)
}

// ProviderConfig contains common configuration for SSO providers
type ProviderConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
	Scopes       []string
}
