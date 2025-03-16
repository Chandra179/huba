package sso

import (
	"encoding/json"
	"net/http"
	"time"
)

// SessionManager defines the interface for managing user sessions
type SessionManager interface {
	// SaveSession saves the user session
	SaveSession(w http.ResponseWriter, profile *UserProfile) error

	// GetSession retrieves the user session from the request
	GetSession(r *http.Request) (*UserProfile, error)

	// ClearSession removes the user session
	ClearSession(w http.ResponseWriter) error
}

// CookieSessionManager implements SessionManager using cookies
type CookieSessionManager struct {
	CookieName   string
	CookieDomain string
	CookiePath   string
	CookieMaxAge int
	SecureCookie bool
	HTTPOnly     bool
}

// NewCookieSessionManager creates a new CookieSessionManager
func NewCookieSessionManager(cookieName, cookieDomain, cookiePath string, maxAge int, secure, httpOnly bool) *CookieSessionManager {
	return &CookieSessionManager{
		CookieName:   cookieName,
		CookieDomain: cookieDomain,
		CookiePath:   cookiePath,
		CookieMaxAge: maxAge,
		SecureCookie: secure,
		HTTPOnly:     httpOnly,
	}
}

// SaveSession saves the user profile as a cookie
func (sm *CookieSessionManager) SaveSession(w http.ResponseWriter, profile *UserProfile) error {
	// Serialize to JSON
	jsonData, err := json.Marshal(profile)
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

// GetSession retrieves the user profile from the request cookie
func (sm *CookieSessionManager) GetSession(r *http.Request) (*UserProfile, error) {
	cookie, err := r.Cookie(sm.CookieName)
	if err != nil {
		return nil, err
	}

	var profile UserProfile
	if err := json.Unmarshal([]byte(cookie.Value), &profile); err != nil {
		return nil, err
	}

	return &profile, nil
}

// ClearSession removes the session cookie
func (sm *CookieSessionManager) ClearSession(w http.ResponseWriter) error {
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

// StateManager manages the state tokens for CSRF protection
type StateManager struct {
	states map[string]time.Time
}

// NewStateManager creates a new StateManager
func NewStateManager() *StateManager {
	return &StateManager{
		states: make(map[string]time.Time),
	}
}

// SaveState saves a state token with an expiration time
func (sm *StateManager) SaveState(state string, expiration time.Duration) {
	sm.states[state] = time.Now().Add(expiration)
}

// ValidateState checks if a state token is valid and not expired
func (sm *StateManager) ValidateState(state string) bool {
	expiration, exists := sm.states[state]
	if !exists {
		return false
	}

	// Check if the state token has expired
	if time.Now().After(expiration) {
		delete(sm.states, state)
		return false
	}

	// Remove the used state token
	delete(sm.states, state)
	return true
}
