package http

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"
)

type Config struct {
	Port         string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
	CertFile     string
	KeyFile      string
}

type healthStatus struct {
	healthy atomic.Bool
}

func (hs *healthStatus) setHealth(healthy bool) {
	hs.healthy.Store(healthy)
}

func (hs *healthStatus) isHealthy() bool {
	return hs.healthy.Load()
}

func SecurityHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		w.Header().Set("Content-Security-Policy", "default-src 'self'")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		next.ServeHTTP(w, r)
	})
}

func NewHttpsServer() {
	config := Config{
		Port:         ":443",
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
		CertFile:     "/path/to/cert.pem",
		KeyFile:      "/path/to/key.pem",
	}

	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS13,
		CurvePreferences: []tls.CurveID{
			tls.X25519,
			tls.CurveP256,
		},
		PreferServerCipherSuites: true,
		CipherSuites: []uint16{
			tls.TLS_AES_128_GCM_SHA256,
			tls.TLS_AES_256_GCM_SHA384,
			tls.TLS_CHACHA20_POLY1305_SHA256,
		},
	}

	health := &healthStatus{}
	health.setHealth(true)

	mux := http.NewServeMux()

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		if !health.isHealthy() {
			w.WriteHeader(http.StatusServiceUnavailable)
			json.NewEncoder(w).Encode(map[string]string{"status": "unhealthy"})
			return
		}
		json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
	})

	// Create rate limiter: 10 requests per second per IP
	rateLimiter := NewRateLimiter(10, time.Second)

	// Build handler chain
	var handler http.Handler = mux
	handler = RateLimiterMiddleware(rateLimiter)(handler)
	handler = LoggingMiddleware(handler)
	handler = SecurityHeadersMiddleware(handler)

	server := &http.Server{
		Addr:         config.Port,
		Handler:      handler,
		TLSConfig:    tlsConfig,
		ReadTimeout:  config.ReadTimeout,
		WriteTimeout: config.WriteTimeout,
		IdleTimeout:  config.IdleTimeout,
		ErrorLog:     log.New(os.Stderr, "server: ", log.LstdFlags|log.Lshortfile),
	}

	// Start server
	go func() {
		log.Printf("Starting server on port %s", config.Port)
		if err := server.ListenAndServeTLS(config.CertFile, config.KeyFile); err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	// Mark as unhealthy during shutdown
	health.setHealth(false)

	// Graceful shutdown
	log.Println("Server is shutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited properly")
}
