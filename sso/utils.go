package sso

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/url"
)

// GenerateRandomBytes returns securely generated random bytes
func GenerateRandomBytes(n int) ([]byte, error) {
	b := make([]byte, n)
	_, err := rand.Read(b)
	if err != nil {
		return nil, err
	}
	return b, nil
}

// GenerateRandomString returns a URL-safe, base64 encoded
// securely generated random string
func GenerateRandomString(s int) (string, error) {
	b, err := GenerateRandomBytes(s)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// IsValidRedirectURL checks if a redirect URL is valid
// This is a simple implementation that checks if the URL is well-formed
// In production, you should also check if the URL is in a whitelist
func IsValidRedirectURL(redirectURL string) bool {
	if redirectURL == "" {
		return false
	}

	u, err := url.Parse(redirectURL)
	if err != nil {
		return false
	}

	// Check if the URL is absolute
	if !u.IsAbs() {
		// For relative URLs, we can allow them if they start with /
		return len(redirectURL) > 0 && redirectURL[0] == '/'
	}

	// For absolute URLs, we should check if they are in a whitelist
	// This is a simple implementation that allows all absolute URLs
	// In production, you should check if the URL is in a whitelist
	return true
}

// EncodeState encodes a state token with a redirect URL
func EncodeState(state, redirectURL string) string {
	return fmt.Sprintf("%s:%s", state, redirectURL)
}

// DecodeState decodes a state token with a redirect URL
func DecodeState(encodedState string) (state, redirectURL string) {
	for i, c := range encodedState {
		if c == ':' {
			return encodedState[:i], encodedState[i+1:]
		}
	}
	return encodedState, ""
}
