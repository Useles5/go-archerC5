package archerC5

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/hex"
	"fmt"
	"math/big"
	"strconv"
	"strings"
)

func EncryptPayload(plaintext, modulusHex, exponentHex string) (string, error) {
	exponent, err := strconv.ParseInt(exponentHex, 16, 64)
	if err != nil {
		return "", fmt.Errorf("failed to parse exponent %q: %w", exponentHex, err)
	}

	// modulus(N) requires Big Int as it cannot be stored in int64
	modulus, okay := new(big.Int).SetString(modulusHex, 16)
	if !okay {
		return "", fmt.Errorf("failed to parse modulus %q", modulusHex)
	}

	pubKey := &rsa.PublicKey{
		N: modulus,
		E: int(exponent),
	}

	// EncryptPKCS1v15 is deprecated
	encryptedBytes, err := rsa.EncryptPKCS1v15(rand.Reader, pubKey, []byte(plaintext))
	if err != nil {
		return "", err
	}

	// Output requires a Hex string
	encryptedHex := hex.EncodeToString(encryptedBytes)

	// Add padding to fit the required size
	if len(encryptedHex) < 256 {
		paddingReq := 256 - len(encryptedHex)

		encryptedHex = strings.Repeat("0", paddingReq) + encryptedHex
	}

	return encryptedHex, nil

}
