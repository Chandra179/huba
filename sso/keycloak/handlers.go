package keycloak

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"securedesign/oauth"
)

// KeycloakOAuthHandler handles Keycloak OAuth2 authentication
type KeycloakOAuthHandler struct {
	Config         KeycloakConfig
	SessionManager oauth.SessionManager
	StateStore     map[string]time.Time // Simple in-memory state storage
}

// NewKeycloakOAuthHandler creates a new KeycloakOAuthHandler
func NewKeycloakOAuthHandler(config KeycloakConfig, sessionManager oauth.SessionManager) *KeycloakOAuthHandler {
	return &KeycloakOAuthHandler{
		Config:         config,
		SessionManager: sessionManager,
		StateStore:     make(map[string]time.Time),
	}
}

// LoginHandler initiates the Keycloak OAuth flow
func (h *KeycloakOAuthHandler) LoginHandler(w http.ResponseWriter, r *http.Request) {
	// Generate a state token for CSRF protection
	state, err := oauth.GenerateStateToken()
	if err != nil {
		http.Error(w, "Failed to generate state token", http.StatusInternalServerError)
		return
	}

	// Store the state token with an expiration time (e.g., 10 minutes)
	h.StateStore[state] = time.Now().Add(10 * time.Minute)

	// Create the OAuth2 config
	oauthConfig := NewKeycloakOAuth(h.Config)

	// Redirect to Keycloak's OAuth 2.0 server
	url := GetKeycloakLoginURL(oauthConfig, state)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

// CallbackHandler handles the callback from Keycloak OAuth
func (h *KeycloakOAuthHandler) CallbackHandler(w http.ResponseWriter, r *http.Request) {
	// Get the state and code from the callback
	state := r.URL.Query().Get("state")
	code := r.URL.Query().Get("code")

	// Validate state token to prevent CSRF
	expirationTime, exists := h.StateStore[state]
	if !exists || time.Now().After(expirationTime) {
		http.Error(w, "Invalid or expired state token", http.StatusBadRequest)
		return
	}

	// Remove the used state token
	delete(h.StateStore, state)

	// Create the OAuth2 config
	oauthConfig := NewKeycloakOAuth(h.Config)

	// Exchange the authorization code for a token
	token, err := HandleKeycloakCallback(r.Context(), oauthConfig, state, code)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to exchange token: %v", err), http.StatusInternalServerError)
		return
	}

	// Get the user info
	userInfo, err := GetKeycloakUserInfo(r.Context(), token, h.Config)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get user info: %v", err), http.StatusInternalServerError)
		return
	}

	// Save the user session
	err = h.SessionManager.SaveSession(w, userInfo.ID, userInfo.Email, userInfo.Name)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to save session: %v", err), http.StatusInternalServerError)
		return
	}

	// Log the successful authentication
	log.Printf("User authenticated: ID=%s, Email=%s, Name=%s", userInfo.ID, userInfo.Email, userInfo.Name)

	// Redirect to the home page or dashboard
	http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
}

// LogoutHandler handles user logout
func (h *KeycloakOAuthHandler) LogoutHandler(w http.ResponseWriter, r *http.Request) {
	// Clear the session
	err := h.SessionManager.ClearSession(w)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to clear session: %v", err), http.StatusInternalServerError)
		return
	}

	// Construct the logout URL
	logoutURL := fmt.Sprintf("%s/realms/%s/protocol/openid-connect/logout", h.Config.BaseURL, h.Config.Realm)

	// Redirect to Keycloak logout endpoint
	http.Redirect(w, r, logoutURL, http.StatusTemporaryRedirect)
}

// RegisterHandlers registers the OAuth handlers with the provided ServeMux
func (h *KeycloakOAuthHandler) RegisterHandlers(mux *http.ServeMux) {
	mux.HandleFunc("/auth/keycloak/login", h.LoginHandler)
	mux.HandleFunc("/auth/keycloak/callback", h.CallbackHandler)
	mux.HandleFunc("/auth/keycloak/logout", h.LogoutHandler)
}
