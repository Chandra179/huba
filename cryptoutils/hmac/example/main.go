package main

import (
	"fmt"
	"log"
	"os"
	hmacutils "securedesign/cryptoutils/hmac"
)

func main() {
	// Example 1: Basic usage with SHA256 and hex encoding
	secretKey := []byte(os.Getenv("API_SECRET_KEY"))
	if len(secretKey) == 0 {
		secretKey = []byte("your-secure-key") // Default for demo purposes only
	}

	// Create a new HMAC utility using SHA256 and hex encoding
	hmacUtil, err := hmacutils.NewHMAC(secretKey, hmacutils.SHA256, hmacutils.HEX)
	if err != nil {
		log.Fatalf("Failed to create HMAC utility: %v", err)
	}

	// Message to authenticate
	message := []byte("This is a sensitive message that needs verification")

	// Generate signature
	signature, err := hmacUtil.Sign(message)
	if err != nil {
		log.Fatalf("Failed to sign message: %v", err)
	}
	fmt.Printf("Generated signature (SHA256, HEX): %s\n", signature)

	// Verify the signature
	err = hmacUtil.Verify(message, signature)
	if err != nil {
		log.Fatalf("Signature verification failed: %v", err)
	}
	fmt.Println("Signature verified successfully!")

	// Example 2: Using SHA512 and BASE64 encoding for higher security
	hmacUtil512, err := hmacutils.NewHMAC(secretKey, hmacutils.SHA512, hmacutils.BASE64)
	if err != nil {
		log.Fatalf("Failed to create HMAC utility: %v", err)
	}

	// Generate signature with SHA512 and BASE64
	signature512, err := hmacUtil512.Sign(message)
	if err != nil {
		log.Fatalf("Failed to sign message with SHA512: %v", err)
	}
	fmt.Printf("Generated signature (SHA512, BASE64): %s\n", signature512)

	// Example 3: Verification with tampered message
	tamperedMessage := []byte("This is a sensitive message that was tampered with")
	err = hmacUtil.Verify(tamperedMessage, signature)
	if err != nil {
		fmt.Printf("Expected failure on tampered message: %v\n", err)
	} else {
		log.Fatal("Security error: Tampered message was incorrectly verified!")
	}
}
