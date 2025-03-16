# Keycloak SSO Integration

This package provides integration with Keycloak for Single Sign-On (SSO) authentication in Go applications.

## Features

- OAuth 2.0 and OpenID Connect (OIDC) integration with Keycloak
- User authentication and session management
- Role-based access control
- Token validation and introspection
- Middleware for protecting routes

## Setup

### 1. Install and Configure Keycloak

1. Download and install Keycloak from [https://www.keycloak.org/downloads](https://www.keycloak.org/downloads)
2. Start Keycloak server:
   ```
   bin/kc.sh start-dev
   ```
3. Access the Keycloak admin console at `http://localhost:8080/admin`
4. Create a new realm (e.g., `myrealm`)
5. Create a new client:
   - Client ID: `myclient`
   - Client Protocol: `openid-connect`
   - Access Type: `confidential`
   - Valid Redirect URIs: `http://localhost:3000/auth/keycloak/callback`
6. After saving, go to the "Credentials" tab to get the client secret
7. Create roles (e.g., `admin`, `user`)
8. Create users and assign roles

### 2. Configure Your Application

Set the following environment variables or use the default values in your application:

```
KEYCLOAK_BASE_URL=http://localhost:8080
KEYCLOAK_REALM=myrealm
KEYCLOAK_CLIENT_ID=myclient
KEYCLOAK_CLIENT_SECRET=your-client-secret
KEYCLOAK_REDIRECT_URL=http://localhost:3000/auth/keycloak/callback
PORT=3000
```

## Usage

### Basic Integration

```go
package main

import (
	"log"
	"net/http"
	"os"

	"securedesign/oauth"
	"securedesign/sso/keycloak"
)

func main() {
	// Create Keycloak configuration
	keycloakConfig := keycloak.KeycloakConfig{
		BaseURL:      os.Getenv("KEYCLOAK_BASE_URL"),
		Realm:        os.Getenv("KEYCLOAK_REALM"),
		ClientID:     os.Getenv("KEYCLOAK_CLIENT_ID"),
		ClientSecret: os.Getenv("KEYCLOAK_CLIENT_SECRET"),
		RedirectURL:  os.Getenv("KEYCLOAK_REDIRECT_URL"),
		Scopes:       []string{"openid", "profile", "email"},
	}

	// Create session manager
	sessionManager := oauth.NewDefaultSessionManager(
		"keycloak_session", // Cookie name
		"",                 // Cookie domain
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

	// Public route
	mux.Handle("/", http.HandlerFunc(publicHandler))

	// Protected route (requires authentication)
	mux.Handle("/protected", authMiddleware.RequireAuth(http.HandlerFunc(protectedHandler)))

	// Admin route (requires admin role)
	mux.Handle("/admin", authMiddleware.RequireRole("admin", http.HandlerFunc(adminHandler)))

	// Start HTTP server
	log.Fatal(http.ListenAndServe(":3000", mux))
}
```

### Complete Example

See the [example](./example) directory for a complete working example.

## API Reference

### Configuration

- `KeycloakConfig`: Configuration for Keycloak OAuth
  - `BaseURL`: Base URL of the Keycloak server
  - `Realm`: Keycloak realm name
  - `ClientID`: Client ID
  - `ClientSecret`: Client secret
  - `RedirectURL`: Redirect URL after authentication
  - `Scopes`: OAuth scopes

### Handlers

- `KeycloakOAuthHandler`: Handles Keycloak OAuth2 authentication
  - `LoginHandler`: Initiates the Keycloak OAuth flow
  - `CallbackHandler`: Handles the callback from Keycloak OAuth
  - `LogoutHandler`: Handles user logout
  - `RegisterHandlers`: Registers the OAuth handlers with the provided ServeMux

### Middleware

- `KeycloakAuthMiddleware`: Middleware for authentication and authorization
  - `RequireAuth`: Middleware that requires authentication
  - `RequireRole`: Middleware that requires a specific role
  - `OptionalAuth`: Middleware that adds user info to the context if available

### User Info

- `KeycloakUserInfo`: Represents the authenticated user information with roles
  - `ID`: User ID
  - `Email`: User email
  - `Name`: User name
  - `Roles`: User roles

### Utility Functions

- `GetUserFromContext`: Retrieves the user info from the request context
- `GetUserRoles`: Returns the roles assigned to the user
- `ValidateToken`: Validates the token with Keycloak

## Security Considerations

1. Always use HTTPS in production
2. Set the `Secure` flag to `true` for cookies in production
3. Use a strong client secret
4. Regularly rotate client secrets
5. Validate tokens on the server side
6. Implement proper session management
7. Use role-based access control for sensitive operations

## License

This package is licensed under the MIT License. 