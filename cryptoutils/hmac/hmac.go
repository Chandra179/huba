package hmac

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/sha512"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"hash"
)

// Common errors returned by the package
var (
	ErrInvalidKey       = errors.New("hmac: key cannot be empty")
	ErrInvalidMessage   = errors.New("hmac: message cannot be empty")
	ErrInvalidSignature = errors.New("hmac: invalid signature")
)

// HashAlgorithm represents supported hash algorithms
type HashAlgorithm int

const (
	// SHA256 algorithm
	SHA256 HashAlgorithm = iota
	// SHA512 algorithm
	SHA512
)

// Encoding represents supported encoding formats
type Encoding int

const (
	// HEX encoding
	HEX Encoding = iota
	// BASE64 encoding
	BASE64
)

// HMACer defines the interface for HMAC operations
type HMACer interface {
	// Sign creates an HMAC signature for the given message
	Sign(message []byte) (string, error)

	// Verify checks if the provided signature matches the expected HMAC for the message
	Verify(message []byte, providedSignature string) error
}

// HMAC implements the HMACer interface
type HMAC struct {
	key       []byte
	algorithm HashAlgorithm
	encoding  Encoding
}

// NewHMAC creates a new HMAC utility with the specified configuration
func NewHMAC(key []byte, algorithm HashAlgorithm, encoding Encoding) (HMACer, error) {
	if len(key) == 0 {
		return nil, ErrInvalidKey
	}

	return &HMAC{
		key:       key,
		algorithm: algorithm,
		encoding:  encoding,
	}, nil
}

// getHashFunc returns the appropriate hash function based on the algorithm
func (h *HMAC) getHashFunc() func() hash.Hash {
	switch h.algorithm {
	case SHA512:
		return sha512.New
	default:
		return sha256.New
	}
}

// Sign creates an HMAC signature for the given message
func (h *HMAC) Sign(message []byte) (string, error) {
	if len(message) == 0 {
		return "", ErrInvalidMessage
	}

	mac := hmac.New(h.getHashFunc(), h.key)
	mac.Write(message)
	signature := mac.Sum(nil)

	return h.encode(signature), nil
}

// Verify checks if the provided signature matches the expected HMAC for the message
func (h *HMAC) Verify(message []byte, providedSignature string) error {
	if len(message) == 0 {
		return ErrInvalidMessage
	}

	if providedSignature == "" {
		return ErrInvalidSignature
	}

	// Generate the expected signature
	expectedSignature, err := h.Sign(message)
	if err != nil {
		return err
	}

	// Use a constant-time comparison to prevent timing attacks
	if subtle.ConstantTimeCompare([]byte(expectedSignature), []byte(providedSignature)) != 1 {
		return ErrInvalidSignature
	}

	return nil
}

// encode converts the byte signature to the configured encoding format
func (h *HMAC) encode(signature []byte) string {
	switch h.encoding {
	case BASE64:
		return base64.StdEncoding.EncodeToString(signature)
	default:
		return hex.EncodeToString(signature)
	}
}

// decode converts the string signature from the configured encoding to bytes
func (h *HMAC) decode(signature string) ([]byte, error) {
	switch h.encoding {
	case BASE64:
		return base64.StdEncoding.DecodeString(signature)
	default:
		return hex.DecodeString(signature)
	}
}
