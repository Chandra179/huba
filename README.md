# GoLib

A comprehensive Go library for implementing secure authentication and authorization patterns in web applications.

## Overview

SecureDesign provides a collection of security-focused modules for Go applications, including:

- OAuth 2.0 integration
- WebAuthn (passwordless authentication)
- Single Sign-On (SSO) implementations
- Kafka integration
- OpenTelemetry observability
- Cryptographic utilities
- Worker pool for concurrent processing

## Modules

### OAuth

The `oauth` package provides OAuth 2.0 client implementations for various providers:

- Google OAuth integration
- Middleware for OAuth-based authentication
- Request handlers for OAuth flows

### WebAuthn

The `webauthn` package implements passwordless authentication using the Web Authentication API:

- User registration with WebAuthn
- User authentication with WebAuthn
- In-memory user store
- HTTP handlers for WebAuthn operations

### SSO

The `sso` directory contains implementations for various Single Sign-On (SSO) providers:

- Keycloak integration with OAuth 2.0 and OpenID Connect (OIDC)
- Common features like user authentication, session management, and role-based access control

### Kafka

The `kafka` package provides utilities for working with Apache Kafka:

- Producer and consumer implementations
- Message handling utilities

### OpenTelemetry (OTEL)

The `otel` package provides OpenTelemetry integration for observability:

- Tracing
- Metrics
- Logging

### Cryptographic Utilities

The `cryptoutils` package provides utilities for cryptographic operations.

### Worker Pool

The `workerpool` package provides a worker pool implementation for concurrent processing.

## Getting Started

### Prerequisites

- Go 1.24 or higher
- Docker (for running examples with dependencies)

### Installation

```bash
# Clone the repository
git clone https://github.com/yourusername/securedesign.git
cd securedesign

# Install dependencies
go mod download
```

### Running Examples

Each module contains an `example` directory with sample implementations:

```bash
# Run WebAuthn example
cd webauthn/example
go run main.go

# Run OAuth example
cd oauth/example
go run main.go

# Run OTEL example
cd otel/example
go run main.go
```

## Development

### Building

```bash
make build
```

### Testing

```bash
go test ./...
```

### Docker

A Dockerfile is provided for containerized deployment:

```bash
docker build -t securedesign .
docker run -p 8080:8080 securedesign
```

## Configuration

Configuration is handled through environment variables. See `.env.example` for available options.

## Security Considerations

- Always use HTTPS in production
- Set the `Secure` flag to `true` for cookies in production
- Use strong client secrets and regularly rotate them
- Validate tokens on the server side
- Implement proper session management
- Use role-based access control for sensitive operations

## License

This project is licensed under the MIT License. 