package http

import (
	"fmt"
	"io"
	"net/http"
)

// ForwardProxyMiddleware handles the forwarding of requests to the target
func ForwardProxyMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract the full target URL
		targetURL := fmt.Sprintf("%s://%s%s", "http", r.Host, r.URL.Path)

		// Create a new request for the target server with the same method, URL, and headers
		req, err := http.NewRequest(r.Method, targetURL+r.RequestURI, r.Body)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to create request: %v", err), http.StatusInternalServerError)
			return
		}

		// Copy headers from the original request to the new request
		req.Header = r.Header

		// Send the new request to the target server
		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to forward request: %v", err), http.StatusBadGateway)
			return
		}
		defer resp.Body.Close()

		// Copy the response headers from the target server to the original response
		for key, value := range resp.Header {
			w.Header()[key] = value
		}

		// Set the status code
		w.WriteHeader(resp.StatusCode)

		// Read the response body from the target server and write it to the original response
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to read response body: %v", err), http.StatusInternalServerError)
			return
		}

		// Write the body to the response writer
		w.Write(body)
	})
}
