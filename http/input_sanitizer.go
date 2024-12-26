package http

import (
	"html"
	"net/http"
	"strings"
)

func InputSanitizerMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Sanitize query parameters
			query := r.URL.Query()
			for key, values := range query {
				for i, value := range values {
					// Escape any potentially dangerous characters
					query[key][i] = html.EscapeString(value)
				}
			}
			r.URL.RawQuery = query.Encode()

			// Sanitize form data for POST/PUT requests
			if r.Method == http.MethodPost || r.Method == http.MethodPut {
				if strings.Contains(r.Header.Get("Content-Type"), "application/x-www-form-urlencoded") {
					if err := r.ParseForm(); err == nil {
						for key, values := range r.Form {
							for i, value := range values {
								// Escape any potentially dangerous characters
								r.Form[key][i] = html.EscapeString(value)
							}
						}
					}
				}
			}

			// Pass the sanitized request to the next handler
			next.ServeHTTP(w, r)
		})
	}
}
