package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"securedesign/oauth"
	"securedesign/sso/keycloak"
)

func main() {
	// Get Keycloak configuration from environment variables
	keycloakBaseURL := getEnv("KEYCLOAK_BASE_URL", "http://localhost:8080")
	keycloakRealm := getEnv("KEYCLOAK_REALM", "myrealm")
	keycloakClientID := getEnv("KEYCLOAK_CLIENT_ID", "myclient")
	keycloakClientSecret := getEnv("KEYCLOAK_CLIENT_SECRET", "")
	keycloakRedirectURL := getEnv("KEYCLOAK_REDIRECT_URL", "http://localhost:3000/auth/keycloak/callback")

	// Create Keycloak configuration
	keycloakConfig := keycloak.KeycloakConfig{
		BaseURL:      keycloakBaseURL,
		Realm:        keycloakRealm,
		ClientID:     keycloakClientID,
		ClientSecret: keycloakClientSecret,
		RedirectURL:  keycloakRedirectURL,
		Scopes:       []string{"openid", "profile", "email"},
	}

	// Create session manager
	sessionManager := oauth.NewDefaultSessionManager(
		"keycloak_session", // Cookie name
		"",                 // Cookie domain (empty for current domain)
		"/",                // Cookie path
		3600,               // Cookie max age (1 hour)
		false,              // Secure cookie (set to true in production with HTTPS)
		true,               // HTTP only
	)

	// Create Keycloak OAuth handler
	keycloakHandler := keycloak.NewKeycloakOAuthHandler(keycloakConfig, sessionManager)

	// Create Keycloak auth middleware
	authMiddleware := keycloak.NewKeycloakAuthMiddleware(
		"keycloak_session",     // Cookie name
		"/auth/keycloak/login", // Redirect URL for unauthenticated users
		keycloakConfig,
	)

	// Create HTTP server mux
	mux := http.NewServeMux()

	// Register Keycloak OAuth handlers
	keycloakHandler.RegisterHandlers(mux)

	// Public home page
	mux.Handle("/", authMiddleware.OptionalAuth(http.HandlerFunc(homeHandler)))

	// Protected dashboard page (requires authentication)
	mux.Handle("/dashboard", authMiddleware.RequireAuth(http.HandlerFunc(dashboardHandler)))

	// Admin page (requires admin role)
	mux.Handle("/admin", authMiddleware.RequireRole("admin", http.HandlerFunc(adminHandler)))

	// Start HTTP server
	port := getEnv("PORT", "3000")
	log.Printf("Starting server on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}

// homeHandler handles the home page
func homeHandler(w http.ResponseWriter, r *http.Request) {
	// Get user info from context (if authenticated)
	userInfo := keycloak.GetUserFromContext(r.Context())

	if userInfo != nil {
		// User is authenticated
		fmt.Fprintf(w, "Welcome, %s! You are logged in.<br>", userInfo.Name)
		fmt.Fprintf(w, "Your email: %s<br>", userInfo.Email)
		fmt.Fprintf(w, "Your roles: %v<br>", userInfo.Roles)
		fmt.Fprintf(w, "<a href=\"/dashboard\">Go to Dashboard</a><br>")
		fmt.Fprintf(w, "<a href=\"/admin\">Go to Admin Page</a><br>")
		fmt.Fprintf(w, "<a href=\"/auth/keycloak/logout\">Logout</a>")
	} else {
		// User is not authenticated
		fmt.Fprintf(w, "Welcome, Guest!<br>")
		fmt.Fprintf(w, "<a href=\"/auth/keycloak/login\">Login with Keycloak</a>")
	}
}

// dashboardHandler handles the dashboard page (protected)
func dashboardHandler(w http.ResponseWriter, r *http.Request) {
	// Get user info from context
	userInfo := keycloak.GetUserFromContext(r.Context())

	fmt.Fprintf(w, "Dashboard - Welcome, %s!<br>", userInfo.Name)
	fmt.Fprintf(w, "Your email: %s<br>", userInfo.Email)
	fmt.Fprintf(w, "Your roles: %v<br>", userInfo.Roles)
	fmt.Fprintf(w, "<a href=\"/\">Go to Home</a><br>")
	fmt.Fprintf(w, "<a href=\"/auth/keycloak/logout\">Logout</a>")
}

// adminHandler handles the admin page (protected and requires admin role)
func adminHandler(w http.ResponseWriter, r *http.Request) {
	// Get user info from context
	userInfo := keycloak.GetUserFromContext(r.Context())

	fmt.Fprintf(w, "Admin Page - Welcome, %s!<br>", userInfo.Name)
	fmt.Fprintf(w, "Your email: %s<br>", userInfo.Email)
	fmt.Fprintf(w, "Your roles: %v<br>", userInfo.Roles)
	fmt.Fprintf(w, "<a href=\"/\">Go to Home</a><br>")
	fmt.Fprintf(w, "<a href=\"/dashboard\">Go to Dashboard</a><br>")
	fmt.Fprintf(w, "<a href=\"/auth/keycloak/logout\">Logout</a>")
}

// getEnv gets an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}
