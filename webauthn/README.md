# WebAuthn Implementation in Go

This package provides a simple WebAuthn implementation in Go using the [go-webauthn/webauthn](https://github.com/go-webauthn/webauthn) library.

## Features

- User registration with WebAuthn
- User authentication with WebAuthn
- In-memory user store
- HTTP handlers for WebAuthn operations

## Installation

```bash
go get github.com/go-webauthn/webauthn
```

## Usage

### Basic Setup

```go
package main

import (
    "log"
    "net/http"
    
    "securedesign/webauthn"
)

func main() {
    // Create WebAuthn service
    service, err := webauthn.NewService(
        "example.com",           // RPID - your domain
        "https://example.com",   // RPOrigin - your origin
        "My WebAuthn Service",   // RPDisplayName - your service name
    )
    if err != nil {
        log.Fatalf("Failed to create WebAuthn service: %v", err)
    }
    
    // Create handlers
    handlers := webauthn.NewHandlers(service)
    
    // Create a new ServeMux
    mux := http.NewServeMux()
    
    // Register WebAuthn handlers
    handlers.RegisterHandlers(mux)
    
    // Start server
    log.Println("Starting server on :8080")
    if err := http.ListenAndServe(":8080", mux); err != nil {
        log.Fatalf("Server failed to start: %v", err)
    }
}
```

### API Endpoints

The following endpoints are available:

- `POST /webauthn/register/begin` - Begin registration
- `POST /webauthn/register/finish?username=<username>` - Finish registration
- `POST /webauthn/login/begin` - Begin login
- `POST /webauthn/login/finish?username=<username>` - Finish login

### Example

An example implementation is provided in the `example` directory. To run it:

```bash
cd webauthn/example
go run main.go
```

Then open `http://localhost:8080` in your browser.

## Client-Side Integration

For client-side integration, you need to:

1. Call the begin registration/login endpoint
2. Convert base64url-encoded challenge and credential IDs to ArrayBuffers
3. Call `navigator.credentials.create()` or `navigator.credentials.get()`
4. Convert the response to JSON and send it to the finish registration/login endpoint

See the example HTML file for a complete implementation.

## Security Considerations

- This implementation uses an in-memory store, which is not suitable for production. In a real application, you should use a persistent store.
- The RPID should be your domain name.
- The RPOrigin should be the full origin of your site, including the protocol and port if applicable.
- WebAuthn requires HTTPS in production. For local development, you can use `localhost`.

## License

MIT 