package archerC5

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/hex"
	"strconv"
	"testing"
)

func TestEncryptPayload(t *testing.T) {
	// Generate a dummy 1024-bit RSA key pair for testing
	// 1024 bits = 128 bytes = 256 hex characters
	// 1024 bits / 4 bits for single hex char = 256 hex characters
	privateKey, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		t.Fatalf("Failed to generate test key: %v", err)
	}

	// Extract the public modulus and exponent as hex strings
	// privateKey has embedded PublicKey(no variable name) struct so it can directly access its field
	// privateKey.PublicKey.N

	// .Text() is a string convertor built-in method for math/big objects
	modulusHex := privateKey.N.Text(16)
	exponentHex := strconv.FormatInt(int64(privateKey.E), 16)
	originalPassword := "MySuperSecretRouterPassword"

	encryptedHex, err := EncryptPayload(originalPassword, modulusHex, exponentHex)
	if err != nil {
		t.Fatalf("EncryptPayload failed: %v", err)
	}

	// Verify TP-Link length requirement
	if len(encryptedHex) != 256 {
		t.Errorf("Expected exactly 256 characters, got %d", len(encryptedHex))
	}

	// Decode the hex string back to raw bytes
	ciphertext, err := hex.DecodeString(encryptedHex)
	if err != nil {
		t.Fatalf("Failed to decode hex string: %v", err)
	}

	// Decrypt
	decryptedBytes, err := rsa.DecryptPKCS1v15(rand.Reader, privateKey, ciphertext)
	if err != nil {
		t.Fatalf("Decryption failed (padding or math error): %v", err)
	}

	decodedPasswordBytes, err := base64.StdEncoding.DecodeString(string(decryptedBytes))
	if err != nil {
		t.Fatalf("Failed to decode base64 string: %v", err)
	}
	// Compare the decrypted password
	if string(decodedPasswordBytes) != originalPassword {
		t.Errorf("Expected %q, but got %q", originalPassword, string(decodedPasswordBytes))
	}
}
