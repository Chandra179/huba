package sso

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"golang.org/x/oauth2"
)

// GitHubProvider implements the Provider interface for GitHub SSO
type GitHubProvider struct {
	config *oauth2.Config
}

// NewGitHubProvider creates a new GitHub SSO provider
func NewGitHubProvider(config ProviderConfig) *GitHubProvider {
	// Set default scopes if none provided
	if len(config.Scopes) == 0 {
		config.Scopes = []string{"user:email", "read:user"}
	}

	// Define GitHub OAuth2 endpoints
	githubEndpoint := oauth2.Endpoint{
		AuthURL:  "https://github.com/login/oauth/authorize",
		TokenURL: "https://github.com/login/oauth/access_token",
	}

	oauthConfig := &oauth2.Config{
		ClientID:     config.ClientID,
		ClientSecret: config.ClientSecret,
		RedirectURL:  config.RedirectURL,
		Scopes:       config.Scopes,
		Endpoint:     githubEndpoint,
	}

	return &GitHubProvider{
		config: oauthConfig,
	}
}

// Name returns the name of the provider
func (p *GitHubProvider) Name() string {
	return "github"
}

// GetAuthURL returns the URL to redirect the user to for authentication
func (p *GitHubProvider) GetAuthURL(state string) string {
	return p.config.AuthCodeURL(state, oauth2.AccessTypeOnline)
}

// HandleCallback processes the callback from GitHub
func (p *GitHubProvider) HandleCallback(ctx context.Context, r *http.Request) (*UserProfile, error) {
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

	// Fetch user info from GitHub API
	resp, err := client.Get("https://api.github.com/user")
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

	// Get email from GitHub API (since it might be private)
	email, emailVerified := p.getEmail(ctx, client)

	// Create a standardized user profile
	profile := &UserProfile{
		Provider:      p.Name(),
		ID:            getStringValue(userInfo, "id"),
		Email:         email,
		EmailVerified: emailVerified,
		Name:          getStringValue(userInfo, "name"),
		// GitHub doesn't provide first/last name separately
		Picture: getStringValue(userInfo, "avatar_url"),
		Locale:  "", // GitHub doesn't provide locale
		RawData: userInfo,
	}

	return profile, nil
}

// getEmail fetches the user's email from GitHub API
func (p *GitHubProvider) getEmail(ctx context.Context, client *http.Client) (string, bool) {
	resp, err := client.Get("https://api.github.com/user/emails")
	if err != nil {
		return "", false
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", false
	}

	var emails []map[string]interface{}
	if err := json.Unmarshal(data, &emails); err != nil {
		return "", false
	}

	// Find the primary email
	for _, email := range emails {
		isPrimary, ok := email["primary"].(bool)
		if ok && isPrimary {
			emailStr, _ := email["email"].(string)
			verified, _ := email["verified"].(bool)
			return emailStr, verified
		}
	}

	// If no primary email found, return the first one
	if len(emails) > 0 {
		emailStr, _ := emails[0]["email"].(string)
		verified, _ := emails[0]["verified"].(bool)
		return emailStr, verified
	}

	return "", false
}
