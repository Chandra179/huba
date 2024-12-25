package http

import (
	"log"
	"net/http"
	"time"
)

type ResponseWriter struct {
	http.ResponseWriter
	statusCode   int
	responseSize int64
}

func NewResponseWriter(w http.ResponseWriter) *ResponseWriter {
	return &ResponseWriter{w, http.StatusOK, 0}
}

func (rw *ResponseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *ResponseWriter) Write(b []byte) (int, error) {
	size, err := rw.ResponseWriter.Write(b)
	rw.responseSize += int64(size)
	return size, err
}

func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := NewResponseWriter(w)

		next.ServeHTTP(rw, r)

		duration := time.Since(start)
		log.Printf(
			"method=%s path=%s status=%d duration=%s size=%d ip=%s",
			r.Method,
			r.URL.Path,
			rw.statusCode,
			duration,
			rw.responseSize,
			r.RemoteAddr,
		)
	})
}
