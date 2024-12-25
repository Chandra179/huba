package proxy

import (
	"net/http"
	"net/url"
)

// Handler defines the common interface for all proxy types
type Handler interface {
	ServeHTTP(w http.ResponseWriter, r *http.Request)
}

// Config holds common configuration for all proxy types
type Config struct {
	Target    *url.URL
	Logger    Logger
	TLSConfig TLSConfig
}

type TLSConfig struct {
	CertFile           string
	KeyFile            string
	InsecureSkipVerify bool
}

// Logger interface allows for custom logging implementations
type Logger interface {
	Info(format string, v ...interface{})
	Error(format string, v ...interface{})
}
