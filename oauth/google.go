package oauth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// GoogleOAuthConfig holds the configuration for Google OAuth
type GoogleOAuthConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
	Scopes       []string
}

// GoogleUserInfo represents the user information returned by Google
type GoogleUserInfo struct {
	ID            string `json:"id"`
	Email         string `json:"email"`
	VerifiedEmail bool   `json:"verified_email"`
	Name          string `json:"name"`
	GivenName     string `json:"given_name"`
	FamilyName    string `json:"family_name"`
	Picture       string `json:"picture"`
	Locale        string `json:"locale"`
}

// NewGoogleOAuth creates a new Google OAuth2 config
func NewGoogleOAuth(config GoogleOAuthConfig) *oauth2.Config {
	// If no scopes are provided, use default ones
	if len(config.Scopes) == 0 {
		config.Scopes = []string{
			"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/userinfo.profile",
		}
	}

	return &oauth2.Config{
		ClientID:     config.ClientID,
		ClientSecret: config.ClientSecret,
		RedirectURL:  config.RedirectURL,
		Scopes:       config.Scopes,
		Endpoint:     google.Endpoint,
	}
}

// GenerateStateToken creates a random state token for CSRF protection
func GenerateStateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(b), nil
}

// GetGoogleLoginURL returns the URL to redirect the user to for Google login
func GetGoogleLoginURL(oauthConfig *oauth2.Config, state string) string {
	return oauthConfig.AuthCodeURL(state, oauth2.AccessTypeOnline)
}

// HandleGoogleCallback processes the callback from Google OAuth
func HandleGoogleCallback(ctx context.Context, oauthConfig *oauth2.Config, state, code string) (*oauth2.Token, error) {
	// Exchange the authorization code for a token
	token, err := oauthConfig.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("code exchange failed: %s", err.Error())
	}
	return token, nil
}

// GetGoogleUserInfo fetches the user info from Google API
func GetGoogleUserInfo(ctx context.Context, token *oauth2.Token, oauthConfig *oauth2.Config) (*GoogleUserInfo, error) {
	// Create an HTTP client with the token
	client := oauthConfig.Client(ctx, token)

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

	var userInfo GoogleUserInfo
	if err := json.Unmarshal(data, &userInfo); err != nil {
		return nil, fmt.Errorf("failed parsing user info: %s", err.Error())
	}

	return &userInfo, nil
}
