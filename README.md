# GoLib: Secure, Modular Go Library for Modern Web Applications

GoLib is a comprehensive, modular Go library for implementing secure authentication, authorization, distributed caching, observability, and scalable patterns in web applications and microservices.

---

## Features

- **OAuth 2.0 & SSO**: Integrations for Google OAuth, Keycloak, and OpenID Connect.
- **WebAuthn**: Passwordless authentication using the Web Authentication API.
- **Kafka**: Producer/consumer utilities for Apache Kafka.
- **OpenTelemetry (OTEL)**: Tracing, metrics, and logging for observability.
- **Cryptographic Utilities**: ECDSA, HMAC, and more.
- **Worker Pool**: Efficient concurrent processing.
- **Distributed Cache (Redis)**: Cache-aside pattern, distributed locking, and rate limiting.
- **Modular Design**: Each feature is provided as a standalone Go package.
- **Extensive Examples**: Each module includes runnable examples.
- **Docker Support**: Easily run dependencies and the library in containers.

---

## Module Overview

| Module         | Description                                                      |
|----------------|------------------------------------------------------------------|
| `oauth`        | OAuth 2.0 client, Google integration, middleware, handlers        |
| `webauthn`     | Passwordless authentication, user store, HTTP handlers           |
| `sso`          | SSO providers (Keycloak), session management, RBAC               |
| `kafka`        | Kafka producer/consumer, message utilities                       |
| `otel`         | OpenTelemetry tracing, metrics, logging                          |
| `cryptoutils`  | ECDSA, HMAC, and cryptographic helpers                           |
| `workerpool`   | Goroutine pool for concurrent task processing                    |
| `cache`        | Redis-based distributed cache, locking, rate limiting            |

See each module's `README.md` or `example/` directory for details and usage.

---

## Getting Started

### Prerequisites

- Go 1.24 or higher
- Docker (for running examples with dependencies)

### Installation

```bash
git clone https://github.com/yourusername/securedesign.git
cd securedesign
go mod download
```

### Running Examples

Each module contains an `example` directory with sample implementations:

```bash
cd webauthn/example && go run main.go
cd oauth/example && go run main.go
cd otel/example && go run main.go
cd kafka/example && go run main.go
cd cache/example && go run main.go
```

---

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

---

## Configuration

Configuration is handled through environment variables. See `.env.example` for available options.

---

## Security Best Practices

- Always use HTTPS in production
- Set the `Secure` flag to `true` for cookies in production
- Use strong client secrets and rotate them regularly
- Validate tokens on the server side
- Implement proper session management
- Use role-based access control for sensitive operations

---

## Distributed Cache System with Redis

The `cache` package provides a Go implementation of a distributed cache system using Redis.

### Features

- Basic cache operations (Get, Set, Delete, Exists)
- Cache-aside pattern for transparent loading from a data source
- Distributed locking for coordinating access across services
- Rate limiting with sliding window algorithm

### Requirements

- Go 1.24 or later
- Redis server (v6.0 or later recommended)

### Running Redis with Docker

The project includes a Docker Compose configuration for running Redis:

```bash
docker-compose up -d redis
# Or use the management script
./cache/redis-management.sh start
```

The Redis container is configured with:

- Port: 6379 (default Redis port)
- Persistence: AOF (Append Only File) enabled
- Optional password protection (set via REDIS_PASSWORD environment variable)

Management script commands:

```bash
./cache/redis-management.sh start   # Start Redis container
./cache/redis-management.sh stop    # Stop Redis container
./cache/redis-management.sh cli     # Open Redis CLI
./cache/redis-management.sh flush   # Flush all Redis data
./cache/redis-management.sh logs    # View Redis logs
./cache/redis-management.sh info    # Show Redis info
```