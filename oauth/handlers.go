package oauth

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

// SessionManager interface for managing user sessions
type SessionManager interface {
	SaveSession(w http.ResponseWriter, userID string, email string, name string) error
	ClearSession(w http.ResponseWriter) error
}

// DefaultSessionManager is a simple implementation of SessionManager using cookies
type DefaultSessionManager struct {
	CookieName   string
	CookieDomain string
	CookiePath   string
	CookieMaxAge int
	SecureCookie bool
	HTTPOnly     bool
}

// SaveSession saves the user session as a cookie
func (sm *DefaultSessionManager) SaveSession(w http.ResponseWriter, userID string, email string, name string) error {
	// Create a simple session data structure
	sessionData := map[string]string{
		"user_id": userID,
		"email":   email,
		"name":    name,
	}

	// Serialize to JSON
	jsonData, err := json.Marshal(sessionData)
	if err != nil {
		return err
	}

	// Create and set the cookie
	cookie := &http.Cookie{
		Name:     sm.CookieName,
		Value:    string(jsonData),
		Domain:   sm.CookieDomain,
		Path:     sm.CookiePath,
		MaxAge:   sm.CookieMaxAge,
		Secure:   sm.SecureCookie,
		HttpOnly: sm.HTTPOnly,
		SameSite: http.SameSiteLaxMode,
	}
	http.SetCookie(w, cookie)
	return nil
}

// ClearSession removes the session cookie
func (sm *DefaultSessionManager) ClearSession(w http.ResponseWriter) error {
	cookie := &http.Cookie{
		Name:     sm.CookieName,
		Value:    "",
		Domain:   sm.CookieDomain,
		Path:     sm.CookiePath,
		MaxAge:   -1,
		Secure:   sm.SecureCookie,
		HttpOnly: sm.HTTPOnly,
		SameSite: http.SameSiteLaxMode,
	}
	http.SetCookie(w, cookie)
	return nil
}

// NewDefaultSessionManager creates a new DefaultSessionManager
func NewDefaultSessionManager(cookieName, cookieDomain, cookiePath string, maxAge int, secure, httpOnly bool) *DefaultSessionManager {
	return &DefaultSessionManager{
		CookieName:   cookieName,
		CookieDomain: cookieDomain,
		CookiePath:   cookiePath,
		CookieMaxAge: maxAge,
		SecureCookie: secure,
		HTTPOnly:     httpOnly,
	}
}

// GoogleOAuthHandler handles Google OAuth2 authentication
type GoogleOAuthHandler struct {
	Config         GoogleOAuthConfig
	SessionManager SessionManager
	StateStore     map[string]time.Time // Simple in-memory state storage
}

// NewGoogleOAuthHandler creates a new GoogleOAuthHandler
func NewGoogleOAuthHandler(config GoogleOAuthConfig, sessionManager SessionManager) *GoogleOAuthHandler {
	return &GoogleOAuthHandler{
		Config:         config,
		SessionManager: sessionManager,
		StateStore:     make(map[string]time.Time),
	}
}

// LoginHandler initiates the Google OAuth flow
func (h *GoogleOAuthHandler) LoginHandler(w http.ResponseWriter, r *http.Request) {
	// Generate a state token for CSRF protection
	state, err := GenerateStateToken()
	if err != nil {
		http.Error(w, "Failed to generate state token", http.StatusInternalServerError)
		return
	}

	// Store the state token with an expiration time (e.g., 10 minutes)
	h.StateStore[state] = time.Now().Add(10 * time.Minute)

	// Create the OAuth2 config
	oauthConfig := NewGoogleOAuth(h.Config)

	// Redirect to Google's OAuth 2.0 server
	url := GetGoogleLoginURL(oauthConfig, state)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

// CallbackHandler handles the callback from Google OAuth
func (h *GoogleOAuthHandler) CallbackHandler(w http.ResponseWriter, r *http.Request) {
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
	oauthConfig := NewGoogleOAuth(h.Config)

	// Exchange the authorization code for a token
	token, err := HandleGoogleCallback(r.Context(), oauthConfig, state, code)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to exchange token: %v", err), http.StatusInternalServerError)
		return
	}

	// Get the user info
	userInfo, err := GetGoogleUserInfo(r.Context(), token, oauthConfig)
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
func (h *GoogleOAuthHandler) LogoutHandler(w http.ResponseWriter, r *http.Request) {
	// Clear the session
	err := h.SessionManager.ClearSession(w)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to clear session: %v", err), http.StatusInternalServerError)
		return
	}

	// Redirect to the home page
	http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
}

// RegisterHandlers registers the OAuth handlers with the provided ServeMux
func (h *GoogleOAuthHandler) RegisterHandlers(mux *http.ServeMux) {
	mux.HandleFunc("/auth/google/login", h.LoginHandler)
	mux.HandleFunc("/auth/google/callback", h.CallbackHandler)
	mux.HandleFunc("/auth/logout", h.LogoutHandler)
}
