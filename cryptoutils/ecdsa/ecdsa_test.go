package ecdsa

import (
	"bytes"
	"crypto/elliptic"
	"encoding/asn1"
	"math/big"
	"os"
	"path/filepath"
	"testing"
)

func TestGenerateKeyPair(t *testing.T) {
	t.Parallel()

	keyPair, err := generateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair() error = %v", err)
	}

	if keyPair == nil {
		t.Fatal("GenerateKeyPair() returned nil keyPair")
	}

	if keyPair.PrivateKey == nil {
		t.Error("GenerateKeyPair() returned nil PrivateKey")
	}

	if keyPair.PublicKey == nil {
		t.Error("GenerateKeyPair() returned nil PublicKey")
	}

	// Verify the curve is P-256
	if keyPair.PrivateKey.Curve != elliptic.P256() {
		t.Error("GenerateKeyPair() did not use P-256 curve")
	}
}

func TestSignAndVerify(t *testing.T) {
	t.Parallel()

	// Define test cases
	testCases := []struct {
		name    string
		message []byte
	}{
		{
			name:    "Empty message",
			message: []byte(""),
		},
		{
			name:    "Short message",
			message: []byte("Hello"),
		},
		{
			name:    "Long message",
			message: bytes.Repeat([]byte("Test message for signing. "), 100),
		},
	}

	// Generate a key pair for testing
	keyPair, err := generateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate key pair: %v", err)
	}

	for _, tc := range testCases {
		tc := tc // Capture range variable for parallel execution
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Sign the message
			signature, err := sign(keyPair.PrivateKey, tc.message)
			if err != nil {
				t.Fatalf("Sign() error = %v", err)
			}

			// Verify the signature
			valid, err := verify(keyPair.PublicKey, tc.message, signature)
			if err != nil {
				t.Fatalf("Verify() error = %v", err)
			}

			if !valid {
				t.Errorf("Verify() = false, want true")
			}

			// Test with wrong message to ensure verification fails
			if len(tc.message) > 0 {
				wrongMessage := append([]byte{}, tc.message...)
				wrongMessage[0] = wrongMessage[0] ^ 0xFF // Flip bits in the first byte

				valid, err := verify(keyPair.PublicKey, wrongMessage, signature)
				if err != nil {
					t.Fatalf("Verify() with wrong message error = %v", err)
				}

				if valid {
					t.Errorf("Verify() with wrong message = true, want false")
				}
			}
		})
	}
}

func TestSignWithNilKey(t *testing.T) {
	t.Parallel()

	_, err := sign(nil, []byte("test message"))
	if err == nil {
		t.Error("Sign() with nil key did not return an error")
	}
}

func TestVerifyWithNilKey(t *testing.T) {
	t.Parallel()

	_, err := verify(nil, []byte("test message"), []byte("invalid signature"))
	if err == nil {
		t.Error("Verify() with nil key did not return an error")
	}
}

func TestVerifyWithInvalidSignature(t *testing.T) {
	t.Parallel()

	keyPair, err := generateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate key pair: %v", err)
	}

	_, err = verify(keyPair.PublicKey, []byte("test message"), []byte("invalid signature"))
	if err == nil {
		t.Error("Verify() with invalid signature did not return an error")
	}
}

func TestKeyPEMSerialization(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name         string
		createTempFn func() (string, error)
	}{
		{
			name: "Existing directory",
			createTempFn: func() (string, error) {
				return os.MkdirTemp("", "ecdsa-test-*")
			},
		},
	}

	for _, tc := range testCases {
		tc := tc // Capture range variable for parallel execution
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Create a temporary directory for the test
			tempDir, err := tc.createTempFn()
			if err != nil {
				t.Fatalf("Failed to create temporary directory: %v", err)
			}
			defer os.RemoveAll(tempDir)

			// Generate key pair
			keyPair, err := generateKeyPair()
			if err != nil {
				t.Fatalf("Failed to generate key pair: %v", err)
			}

			// Define file paths
			privateKeyPath := filepath.Join(tempDir, "private.pem")
			publicKeyPath := filepath.Join(tempDir, "public.pem")

			// Test saving and loading private key
			if err := savePrivateKeyToPEM(keyPair.PrivateKey, privateKeyPath); err != nil {
				t.Fatalf("SavePrivateKeyToPEM() error = %v", err)
			}

			loadedPrivateKey, err := loadPrivateKeyFromPEM(privateKeyPath)
			if err != nil {
				t.Fatalf("LoadPrivateKeyFromPEM() error = %v", err)
			}

			// Compare original and loaded private keys
			if loadedPrivateKey.D.Cmp(keyPair.PrivateKey.D) != 0 {
				t.Error("Loaded private key doesn't match the original")
			}

			// Test saving and loading public key
			if err := savePublicKeyToPEM(keyPair.PublicKey, publicKeyPath); err != nil {
				t.Fatalf("SavePublicKeyToPEM() error = %v", err)
			}

			loadedPublicKey, err := loadPublicKeyFromPEM(publicKeyPath)
			if err != nil {
				t.Fatalf("LoadPublicKeyFromPEM() error = %v", err)
			}

			// Compare original and loaded public keys
			if loadedPublicKey.X.Cmp(keyPair.PublicKey.X) != 0 || loadedPublicKey.Y.Cmp(keyPair.PublicKey.Y) != 0 {
				t.Error("Loaded public key doesn't match the original")
			}

			// Test that a signature created with the original key can be verified with the loaded key and vice versa
			message := []byte("test message for key serialization")
			signature, err := sign(keyPair.PrivateKey, message)
			if err != nil {
				t.Fatalf("Sign() error = %v", err)
			}

			valid, err := verify(loadedPublicKey, message, signature)
			if err != nil {
				t.Fatalf("Verify() with loaded public key error = %v", err)
			}
			if !valid {
				t.Error("Signature verified with loaded public key = false, want true")
			}
		})
	}
}

func TestPEMErrorCases(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		testFunc func(*testing.T)
	}{
		{
			name: "SavePrivateKeyToPEM with nil key",
			testFunc: func(t *testing.T) {
				err := savePrivateKeyToPEM(nil, "nonexistent.pem")
				if err == nil {
					t.Error("SavePrivateKeyToPEM() with nil key did not return an error")
				}
			},
		},
		{
			name: "SavePublicKeyToPEM with nil key",
			testFunc: func(t *testing.T) {
				err := savePublicKeyToPEM(nil, "nonexistent.pem")
				if err == nil {
					t.Error("SavePublicKeyToPEM() with nil key did not return an error")
				}
			},
		},
		{
			name: "LoadPrivateKeyFromPEM with nonexistent file",
			testFunc: func(t *testing.T) {
				_, err := loadPrivateKeyFromPEM("nonexistent.pem")
				if err == nil {
					t.Error("LoadPrivateKeyFromPEM() with nonexistent file did not return an error")
				}
			},
		},
		{
			name: "LoadPublicKeyFromPEM with nonexistent file",
			testFunc: func(t *testing.T) {
				_, err := loadPublicKeyFromPEM("nonexistent.pem")
				if err == nil {
					t.Error("LoadPublicKeyFromPEM() with nonexistent file did not return an error")
				}
			},
		},
	}

	for _, tc := range testCases {
		tc := tc // Capture range variable for parallel execution
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tc.testFunc(t)
		})
	}
}

func TestBaseEncoding(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		data    []byte
		corrupt bool
	}{
		{
			name:    "Empty data",
			data:    []byte{},
			corrupt: false,
		},
		{
			name:    "Small data",
			data:    []byte{1, 2, 3, 4, 5},
			corrupt: false,
		},
		{
			name:    "Large data",
			data:    bytes.Repeat([]byte{1, 2, 3, 4, 5}, 100),
			corrupt: false,
		},
		{
			name:    "Invalid base64",
			data:    []byte{1, 2, 3},
			corrupt: true,
		},
	}

	for _, tc := range testCases {
		tc := tc // Capture range variable for parallel execution
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var encodedData string
			if tc.corrupt {
				encodedData = "ThisIsNotValidBase64!@#$"
			} else {
				encodedData = encodeSignatureBase64(tc.data)
			}

			decodedData, err := decodeSignatureBase64(encodedData)
			if tc.corrupt {
				if err == nil {
					t.Error("DecodeSignatureBase64() with invalid base64 did not return an error")
				}
			} else {
				if err != nil {
					t.Fatalf("DecodeSignatureBase64() error = %v", err)
				}

				if !bytes.Equal(decodedData, tc.data) {
					t.Errorf("Decoded data doesn't match original: got %v, want %v", decodedData, tc.data)
				}
			}
		})
	}
}

func TestSignatureRoundtrip(t *testing.T) {
	t.Parallel()

	// Generate key pair and create a signature
	keyPair, err := generateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate key pair: %v", err)
	}

	message := []byte("Test message for signature roundtrip")
	signature, err := sign(keyPair.PrivateKey, message)
	if err != nil {
		t.Fatalf("Sign() error = %v", err)
	}

	// Encode and decode the signature
	encodedSignature := encodeSignatureBase64(signature)
	decodedSignature, err := decodeSignatureBase64(encodedSignature)
	if err != nil {
		t.Fatalf("DecodeSignatureBase64() error = %v", err)
	}

	// Verify the decoded signature
	valid, err := verify(keyPair.PublicKey, message, decodedSignature)
	if err != nil {
		t.Fatalf("Verify() error = %v", err)
	}

	if !valid {
		t.Error("Verify() with decoded signature = false, want true")
	}
}

func TestECDSASignatureStruct(t *testing.T) {
	t.Parallel()

	// Create a signature with known R and S values
	r := big.NewInt(12345)
	s := big.NewInt(67890)
	signature := ECDSASignature{R: r, S: s}

	// Marshal and unmarshal the signature
	marshaled, err := asn1.Marshal(signature)
	if err != nil {
		t.Fatalf("Failed to marshal signature: %v", err)
	}

	var unmarshaled ECDSASignature
	_, err = asn1.Unmarshal(marshaled, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal signature: %v", err)
	}

	// Compare R and S values
	if unmarshaled.R.Cmp(r) != 0 || unmarshaled.S.Cmp(s) != 0 {
		t.Errorf("Unmarshaled signature doesn't match original: got R=%v, S=%v, want R=%v, S=%v",
			unmarshaled.R, unmarshaled.S, r, s)
	}
}

// TestKeyPairGeneration verifies key generation across multiple iterations
func TestKeyPairGenerationMultiple(t *testing.T) {
	t.Parallel()

	// Generate multiple key pairs to ensure uniqueness
	iterations := 5
	keyPairs := make([]*KeyPair, iterations)

	for i := 0; i < iterations; i++ {
		keyPair, err := generateKeyPair()
		if err != nil {
			t.Fatalf("GenerateKeyPair() iteration %d error = %v", i, err)
		}
		keyPairs[i] = keyPair
	}

	// Verify that each key pair is unique
	for i := 0; i < iterations; i++ {
		for j := i + 1; j < iterations; j++ {
			if keyPairs[i].PrivateKey.D.Cmp(keyPairs[j].PrivateKey.D) == 0 {
				t.Errorf("Key pairs %d and %d have the same private key", i, j)
			}
		}
	}
}
