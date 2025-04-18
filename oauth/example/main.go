package oauth

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"securedesign/oauth"
)

// ExampleSetup demonstrates how to set up Google OAuth2 authentication
func ExampleSetup() {
	// Create a session manager
	sessionManager := oauth.NewDefaultSessionManager(
		os.Getenv("SESSION_COOKIE_NAME"), // Cookie name from env
		os.Getenv("COOKIE_DOMAIN"),       // Cookie domain from env
		os.Getenv("COOKIE_PATH"),         // Cookie path from env
		24*60*60,                         // Cookie max age (24 hours)
		true,                             // Secure cookie (requires HTTPS)
		true,                             // HTTP only
	)

	// Create Google OAuth config with values from environment variables
	googleConfig := oauth.GoogleOAuthConfig{
		ClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		ClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		RedirectURL:  os.Getenv("GOOGLE_REDIRECT_URL"),
		Scopes: []string{
			"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/userinfo.profile",
		},
	}

	// Create Google OAuth handler
	googleHandler := oauth.NewGoogleOAuthHandler(googleConfig, sessionManager)

	// Create auth middleware
	authMiddleware := oauth.NewAuthMiddleware(os.Getenv("SESSION_COOKIE_NAME"), "/auth/google/login")

	// Create a new ServeMux
	mux := http.NewServeMux()

	// Register OAuth handlers
	googleHandler.RegisterHandlers(mux)

	// Example of a public route (no authentication required)
	mux.Handle("/", authMiddleware.OptionalAuth(http.HandlerFunc(publicHandler)))

	// Example of a protected route (authentication required)
	mux.Handle("/dashboard", authMiddleware.RequireAuth(http.HandlerFunc(dashboardHandler)))

	// Start the server
	log.Println("Starting server on :8080")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}

// publicHandler handles public routes
func publicHandler(w http.ResponseWriter, r *http.Request) {
	// Get user from context (if authenticated)
	user := oauth.GetUserFromContext(r.Context())

	if user != nil {
		// User is authenticated
		fmt.Fprintf(w, "Hello, %s! You are logged in with email %s.", user.Name, user.Email)
		fmt.Fprintf(w, `<br><a href="/dashboard">Go to Dashboard</a>`)
		fmt.Fprintf(w, `<br><a href="/auth/logout">Logout</a>`)
	} else {
		// User is not authenticated
		fmt.Fprintf(w, "Hello, Guest! You are not logged in.")
		fmt.Fprintf(w, `<br><a href="/auth/google/login">Login with Google</a>`)
	}
}

// dashboardHandler handles protected routes
func dashboardHandler(w http.ResponseWriter, r *http.Request) {
	// Get user from context (will always be available due to RequireAuth middleware)
	user := oauth.GetUserFromContext(r.Context())

	fmt.Fprintf(w, "Welcome to your Dashboard, %s!", user.Name)
	fmt.Fprintf(w, `<br>Your email: %s`, user.Email)
	fmt.Fprintf(w, `<br>Your ID: %s`, user.ID)
	fmt.Fprintf(w, `<br><a href="/">Go to Home</a>`)
	fmt.Fprintf(w, `<br><a href="/auth/logout">Logout</a>`)
}
