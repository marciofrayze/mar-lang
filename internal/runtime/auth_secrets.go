package runtime

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"
)

// randomCode6 returns a zero-padded 6-digit cryptographically random code.
func randomCode6() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(1_000_000))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}

// randomToken returns a hex-encoded cryptographically random token.
func randomToken(bytesLen int) (string, error) {
	buf := make([]byte, bytesLen)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func hashAuthSecret(value string) string {
	sum := sha256.Sum256([]byte(value))
	return "sha256:" + hex.EncodeToString(sum[:])
}

func storedSecretMatches(storedValue, rawValue string) bool {
	stored := strings.TrimSpace(storedValue)
	raw := strings.TrimSpace(rawValue)
	if stored == "" || raw == "" {
		return false
	}
	if subtle.ConstantTimeCompare([]byte(stored), []byte(raw)) == 1 {
		return true
	}
	return subtle.ConstantTimeCompare([]byte(stored), []byte(hashAuthSecret(raw))) == 1
}
