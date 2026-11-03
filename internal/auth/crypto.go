package auth

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base32"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"hash"
	"strings"
	"time"
)

func deriveKey(secret, purpose string) []byte {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte("zaminflow:" + purpose))
	return mac.Sum(nil)
}

func randomToken(bytes int) (string, error) {
	value := make([]byte, bytes)
	if _, err := rand.Read(value); err != nil {
		return "", fmt.Errorf("generate random token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(value), nil
}

func tokenHash(token string) []byte {
	sum := sha256.Sum256([]byte(token))
	return sum[:]
}

func newCSRFToken(secret string) (string, error) {
	nonce, err := randomToken(32)
	if err != nil {
		return "", err
	}
	mac := hmac.New(sha256.New, deriveKey(secret, "csrf-token"))
	_, _ = mac.Write([]byte(nonce))
	signature := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return nonce + "." + signature, nil
}

func validCSRFToken(secret, token string) bool {
	parts := strings.Split(token, ".")
	if len(parts) != 2 || parts[0] == "" {
		return false
	}
	signature, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return false
	}
	mac := hmac.New(sha256.New, deriveKey(secret, "csrf-token"))
	_, _ = mac.Write([]byte(parts[0]))
	return hmac.Equal(signature, mac.Sum(nil))
}

func encryptSecret(secret, plaintext string) ([]byte, error) {
	block, err := aes.NewCipher(deriveKey(secret, "mfa-encryption"))
	if err != nil {
		return nil, fmt.Errorf("create MFA cipher: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create MFA cipher mode: %w", err)
	}
	nonce := make([]byte, aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("generate MFA nonce: %w", err)
	}
	return append(nonce, aead.Seal(nil, nonce, []byte(plaintext), nil)...), nil
}

func decryptSecret(secret string, ciphertext []byte) (string, error) {
	block, err := aes.NewCipher(deriveKey(secret, "mfa-encryption"))
	if err != nil {
		return "", fmt.Errorf("create MFA cipher: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("create MFA cipher mode: %w", err)
	}
	if len(ciphertext) < aead.NonceSize() {
		return "", fmt.Errorf("encrypted MFA secret is truncated")
	}
	plaintext, err := aead.Open(nil, ciphertext[:aead.NonceSize()], ciphertext[aead.NonceSize():], nil)
	if err != nil {
		return "", fmt.Errorf("decrypt MFA secret: %w", err)
	}
	return string(plaintext), nil
}

func newTOTPSecret() (string, error) {
	value := make([]byte, 20)
	if _, err := rand.Read(value); err != nil {
		return "", fmt.Errorf("generate TOTP secret: %w", err)
	}
	return base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(value), nil
}

func totpCode(secret string, at time.Time) (string, error) {
	decoded, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(strings.ToUpper(strings.TrimSpace(secret)))
	if err != nil {
		return "", fmt.Errorf("decode TOTP secret: %w", err)
	}
	counter := uint64(at.Unix() / 30)
	var message [8]byte
	binary.BigEndian.PutUint64(message[:], counter)
	mac := hmac.New(sha1Hash, decoded)
	_, _ = mac.Write(message[:])
	digest := mac.Sum(nil)
	offset := digest[len(digest)-1] & 0x0f
	value := (binary.BigEndian.Uint32(digest[offset:offset+4]) & 0x7fffffff) % 1_000_000
	return fmt.Sprintf("%06d", value), nil
}

func verifyTOTP(secret, provided string, at time.Time) bool {
	provided = strings.TrimSpace(provided)
	if len(provided) != 6 {
		return false
	}
	for delta := -1; delta <= 1; delta++ {
		expected, err := totpCode(secret, at.Add(time.Duration(delta)*30*time.Second))
		if err == nil && hmac.Equal([]byte(expected), []byte(provided)) {
			return true
		}
	}
	return false
}

func sha1Hash() hash.Hash {
	return sha1.New()
}

func recoveryCodeHash(secret, code string) []byte {
	mac := hmac.New(sha256.New, deriveKey(secret, "recovery-code"))
	_, _ = mac.Write([]byte(strings.ToUpper(strings.TrimSpace(code))))
	return mac.Sum(nil)
}
