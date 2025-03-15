package hmac

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	"testing"
)

func TestNewHMAC(t *testing.T) {
	tests := []struct {
		name      string
		key       []byte
		algorithm HashAlgorithm
		encoding  Encoding
		wantErr   bool
	}{
		{
			name:      "Valid configuration with SHA256 and HEX",
			key:       []byte("test-key"),
			algorithm: SHA256,
			encoding:  HEX,
			wantErr:   false,
		},
		{
			name:      "Valid configuration with SHA512 and BASE64",
			key:       []byte("test-key"),
			algorithm: SHA512,
			encoding:  BASE64,
			wantErr:   false,
		},
		{
			name:      "Empty key",
			key:       []byte{},
			algorithm: SHA256,
			encoding:  HEX,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, err := NewHMAC(tt.key, tt.algorithm, tt.encoding)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewHMAC() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && h == nil {
				t.Error("NewHMAC() returned nil HMAC with no error")
			}
		})
	}
}

func TestHMAC_Sign(t *testing.T) {
	key := []byte("test-key")
	message := []byte("test-message")

	// Manual calculation for SHA256/HEX
	macSha256 := hmac.New(sha256.New, key)
	macSha256.Write(message)
	expectedSha256Hex := hex.EncodeToString(macSha256.Sum(nil))

	// Manual calculation for SHA512/BASE64
	macSha512 := hmac.New(sha512.New, key)
	macSha512.Write(message)
	expectedSha512Base64 := base64.StdEncoding.EncodeToString(macSha512.Sum(nil))

	tests := []struct {
		name      string
		key       []byte
		message   []byte
		algorithm HashAlgorithm
		encoding  Encoding
		expected  string
		wantErr   bool
	}{
		{
			name:      "SHA256 with HEX encoding",
			key:       key,
			message:   message,
			algorithm: SHA256,
			encoding:  HEX,
			expected:  expectedSha256Hex,
			wantErr:   false,
		},
		{
			name:      "SHA512 with BASE64 encoding",
			key:       key,
			message:   message,
			algorithm: SHA512,
			encoding:  BASE64,
			expected:  expectedSha512Base64,
			wantErr:   false,
		},
		{
			name:      "Empty message",
			key:       key,
			message:   []byte{},
			algorithm: SHA256,
			encoding:  HEX,
			expected:  "",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, err := NewHMAC(tt.key, tt.algorithm, tt.encoding)
			if err != nil {
				t.Fatalf("Failed to create HMAC: %v", err)
			}

			got, err := h.Sign(tt.message)
			if (err != nil) != tt.wantErr {
				t.Errorf("HMAC.Sign() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.expected {
				t.Errorf("HMAC.Sign() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestHMAC_Verify(t *testing.T) {
	key := []byte("test-key")
	message := []byte("test-message")
	tamperedMessage := []byte("tampered-message")

	tests := []struct {
		name      string
		key       []byte
		message   []byte
		verifyMsg []byte
		algorithm HashAlgorithm
		encoding  Encoding
		wantErr   bool
	}{
		{
			name:      "Valid signature SHA256/HEX",
			key:       key,
			message:   message,
			verifyMsg: message,
			algorithm: SHA256,
			encoding:  HEX,
			wantErr:   false,
		},
		{
			name:      "Valid signature SHA512/BASE64",
			key:       key,
			message:   message,
			verifyMsg: message,
			algorithm: SHA512,
			encoding:  BASE64,
			wantErr:   false,
		},
		{
			name:      "Tampered message",
			key:       key,
			message:   message,
			verifyMsg: tamperedMessage,
			algorithm: SHA256,
			encoding:  HEX,
			wantErr:   true,
		},
		{
			name:      "Empty message",
			key:       key,
			message:   message,
			verifyMsg: []byte{},
			algorithm: SHA256,
			encoding:  HEX,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, err := NewHMAC(tt.key, tt.algorithm, tt.encoding)
			if err != nil {
				t.Fatalf("Failed to create HMAC: %v", err)
			}

			signature, err := h.Sign(tt.message)
			if err != nil {
				t.Fatalf("Failed to sign message: %v", err)
			}

			err = h.Verify(tt.verifyMsg, signature)
			if (err != nil) != tt.wantErr {
				t.Errorf("HMAC.Verify() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
