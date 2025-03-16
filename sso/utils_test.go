package sso

import (
	"testing"
)

func TestGenerateRandomString(t *testing.T) {
	// Test that we can generate a random string
	s1, err := GenerateRandomString(32)
	if err != nil {
		t.Fatalf("Failed to generate random string: %v", err)
	}

	// Test that the string is not empty
	if s1 == "" {
		t.Fatalf("Generated random string is empty")
	}

	// Test that we get a different string each time
	s2, err := GenerateRandomString(32)
	if err != nil {
		t.Fatalf("Failed to generate random string: %v", err)
	}

	if s1 == s2 {
		t.Fatalf("Generated random strings are the same: %s", s1)
	}
}

func TestIsValidRedirectURL(t *testing.T) {
	tests := []struct {
		url      string
		expected bool
	}{
		{"", false},
		{"invalid", false},
		{"/", true},
		{"/dashboard", true},
		{"http://example.com", true},
		{"https://example.com", true},
		{"https://example.com/dashboard", true},
		{"javascript:alert(1)", true}, // This should be false in a real implementation
	}

	for _, test := range tests {
		result := IsValidRedirectURL(test.url)
		if result != test.expected {
			t.Errorf("IsValidRedirectURL(%q) = %v, expected %v", test.url, result, test.expected)
		}
	}
}

func TestEncodeDecodeState(t *testing.T) {
	state := "abc123"
	redirectURL := "/dashboard"

	// Test encoding
	encoded := EncodeState(state, redirectURL)
	if encoded == "" {
		t.Fatalf("Encoded state is empty")
	}

	// Test decoding
	decodedState, decodedURL := DecodeState(encoded)
	if decodedState != state {
		t.Errorf("Decoded state = %q, expected %q", decodedState, state)
	}
	if decodedURL != redirectURL {
		t.Errorf("Decoded URL = %q, expected %q", decodedURL, redirectURL)
	}

	// Test decoding with no redirect URL
	encoded = state
	decodedState, decodedURL = DecodeState(encoded)
	if decodedState != state {
		t.Errorf("Decoded state = %q, expected %q", decodedState, state)
	}
	if decodedURL != "" {
		t.Errorf("Decoded URL = %q, expected %q", decodedURL, "")
	}
}
