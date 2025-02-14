package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"
)

// Define a custom type for context keys to avoid collisions.
type key int

const (
    // requestIDKey is used to store the request ID in the context.
    requestIDKey key = 0
)

// withRequestID middleware adds a unique request ID to the context.
// In production, you might use a UUID generator for a globally unique ID.
func withRequestID(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // For demo purposes, we use the current time in nanoseconds as a request ID.
        reqID := fmt.Sprintf("%d", time.Now().UnixNano())
        ctx := context.WithValue(r.Context(), requestIDKey, reqID)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}

// loggingMiddleware logs basic request information and execution time.
func loggingMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        start := time.Now()
        // Call the next handler in the chain.
        next.ServeHTTP(w, r)
        duration := time.Since(start)
        // Retrieve the request ID from context.
        reqID, _ := r.Context().Value(requestIDKey).(string)
        log.Printf("RequestID=%s Method=%s URL=%s Duration=%s", reqID, r.Method, r.URL.Path, duration)
    })
}

// recoveryMiddleware catches panics in downstream handlers and returns a 500 error.
func recoveryMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        defer func() {
            if err := recover(); err != nil {
                log.Printf("Recovered from panic: %v", err)
                http.Error(w, "Internal Server Error", http.StatusInternalServerError)
            }
        }()
        next.ServeHTTP(w, r)
    })
}

// mainHandler is a simple HTTP handler for demonstration.
func mainHandler(w http.ResponseWriter, r *http.Request) {
    // Uncomment the next line to simulate a panic.
    // panic("something went wrong")
    w.Write([]byte("Hello, Production-Grade HTTP Interceptor!"))
}

func main() {
    // Build the base handler.
    baseHandler := http.HandlerFunc(mainHandler)
    
    // Chain the middlewares: Recovery -> RequestID -> Logging -> Base Handler.
    handler := recoveryMiddleware(withRequestID(loggingMiddleware(baseHandler)))
    
    // Register the chained handler.
    http.Handle("/", handler)
    
    // Start the server.
    log.Println("Server starting on :8080")
    if err := http.ListenAndServe(":8080", nil); err != nil {
        log.Fatalf("Server failed: %v", err)
    }
}
