package keycloak

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
)

// KeycloakUserInfo represents the authenticated user information with roles
type KeycloakUserInfo struct {
	ID    string
	Email string
	Name  string
	Roles []string
}

// contextKey is a custom type for context keys
type contextKey string

// UserContextKey is the key used to store user info in the request context
const UserContextKey contextKey = "keycloak_user"

// KeycloakAuthMiddleware is a middleware that checks if the user is authenticated
type KeycloakAuthMiddleware struct {
	CookieName  string
	RedirectURL string
	Config      KeycloakConfig
}

// NewKeycloakAuthMiddleware creates a new KeycloakAuthMiddleware
func NewKeycloakAuthMiddleware(cookieName string, redirectURL string, config KeycloakConfig) *KeycloakAuthMiddleware {
	return &KeycloakAuthMiddleware{
		CookieName:  cookieName,
		RedirectURL: redirectURL,
		Config:      config,
	}
}

// RequireAuth is a middleware that requires authentication
func (m *KeycloakAuthMiddleware) RequireAuth(next http.Handler) http.Handler {
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
		roles, hasRoles := sessionData["roles"]

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
		userInfo := &KeycloakUserInfo{
			ID:    userID,
			Email: email,
			Name:  name,
		}

		// Add roles if available
		if hasRoles {
			userInfo.Roles = strings.Split(roles, ",")
		}

		// Add user info to the request context
		ctx := context.WithValue(r.Context(), UserContextKey, userInfo)

		// Call the next handler with the updated context
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequireRole is a middleware that requires a specific role
func (m *KeycloakAuthMiddleware) RequireRole(role string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get the user info from the context
		userInfo, ok := r.Context().Value(UserContextKey).(*KeycloakUserInfo)
		if !ok || userInfo == nil {
			// No user info found, user is not authenticated
			if m.RedirectURL != "" {
				http.Redirect(w, r, m.RedirectURL, http.StatusTemporaryRedirect)
				return
			}
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Check if the user has the required role
		hasRole := false
		for _, r := range userInfo.Roles {
			if r == role {
				hasRole = true
				break
			}
		}

		if !hasRole {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		// Call the next handler
		next.ServeHTTP(w, r)
	})
}

// GetUserFromContext retrieves the user info from the request context
func GetUserFromContext(ctx context.Context) *KeycloakUserInfo {
	user, ok := ctx.Value(UserContextKey).(*KeycloakUserInfo)
	if !ok {
		return nil
	}
	return user
}

// OptionalAuth is a middleware that adds user info to the context if available
// but doesn't require authentication
func (m *KeycloakAuthMiddleware) OptionalAuth(next http.Handler) http.Handler {
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
				roles, hasRoles := sessionData["roles"]

				if hasUserID && hasEmail && hasName {
					// Create user info
					userInfo := &KeycloakUserInfo{
						ID:    userID,
						Email: email,
						Name:  name,
					}

					// Add roles if available
					if hasRoles {
						userInfo.Roles = strings.Split(roles, ",")
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
