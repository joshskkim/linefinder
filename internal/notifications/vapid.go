package notifications

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/base64"
	"fmt"
)

// GenerateVAPIDKeys generates a new VAPID key pair
func GenerateVAPIDKeys() (publicKey, privateKey string, err error) {
	// Generate ECDSA key pair
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate key: %w", err)
	}

	// Encode private key
	privBytes := priv.D.Bytes()
	// Pad to 32 bytes if necessary
	if len(privBytes) < 32 {
		padded := make([]byte, 32)
		copy(padded[32-len(privBytes):], privBytes)
		privBytes = padded
	}
	privateKey = base64.RawURLEncoding.EncodeToString(privBytes)

	// Encode public key (uncompressed point)
	pubBytes := elliptic.Marshal(elliptic.P256(), priv.PublicKey.X, priv.PublicKey.Y)
	publicKey = base64.RawURLEncoding.EncodeToString(pubBytes)

	return publicKey, privateKey, nil
}

// PrintVAPIDKeys generates and prints VAPID keys for .env file
func PrintVAPIDKeys() {
	pub, priv, err := GenerateVAPIDKeys()
	if err != nil {
		fmt.Printf("Error generating keys: %v\n", err)
		return
	}

	fmt.Println("Add these to your .env file:")
	fmt.Println()
	fmt.Printf("VAPID_PUBLIC_KEY=%s\n", pub)
	fmt.Printf("VAPID_PRIVATE_KEY=%s\n", priv)
	fmt.Println("VAPID_SUBJECT=mailto:your-email@example.com")
}
