package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"

	"securedesign/webauthn"
)

func main() {
	// Create WebAuthn service
	service, err := webauthn.NewService(
		"localhost",                // RPID - your domain
		"http://localhost:8080",    // RPOrigin - your origin
		"WebAuthn Example Service", // RPDisplayName - your service name
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

	// Serve static files
	workDir, _ := os.Getwd()
	staticDir := filepath.Join(workDir, "webauthn/example/static")
	fs := http.FileServer(http.Dir(staticDir))
	mux.Handle("/", fs)

	// Start server
	log.Println("Starting server on :8080")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
