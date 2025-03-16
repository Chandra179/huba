package keycloak

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"golang.org/x/oauth2"
)

// KeycloakConfig holds the configuration for Keycloak OAuth
type KeycloakConfig struct {
	BaseURL      string
	Realm        string
	ClientID     string
	ClientSecret string
	RedirectURL  string
	Scopes       []string
}

// UserInfo represents the user information returned by Keycloak
type UserInfo struct {
	ID                string `json:"sub"`
	Email             string `json:"email"`
	EmailVerified     bool   `json:"email_verified"`
	PreferredUsername string `json:"preferred_username"`
	Name              string `json:"name"`
	GivenName         string `json:"given_name"`
	FamilyName        string `json:"family_name"`
	RealmAccess       struct {
		Roles []string `json:"roles"`
	} `json:"realm_access"`
	ResourceAccess map[string]struct {
		Roles []string `json:"roles"`
	} `json:"resource_access"`
}

// NewKeycloakOAuth creates a new Keycloak OAuth2 config
func NewKeycloakOAuth(config KeycloakConfig) *oauth2.Config {
	// If no scopes are provided, use default ones
	if len(config.Scopes) == 0 {
		config.Scopes = []string{"openid", "profile", "email"}
	}

	// Construct Keycloak endpoints
	authURL := fmt.Sprintf("%s/realms/%s/protocol/openid-connect/auth", config.BaseURL, config.Realm)
	tokenURL := fmt.Sprintf("%s/realms/%s/protocol/openid-connect/token", config.BaseURL, config.Realm)

	return &oauth2.Config{
		ClientID:     config.ClientID,
		ClientSecret: config.ClientSecret,
		RedirectURL:  config.RedirectURL,
		Scopes:       config.Scopes,
		Endpoint: oauth2.Endpoint{
			AuthURL:  authURL,
			TokenURL: tokenURL,
		},
	}
}

// GetKeycloakLoginURL returns the URL to redirect the user to for Keycloak login
func GetKeycloakLoginURL(oauthConfig *oauth2.Config, state string) string {
	return oauthConfig.AuthCodeURL(state, oauth2.AccessTypeOnline)
}

// HandleKeycloakCallback processes the callback from Keycloak OAuth
func HandleKeycloakCallback(ctx context.Context, oauthConfig *oauth2.Config, state, code string) (*oauth2.Token, error) {
	// Exchange the authorization code for a token
	token, err := oauthConfig.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("code exchange failed: %s", err.Error())
	}
	return token, nil
}

// GetKeycloakUserInfo fetches the user info from Keycloak API
func GetKeycloakUserInfo(ctx context.Context, token *oauth2.Token, config KeycloakConfig) (*UserInfo, error) {
	// Create an HTTP client with the token
	client := &http.Client{}

	// Construct the userinfo endpoint URL
	userinfoURL := fmt.Sprintf("%s/realms/%s/protocol/openid-connect/userinfo", config.BaseURL, config.Realm)

	// Create a new request
	req, err := http.NewRequestWithContext(ctx, "GET", userinfoURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %s", err.Error())
	}

	// Add the authorization header
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", token.AccessToken))

	// Send the request
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed getting user info: %s", err.Error())
	}
	defer resp.Body.Close()

	// Check if the response is successful
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get user info: status=%d, body=%s", resp.StatusCode, string(body))
	}

	// Read and parse the response
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed reading response body: %s", err.Error())
	}

	var userInfo UserInfo
	if err := json.Unmarshal(data, &userInfo); err != nil {
		return nil, fmt.Errorf("failed parsing user info: %s", err.Error())
	}

	return &userInfo, nil
}

// GetUserRoles returns the roles assigned to the user
func GetUserRoles(userInfo *UserInfo) []string {
	var roles []string
	// Add realm roles
	if userInfo.RealmAccess.Roles != nil {
		roles = append(roles, userInfo.RealmAccess.Roles...)
	}

	// Add client roles
	for _, clientRoles := range userInfo.ResourceAccess {
		if clientRoles.Roles != nil {
			roles = append(roles, clientRoles.Roles...)
		}
	}

	return roles
}

// ValidateToken validates the token with Keycloak
func ValidateToken(ctx context.Context, token string, config KeycloakConfig) (bool, error) {
	// Construct the introspect endpoint URL
	introspectURL := fmt.Sprintf("%s/realms/%s/protocol/openid-connect/token/introspect", config.BaseURL, config.Realm)

	// Create form data
	data := url.Values{}
	data.Set("token", token)
	data.Set("client_id", config.ClientID)
	data.Set("client_secret", config.ClientSecret)

	// Create a new request
	req, err := http.NewRequestWithContext(ctx, "POST", introspectURL, strings.NewReader(data.Encode()))
	if err != nil {
		return false, fmt.Errorf("failed to create request: %s", err.Error())
	}

	// Set content type
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	// Send the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return false, fmt.Errorf("failed to validate token: %s", err.Error())
	}
	defer resp.Body.Close()

	// Read and parse the response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, fmt.Errorf("failed reading response body: %s", err.Error())
	}

	var result struct {
		Active bool `json:"active"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return false, fmt.Errorf("failed parsing response: %s", err.Error())
	}

	return result.Active, nil
}
