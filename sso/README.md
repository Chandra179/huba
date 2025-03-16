# SSO Package for Golang

This package provides a flexible and extensible Single Sign-On (SSO) implementation for Golang applications. It supports multiple authentication providers and can be easily integrated into any web application.

## Features

- Multiple provider support (Google, GitHub, etc.)
- Standardized user profile across providers
- Session management
- Authentication middleware
- CSRF protection with state tokens
- Customizable redirect URLs

## Installation

```bash
go get github.com/yourusername/sso
```

## Usage

### Basic Setup

```go
package main

import (
    "log"
    "net/http"
    "os"

    "securedesign/sso"
)

func main() {
    // Create a session manager
    sessionManager := sso.NewCookieSessionManager(
        "auth_session", // Cookie name
        "",             // Cookie domain
        "/",            // Cookie path
        24*60*60,       // Cookie max age (24 hours)
        true,           // Secure cookie (requires HTTPS)
        true,           // HTTP only
    )

    // Create SSO handler
    ssoHandler := sso.NewSSOHandler(sessionManager, "/")

    // Register Google provider
    googleConfig := sso.ProviderConfig{
        ClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
        ClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
        RedirectURL:  "http://localhost:8080/auth/callback?provider=google",
    }
    ssoHandler.RegisterProvider(sso.NewGoogleProvider(googleConfig))

    // Create auth middleware
    authMiddleware := sso.NewAuthMiddleware(sessionManager, "/auth/login?provider=google")

    // Create a new ServeMux
    mux := http.NewServeMux()

    // Register SSO handlers
    ssoHandler.RegisterHandlers(mux)

    // Example of a protected route
    mux.Handle("/dashboard", authMiddleware.RequireAuth(http.HandlerFunc(dashboardHandler)))

    // Start the server
    log.Println("Starting server on :8080")
    if err := http.ListenAndServe(":8080", mux); err != nil {
        log.Fatalf("Server failed to start: %v", err)
    }
}

func dashboardHandler(w http.ResponseWriter, r *http.Request) {
    user := sso.GetUserFromContext(r.Context())
    // Use user information
}
```

### Environment Variables

Create a `.env` file with the following variables:

```
SESSION_COOKIE_NAME=auth_session
COOKIE_DOMAIN=
COOKIE_PATH=/
GOOGLE_CLIENT_ID=your-google-client-id
GOOGLE_CLIENT_SECRET=your-google-client-secret
GOOGLE_REDIRECT_URL=http://localhost:8080/auth/callback?provider=google
GITHUB_CLIENT_ID=your-github-client-id
GITHUB_CLIENT_SECRET=your-github-client-secret
GITHUB_REDIRECT_URL=http://localhost:8080/auth/callback?provider=github
```

### Adding a Custom Provider

You can add a custom provider by implementing the `Provider` interface:

```go
type MyProvider struct {
    // Your provider implementation
}

func (p *MyProvider) Name() string {
    return "myprovider"
}

func (p *MyProvider) GetAuthURL(state string) string {
    // Return the authorization URL
}

func (p *MyProvider) HandleCallback(ctx context.Context, r *http.Request) (*UserProfile, error) {
    // Handle the callback and return a user profile
}

// Register your provider
ssoHandler.RegisterProvider(&MyProvider{})
```

## Security Considerations

- Always use HTTPS in production
- Store session data securely
- Implement proper CSRF protection
- Validate and sanitize all user input
- Consider using a more secure session store in production (e.g., Redis)

## License

MIT 