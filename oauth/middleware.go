package oauth

import (
	"context"
	"encoding/json"
	"net/http"
)

// UserInfo represents the authenticated user information
type UserInfo struct {
	ID    string
	Email string
	Name  string
}

// contextKey is a custom type for context keys
type contextKey string

// UserContextKey is the key used to store user info in the request context
const UserContextKey contextKey = "user"

// AuthMiddleware is a middleware that checks if the user is authenticated
type AuthMiddleware struct {
	CookieName string
	// Optional redirect URL for unauthenticated users
	RedirectURL string
}

// NewAuthMiddleware creates a new AuthMiddleware
func NewAuthMiddleware(cookieName string, redirectURL string) *AuthMiddleware {
	return &AuthMiddleware{
		CookieName:  cookieName,
		RedirectURL: redirectURL,
	}
}

// RequireAuth is a middleware that requires authentication
func (m *AuthMiddleware) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get the session cookie
		cookie, err := r.Cookie(m.CookieName)
		if err != nil {
			// No cookie found, user is not authenticated
			if m.RedirectURL != "" {
				http.Redirect(w, r, m.RedirectURL, http.StatusTemporaryRedirect)
				return
			}
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Parse the cookie value
		var sessionData map[string]string
		if err := json.Unmarshal([]byte(cookie.Value), &sessionData); err != nil {
			// Invalid cookie format
			if m.RedirectURL != "" {
				http.Redirect(w, r, m.RedirectURL, http.StatusTemporaryRedirect)
				return
			}
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Check if the required fields are present
		userID, hasUserID := sessionData["user_id"]
		email, hasEmail := sessionData["email"]
		name, hasName := sessionData["name"]

		if !hasUserID || !hasEmail || !hasName {
			// Missing required fields
			if m.RedirectURL != "" {
				http.Redirect(w, r, m.RedirectURL, http.StatusTemporaryRedirect)
				return
			}
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Create user info
		userInfo := &UserInfo{
			ID:    userID,
			Email: email,
			Name:  name,
		}

		// Add user info to the request context
		ctx := context.WithValue(r.Context(), UserContextKey, userInfo)

		// Call the next handler with the updated context
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetUserFromContext retrieves the user info from the request context
func GetUserFromContext(ctx context.Context) *UserInfo {
	user, ok := ctx.Value(UserContextKey).(*UserInfo)
	if !ok {
		return nil
	}
	return user
}

// OptionalAuth is a middleware that adds user info to the context if available
// but doesn't require authentication
func (m *AuthMiddleware) OptionalAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get the session cookie
		cookie, err := r.Cookie(m.CookieName)
		if err == nil {
			// Cookie found, try to parse it
			var sessionData map[string]string
			if err := json.Unmarshal([]byte(cookie.Value), &sessionData); err == nil {
				// Check if the required fields are present
				userID, hasUserID := sessionData["user_id"]
				email, hasEmail := sessionData["email"]
				name, hasName := sessionData["name"]

				if hasUserID && hasEmail && hasName {
					// Create user info
					userInfo := &UserInfo{
						ID:    userID,
						Email: email,
						Name:  name,
					}

					// Add user info to the request context
					ctx := context.WithValue(r.Context(), UserContextKey, userInfo)
					r = r.WithContext(ctx)
				}
			}
		}

		// Call the next handler
		next.ServeHTTP(w, r)
	})
}
