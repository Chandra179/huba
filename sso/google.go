package sso

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// GoogleProvider implements the Provider interface for Google SSO
type GoogleProvider struct {
	config *oauth2.Config
}

// NewGoogleProvider creates a new Google SSO provider
func NewGoogleProvider(config ProviderConfig) *GoogleProvider {
	// Set default scopes if none provided
	if len(config.Scopes) == 0 {
		config.Scopes = []string{
			"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/userinfo.profile",
		}
	}

	oauthConfig := &oauth2.Config{
		ClientID:     config.ClientID,
		ClientSecret: config.ClientSecret,
		RedirectURL:  config.RedirectURL,
		Scopes:       config.Scopes,
		Endpoint:     google.Endpoint,
	}

	return &GoogleProvider{
		config: oauthConfig,
	}
}

// Name returns the name of the provider
func (p *GoogleProvider) Name() string {
	return "google"
}

// GetAuthURL returns the URL to redirect the user to for authentication
func (p *GoogleProvider) GetAuthURL(state string) string {
	return p.config.AuthCodeURL(state, oauth2.AccessTypeOnline)
}

// HandleCallback processes the callback from Google
func (p *GoogleProvider) HandleCallback(ctx context.Context, r *http.Request) (*UserProfile, error) {
	// Get the authorization code from the request
	code := r.URL.Query().Get("code")
	if code == "" {
		return nil, fmt.Errorf("no code in request")
	}

	// Exchange the authorization code for a token
	token, err := p.config.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("code exchange failed: %s", err.Error())
	}

	// Create an HTTP client with the token
	client := p.config.Client(ctx, token)

	// Fetch user info from Google API
	resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		return nil, fmt.Errorf("failed getting user info: %s", err.Error())
	}
	defer resp.Body.Close()

	// Read and parse the response
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed reading response body: %s", err.Error())
	}

	// Parse the user info
	var userInfo map[string]interface{}
	if err := json.Unmarshal(data, &userInfo); err != nil {
		return nil, fmt.Errorf("failed parsing user info: %s", err.Error())
	}

	// Create a standardized user profile
	profile := &UserProfile{
		Provider:      p.Name(),
		ID:            getStringValue(userInfo, "id"),
		Email:         getStringValue(userInfo, "email"),
		EmailVerified: getBoolValue(userInfo, "verified_email"),
		Name:          getStringValue(userInfo, "name"),
		FirstName:     getStringValue(userInfo, "given_name"),
		LastName:      getStringValue(userInfo, "family_name"),
		Picture:       getStringValue(userInfo, "picture"),
		Locale:        getStringValue(userInfo, "locale"),
		RawData:       userInfo,
	}

	return profile, nil
}

// Helper functions to safely extract values from the map
func getStringValue(data map[string]interface{}, key string) string {
	if val, ok := data[key]; ok {
		if strVal, ok := val.(string); ok {
			return strVal
		}
	}
	return ""
}

func getBoolValue(data map[string]interface{}, key string) bool {
	if val, ok := data[key]; ok {
		if boolVal, ok := val.(bool); ok {
			return boolVal
		}
	}
	return false
}
