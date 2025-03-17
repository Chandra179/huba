# Keycloak SSO Sequence Diagram

## Authentication Flow

1. User accesses a protected resource in the web application
2. The KeycloakAuthMiddleware checks for a valid session cookie
3. If no valid session is found, the user is redirected to the login page
4. User clicks "Login with Keycloak" button
5. The application calls the KeycloakOAuthHandler's LoginHandler
6. The handler generates a state token for CSRF protection
7. The user is redirected to the Keycloak login URL
8. Keycloak displays the login form to the user
9. User enters credentials
10. Keycloak validates the credentials
11. Keycloak redirects back to the application's callback URL with an authorization code
12. The application's CallbackHandler validates the state token
13. The handler exchanges the authorization code for an access token
14. The handler requests user information from Keycloak using the token
15. Keycloak returns user information (ID, email, name, roles)
16. The SessionManager saves the user session and sets a session cookie
17. The user is redirected to the home page, now authenticated

## Protected Resource Access

1. User accesses a protected resource
2. The RequireAuth middleware checks for a valid session cookie
3. The middleware validates the session and adds user info to the request context
4. The protected resource is displayed to the user

## Role-Based Access

1. User attempts to access a role-protected resource
2. The RequireRole middleware checks if the user has the required role
3. If the user has the role, access is granted
4. If the user lacks the role, a 403 Forbidden error is returned

## Logout Flow

1. User clicks "Logout"
2. The LogoutHandler clears the session
3. The session cookie is removed
4. The user is redirected to the Keycloak logout endpoint
5. Keycloak completes the logout process
6. The user is redirected back to the application's home page
7. The unauthenticated home page is displayed

## Key Components

- **KeycloakOAuthHandler**: Manages the OAuth flow with Keycloak
- **KeycloakAuthMiddleware**: Provides authentication and authorization middleware
- **SessionManager**: Handles user session management
- **Keycloak Server**: External identity provider that authenticates users 