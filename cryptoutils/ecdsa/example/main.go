package main

import (
	"fmt"
	"log"
	"securedesign/cryptoutils/ecdsa"
)

func main() {
	ecdsaService := ecdsa.NewECDSAService()
	// Generate a new key pair
	keyPair, err := ecdsaService.GenerateKeyPair()
	if err != nil {
		log.Fatalf("Failed to generate key pair: %v", err)
	}

	// Sign a message
	message := []byte("This is a secure message")
	signature, err := ecdsaService.Sign(keyPair.PrivateKey, message)
	if err != nil {
		log.Fatalf("Failed to sign message: %v", err)
	}

	// Verify the signature
	valid, err := ecdsaService.Verify(keyPair.PublicKey, message, signature)
	if err != nil {
		log.Fatalf("Failed to verify signature: %v", err)
	}

	fmt.Printf("Signature verified: %v\n", valid)
}
