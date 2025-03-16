package webauthn

import (
	"encoding/json"
	"net/http"
)

// Handlers contains HTTP handlers for WebAuthn
type Handlers struct {
	service *Service
}

// NewHandlers creates new WebAuthn handlers
func NewHandlers(service *Service) *Handlers {
	return &Handlers{
		service: service,
	}
}

// BeginRegistrationHandler handles the begin registration request
func (h *Handlers) BeginRegistrationHandler(w http.ResponseWriter, r *http.Request) {
	// Only allow POST requests
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse request
	var req struct {
		Username    string `json:"username"`
		DisplayName string `json:"displayName"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Begin registration
	options, _, err := h.service.BeginRegistration(req.Username, req.DisplayName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Return options
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(options)
}

// FinishRegistrationHandler handles the finish registration request
func (h *Handlers) FinishRegistrationHandler(w http.ResponseWriter, r *http.Request) {
	// Only allow POST requests
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get username from query parameter
	username := r.URL.Query().Get("username")
	if username == "" {
		http.Error(w, "Username is required", http.StatusBadRequest)
		return
	}

	// Finish registration
	if err := h.service.FinishRegistration(username, r); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Return success
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ok"}`))
}

// BeginLoginHandler handles the begin login request
func (h *Handlers) BeginLoginHandler(w http.ResponseWriter, r *http.Request) {
	// Only allow POST requests
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse request
	var req struct {
		Username string `json:"username"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Begin login
	options, err := h.service.BeginLogin(req.Username)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Return options
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(options)
}

// FinishLoginHandler handles the finish login request
func (h *Handlers) FinishLoginHandler(w http.ResponseWriter, r *http.Request) {
	// Only allow POST requests
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get username from query parameter
	username := r.URL.Query().Get("username")
	if username == "" {
		http.Error(w, "Username is required", http.StatusBadRequest)
		return
	}

	// Finish login
	if err := h.service.FinishLogin(username, r); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Return success
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ok"}`))
}

// RegisterHandlers registers the WebAuthn handlers
func (h *Handlers) RegisterHandlers(mux *http.ServeMux) {
	mux.HandleFunc("/webauthn/register/begin", h.BeginRegistrationHandler)
	mux.HandleFunc("/webauthn/register/finish", h.FinishRegistrationHandler)
	mux.HandleFunc("/webauthn/login/begin", h.BeginLoginHandler)
	mux.HandleFunc("/webauthn/login/finish", h.FinishLoginHandler)
}
