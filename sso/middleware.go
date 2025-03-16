package sso

import (
	"context"
	"net/http"
)

// contextKey is a custom type for context keys
type contextKey string

// UserContextKey is the key used to store user info in the request context
const UserContextKey contextKey = "user"

// AuthMiddleware is a middleware that checks if the user is authenticated
type AuthMiddleware struct {
	SessionManager SessionManager
	// Optional redirect URL for unauthenticated users
	RedirectURL string
}

// NewAuthMiddleware creates a new AuthMiddleware
func NewAuthMiddleware(sessionManager SessionManager, redirectURL string) *AuthMiddleware {
	return &AuthMiddleware{
		SessionManager: sessionManager,
		RedirectURL:    redirectURL,
	}
}

// RequireAuth is a middleware that requires authentication
func (m *AuthMiddleware) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get the user profile from the session
		profile, err := m.SessionManager.GetSession(r)
		if err != nil {
			// No session found, user is not authenticated
			if m.RedirectURL != "" {
				http.Redirect(w, r, m.RedirectURL, http.StatusTemporaryRedirect)
				return
			}
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Add user profile to the request context
		ctx := context.WithValue(r.Context(), UserContextKey, profile)

		// Call the next handler with the updated context
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// OptionalAuth is a middleware that adds user info to the context if available
// but doesn't require authentication
func (m *AuthMiddleware) OptionalAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get the user profile from the session
		profile, err := m.SessionManager.GetSession(r)
		if err == nil {
			// Session found, add user profile to the context
			ctx := context.WithValue(r.Context(), UserContextKey, profile)
			r = r.WithContext(ctx)
		}

		// Call the next handler
		next.ServeHTTP(w, r)
	})
}

// GetUserFromContext retrieves the user profile from the request context
func GetUserFromContext(ctx context.Context) *UserProfile {
	user, ok := ctx.Value(UserContextKey).(*UserProfile)
	if !ok {
		return nil
	}
	return user
}
