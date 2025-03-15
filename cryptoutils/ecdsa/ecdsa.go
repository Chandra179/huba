package ecdsa

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/asn1"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"os"
)

// ECDSAService defines the interface for ECDSA cryptographic operations
type ECDSAService interface {
	// GenerateKeyPair creates a new ECDSA key pair
	GenerateKeyPair() (*KeyPair, error)

	// Sign creates an ECDSA signature for the provided message using the private key
	Sign(privateKey *ecdsa.PrivateKey, message []byte) ([]byte, error)

	// Verify verifies an ECDSA signature against a message using the public key
	Verify(publicKey *ecdsa.PublicKey, message, signature []byte) (bool, error)

	// SavePrivateKeyToPEM saves the private key to a PEM file
	SavePrivateKeyToPEM(privateKey *ecdsa.PrivateKey, filename string) error

	// LoadPrivateKeyFromPEM loads a private key from a PEM file
	LoadPrivateKeyFromPEM(filename string) (*ecdsa.PrivateKey, error)

	// SavePublicKeyToPEM saves the public key to a PEM file
	SavePublicKeyToPEM(publicKey *ecdsa.PublicKey, filename string) error

	// LoadPublicKeyFromPEM loads a public key from a PEM file
	LoadPublicKeyFromPEM(filename string) (*ecdsa.PublicKey, error)

	// EncodeSignatureBase64 encodes a signature as a Base64 string
	EncodeSignatureBase64(signature []byte) string

	// DecodeSignatureBase64 decodes a Base64-encoded signature
	DecodeSignatureBase64(encodedSignature string) ([]byte, error)
}

// DefaultECDSAService is the default implementation of ECDSAService
type DefaultECDSAService struct{}

// NewECDSAService creates a new instance of the default ECDSA service
func NewECDSAService() ECDSAService {
	return &DefaultECDSAService{}
}

// GenerateKeyPair implements ECDSAService.GenerateKeyPair
func (s *DefaultECDSAService) GenerateKeyPair() (*KeyPair, error) {
	return generateKeyPair()
}

// Sign implements ECDSAService.Sign
func (s *DefaultECDSAService) Sign(privateKey *ecdsa.PrivateKey, message []byte) ([]byte, error) {
	return sign(privateKey, message)
}

// Verify implements ECDSAService.Verify
func (s *DefaultECDSAService) Verify(publicKey *ecdsa.PublicKey, message, signature []byte) (bool, error) {
	return verify(publicKey, message, signature)
}

// SavePrivateKeyToPEM implements ECDSAService.SavePrivateKeyToPEM
func (s *DefaultECDSAService) SavePrivateKeyToPEM(privateKey *ecdsa.PrivateKey, filename string) error {
	return savePrivateKeyToPEM(privateKey, filename)
}

// LoadPrivateKeyFromPEM implements ECDSAService.LoadPrivateKeyFromPEM
func (s *DefaultECDSAService) LoadPrivateKeyFromPEM(filename string) (*ecdsa.PrivateKey, error) {
	return loadPrivateKeyFromPEM(filename)
}

// SavePublicKeyToPEM implements ECDSAService.SavePublicKeyToPEM
func (s *DefaultECDSAService) SavePublicKeyToPEM(publicKey *ecdsa.PublicKey, filename string) error {
	return savePublicKeyToPEM(publicKey, filename)
}

// LoadPublicKeyFromPEM implements ECDSAService.LoadPublicKeyFromPEM
func (s *DefaultECDSAService) LoadPublicKeyFromPEM(filename string) (*ecdsa.PublicKey, error) {
	return loadPublicKeyFromPEM(filename)
}

// EncodeSignatureBase64 implements ECDSAService.EncodeSignatureBase64
func (s *DefaultECDSAService) EncodeSignatureBase64(signature []byte) string {
	return encodeSignatureBase64(signature)
}

// DecodeSignatureBase64 implements ECDSAService.DecodeSignatureBase64
func (s *DefaultECDSAService) DecodeSignatureBase64(encodedSignature string) ([]byte, error) {
	return decodeSignatureBase64(encodedSignature)
}

// ECDSASignature represents the R and S components of an ECDSA signature
type ECDSASignature struct {
	R, S *big.Int
}

// KeyPair contains both private and public keys
type KeyPair struct {
	PrivateKey *ecdsa.PrivateKey
	PublicKey  *ecdsa.PublicKey
}

// generateKeyPair creates a new ECDSA key pair using the P-256 curve
func generateKeyPair() (*KeyPair, error) {
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate ECDSA key pair: %w", err)
	}

	return &KeyPair{
		PrivateKey: privateKey,
		PublicKey:  &privateKey.PublicKey,
	}, nil
}

// sign creates an ECDSA signature for the provided message using the private key
func sign(privateKey *ecdsa.PrivateKey, message []byte) ([]byte, error) {
	if privateKey == nil {
		return nil, errors.New("private key cannot be nil")
	}

	// Create a SHA-256 hash of the message
	hash := sha256.Sum256(message)

	// Sign the hash with the private key
	r, s, err := ecdsa.Sign(rand.Reader, privateKey, hash[:])
	if err != nil {
		return nil, fmt.Errorf("failed to sign message: %w", err)
	}

	// Create an ASN.1 sequence containing the R and S values
	signature, err := asn1.Marshal(ECDSASignature{r, s})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal signature: %w", err)
	}

	return signature, nil
}

// verify verifies an ECDSA signature against a message using the public key
func verify(publicKey *ecdsa.PublicKey, message, signature []byte) (bool, error) {
	if publicKey == nil {
		return false, errors.New("public key cannot be nil")
	}

	// Parse the ASN.1 encoded signature
	var ecdsaSignature ECDSASignature
	if _, err := asn1.Unmarshal(signature, &ecdsaSignature); err != nil {
		return false, fmt.Errorf("failed to unmarshal signature: %w", err)
	}

	// Create a SHA-256 hash of the message
	hash := sha256.Sum256(message)

	// Verify the signature
	return ecdsa.Verify(publicKey, hash[:], ecdsaSignature.R, ecdsaSignature.S), nil
}

// savePrivateKeyToPEM saves the private key to a PEM file
func savePrivateKeyToPEM(privateKey *ecdsa.PrivateKey, filename string) error {
	if privateKey == nil {
		return errors.New("private key cannot be nil")
	}

	// Marshal the private key to PKCS8 format
	privateKeyBytes, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		return fmt.Errorf("failed to marshal private key: %w", err)
	}

	// Create a PEM block with the private key data
	privateKeyPEM := &pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: privateKeyBytes,
	}

	// Create the file with appropriate permissions
	file, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("failed to create private key file: %w", err)
	}
	defer file.Close()

	// Write the PEM block to the file
	if err := pem.Encode(file, privateKeyPEM); err != nil {
		return fmt.Errorf("failed to write private key to file: %w", err)
	}

	return nil
}

// loadPrivateKeyFromPEM loads a private key from a PEM file
func loadPrivateKeyFromPEM(filename string) (*ecdsa.PrivateKey, error) {
	// Read the file
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read private key file: %w", err)
	}

	// Decode the PEM block
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, errors.New("failed to decode PEM block")
	}

	// Parse the private key
	privateKey, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	// Ensure the key is an ECDSA key
	ecdsaKey, ok := privateKey.(*ecdsa.PrivateKey)
	if !ok {
		return nil, errors.New("private key is not an ECDSA key")
	}

	return ecdsaKey, nil
}

// savePublicKeyToPEM saves the public key to a PEM file
func savePublicKeyToPEM(publicKey *ecdsa.PublicKey, filename string) error {
	if publicKey == nil {
		return errors.New("public key cannot be nil")
	}

	// Marshal the public key
	publicKeyBytes, err := x509.MarshalPKIXPublicKey(publicKey)
	if err != nil {
		return fmt.Errorf("failed to marshal public key: %w", err)
	}

	// Create a PEM block with the public key data
	publicKeyPEM := &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: publicKeyBytes,
	}

	// Create the file with appropriate permissions
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create public key file: %w", err)
	}
	defer file.Close()

	// Write the PEM block to the file
	if err := pem.Encode(file, publicKeyPEM); err != nil {
		return fmt.Errorf("failed to write public key to file: %w", err)
	}

	return nil
}

// loadPublicKeyFromPEM loads a public key from a PEM file
func loadPublicKeyFromPEM(filename string) (*ecdsa.PublicKey, error) {
	// Read the file
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read public key file: %w", err)
	}

	// Decode the PEM block
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, errors.New("failed to decode PEM block")
	}

	// Parse the public key
	publicKey, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse public key: %w", err)
	}

	// Ensure the key is an ECDSA key
	ecdsaKey, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		return nil, errors.New("public key is not an ECDSA key")
	}

	return ecdsaKey, nil
}

// encodeSignatureBase64 encodes a signature as a Base64 string
func encodeSignatureBase64(signature []byte) string {
	return base64.StdEncoding.EncodeToString(signature)
}

// decodeSignatureBase64 decodes a Base64-encoded signature
func decodeSignatureBase64(encodedSignature string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(encodedSignature)
}
