package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"securedesign/sso"
)

func main() {
	// Create a session manager
	sessionManager := sso.NewCookieSessionManager(
		os.Getenv("SESSION_COOKIE_NAME"), // Cookie name from env
		os.Getenv("COOKIE_DOMAIN"),       // Cookie domain from env
		os.Getenv("COOKIE_PATH"),         // Cookie path from env
		24*60*60,                         // Cookie max age (24 hours)
		true,                             // Secure cookie (requires HTTPS)
		true,                             // HTTP only
	)

	// Create SSO handler
	ssoHandler := sso.NewSSOHandler(sessionManager, "/")

	// Register Google provider
	googleConfig := sso.ProviderConfig{
		ClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		ClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		RedirectURL:  os.Getenv("GOOGLE_REDIRECT_URL"),
		Scopes: []string{
			"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/userinfo.profile",
		},
	}
	ssoHandler.RegisterProvider(sso.NewGoogleProvider(googleConfig))

	// Register GitHub provider
	githubConfig := sso.ProviderConfig{
		ClientID:     os.Getenv("GITHUB_CLIENT_ID"),
		ClientSecret: os.Getenv("GITHUB_CLIENT_SECRET"),
		RedirectURL:  os.Getenv("GITHUB_REDIRECT_URL"),
		Scopes:       []string{"user:email", "read:user"},
	}
	ssoHandler.RegisterProvider(sso.NewGitHubProvider(githubConfig))

	// Create auth middleware
	authMiddleware := sso.NewAuthMiddleware(sessionManager, "/auth/login?provider=google")

	// Create a new ServeMux
	mux := http.NewServeMux()

	// Register SSO handlers
	ssoHandler.RegisterHandlers(mux)

	// Public home page
	mux.Handle("/", authMiddleware.OptionalAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := sso.GetUserFromContext(r.Context())

		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, `<!DOCTYPE html>
<html>
<head>
    <title>SSO Demo</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 40px; line-height: 1.6; }
        .container { max-width: 800px; margin: 0 auto; }
        .card { border: 1px solid #ddd; border-radius: 8px; padding: 20px; margin-bottom: 20px; }
        .btn { display: inline-block; padding: 10px 15px; background-color: #4285f4; color: white; 
               text-decoration: none; border-radius: 4px; margin-right: 10px; }
        .btn-github { background-color: #333; }
        .btn-logout { background-color: #f44336; }
    </style>
</head>
<body>
    <div class="container">
        <h1>SSO Demo</h1>
        <div class="card">`)

		if user != nil {
			fmt.Fprintf(w, `
            <h2>Welcome, %s!</h2>
            <p>You are logged in with %s.</p>
            <p>Email: %s</p>
            <a href="/dashboard" class="btn">Go to Dashboard</a>
            <a href="/auth/logout" class="btn btn-logout">Logout</a>`,
				user.Name, user.Provider, user.Email)
		} else {
			fmt.Fprintf(w, `
            <h2>Hello, Guest!</h2>
            <p>Please login to continue.</p>
            <a href="/auth/login?provider=google" class="btn">Login with Google</a>
            <a href="/auth/login?provider=github" class="btn btn-github">Login with GitHub</a>`)
		}

		fmt.Fprintf(w, `
        </div>
    </div>
</body>
</html>`)
	})))

	// Protected dashboard
	mux.Handle("/dashboard", authMiddleware.RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := sso.GetUserFromContext(r.Context())

		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, `<!DOCTYPE html>
<html>
<head>
    <title>Dashboard - SSO Demo</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 40px; line-height: 1.6; }
        .container { max-width: 800px; margin: 0 auto; }
        .card { border: 1px solid #ddd; border-radius: 8px; padding: 20px; margin-bottom: 20px; }
        .btn { display: inline-block; padding: 10px 15px; background-color: #4285f4; color: white; 
               text-decoration: none; border-radius: 4px; margin-right: 10px; }
        .btn-logout { background-color: #f44336; }
        .profile-info { margin-bottom: 20px; }
        .profile-info p { margin: 5px 0; }
    </style>
</head>
<body>
    <div class="container">
        <h1>Dashboard</h1>
        <div class="card">
            <h2>Welcome to your Dashboard, %s!</h2>
            
            <div class="profile-info">
                <h3>Your Profile</h3>
                <p><strong>ID:</strong> %s</p>
                <p><strong>Email:</strong> %s</p>
                <p><strong>Provider:</strong> %s</p>
            </div>
            
            <a href="/" class="btn">Go to Home</a>
            <a href="/auth/logout" class="btn btn-logout">Logout</a>
        </div>
    </div>
</body>
</html>`, user.Name, user.ID, user.Email, user.Provider)
	})))

	// Start the server
	log.Println("Starting SSO Demo server on :8080")
	log.Println("Open http://localhost:8080 in your browser")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
