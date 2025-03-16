package sso

import (
	"fmt"
	"log"
	"net/http"
	"time"
)

// SSOHandler handles SSO authentication for multiple providers
type SSOHandler struct {
	Providers      map[string]Provider
	SessionManager SessionManager
	StateManager   *StateManager
	// Default URL to redirect after successful login
	DefaultRedirectURL string
}

// NewSSOHandler creates a new SSOHandler
func NewSSOHandler(sessionManager SessionManager, defaultRedirectURL string) *SSOHandler {
	return &SSOHandler{
		Providers:          make(map[string]Provider),
		SessionManager:     sessionManager,
		StateManager:       NewStateManager(),
		DefaultRedirectURL: defaultRedirectURL,
	}
}

// RegisterProvider registers a new SSO provider
func (h *SSOHandler) RegisterProvider(provider Provider) {
	h.Providers[provider.Name()] = provider
}

// LoginHandler initiates the SSO flow for a specific provider
func (h *SSOHandler) LoginHandler(w http.ResponseWriter, r *http.Request) {
	// Get the provider name from the URL path
	providerName := r.URL.Query().Get("provider")
	if providerName == "" {
		http.Error(w, "Provider not specified", http.StatusBadRequest)
		return
	}

	// Get the provider
	provider, exists := h.Providers[providerName]
	if !exists {
		http.Error(w, fmt.Sprintf("Provider '%s' not supported", providerName), http.StatusBadRequest)
		return
	}

	// Generate a state token for CSRF protection
	state, err := GenerateRandomString(32)
	if err != nil {
		http.Error(w, "Failed to generate state token", http.StatusInternalServerError)
		return
	}

	// Store the state token with an expiration time (e.g., 10 minutes)
	h.StateManager.SaveState(state, 10*time.Minute)

	// Get the redirect URL from the query parameters or use the default
	redirectURL := r.URL.Query().Get("redirect_url")
	if redirectURL == "" {
		redirectURL = h.DefaultRedirectURL
	}

	// Validate the redirect URL
	if !IsValidRedirectURL(redirectURL) {
		http.Error(w, "Invalid redirect URL", http.StatusBadRequest)
		return
	}

	// Encode the state token with the redirect URL
	encodedState := EncodeState(state, redirectURL)

	// Redirect to the provider's authorization URL
	url := provider.GetAuthURL(encodedState)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

// CallbackHandler handles the callback from the SSO provider
func (h *SSOHandler) CallbackHandler(w http.ResponseWriter, r *http.Request) {
	// Get the provider name from the URL path
	providerName := r.URL.Query().Get("provider")
	if providerName == "" {
		http.Error(w, "Provider not specified", http.StatusBadRequest)
		return
	}

	// Get the provider
	provider, exists := h.Providers[providerName]
	if !exists {
		http.Error(w, fmt.Sprintf("Provider '%s' not supported", providerName), http.StatusBadRequest)
		return
	}

	// Get the state and code from the callback
	encodedState := r.URL.Query().Get("state")
	if encodedState == "" {
		http.Error(w, "Missing state parameter", http.StatusBadRequest)
		return
	}

	// Decode the state token and redirect URL
	state, redirectURL := DecodeState(encodedState)

	// If no redirect URL was found, use the default
	if redirectURL == "" {
		redirectURL = h.DefaultRedirectURL
	}

	// Validate the redirect URL
	if !IsValidRedirectURL(redirectURL) {
		http.Error(w, "Invalid redirect URL", http.StatusBadRequest)
		return
	}

	// Validate state token to prevent CSRF
	if !h.StateManager.ValidateState(state) {
		http.Error(w, "Invalid or expired state token", http.StatusBadRequest)
		return
	}

	// Handle the callback from the provider
	profile, err := provider.HandleCallback(r.Context(), r)
	if err != nil {
		http.Error(w, fmt.Sprintf("Authentication failed: %v", err), http.StatusInternalServerError)
		return
	}

	// Save the user session
	err = h.SessionManager.SaveSession(w, profile)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to save session: %v", err), http.StatusInternalServerError)
		return
	}

	// Log the successful authentication
	log.Printf("User authenticated: ID=%s, Email=%s, Name=%s, Provider=%s",
		profile.ID, profile.Email, profile.Name, profile.Provider)

	// Redirect to the redirect URL
	http.Redirect(w, r, redirectURL, http.StatusTemporaryRedirect)
}

// LogoutHandler handles user logout
func (h *SSOHandler) LogoutHandler(w http.ResponseWriter, r *http.Request) {
	// Clear the session
	err := h.SessionManager.ClearSession(w)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to clear session: %v", err), http.StatusInternalServerError)
		return
	}

	// Get the redirect URL from the query parameters or use the default
	redirectURL := r.URL.Query().Get("redirect_url")
	if redirectURL == "" {
		redirectURL = h.DefaultRedirectURL
	}

	// Validate the redirect URL
	if !IsValidRedirectURL(redirectURL) {
		http.Error(w, "Invalid redirect URL", http.StatusBadRequest)
		return
	}

	// Redirect to the redirect URL
	http.Redirect(w, r, redirectURL, http.StatusTemporaryRedirect)
}

// RegisterHandlers registers the SSO handlers with the provided ServeMux
func (h *SSOHandler) RegisterHandlers(mux *http.ServeMux) {
	mux.HandleFunc("/auth/login", h.LoginHandler)
	mux.HandleFunc("/auth/callback", h.CallbackHandler)
	mux.HandleFunc("/auth/logout", h.LogoutHandler)
}
