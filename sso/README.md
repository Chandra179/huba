# Single Sign-On (SSO) Implementations

This directory contains implementations for various Single Sign-On (SSO) providers.

## Available Implementations

- [Keycloak](./keycloak/README.md): OAuth 2.0 and OpenID Connect (OIDC) integration with Keycloak

## Common Features

All SSO implementations provide:

- User authentication
- Session management
- Role-based access control
- Middleware for protecting routes

## Usage

Each implementation has its own README with detailed instructions on how to set up and use it.

## Security Considerations

When implementing SSO, consider the following security best practices:

1. Always use HTTPS in production
2. Set the `Secure` flag to `true` for cookies in production
3. Use strong client secrets
4. Regularly rotate client secrets
5. Validate tokens on the server side
6. Implement proper session management
7. Use role-based access control for sensitive operations

## License

This package is licensed under the MIT License. 