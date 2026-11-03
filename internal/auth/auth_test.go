package auth

import (
	"strings"
	"testing"
	"time"
)

func TestPasswordHashRoundTrip(t *testing.T) {
	password := "a-strong-unique-password-739!"
	encoded, err := hashPassword(password)
	if err != nil {
		t.Fatal(err)
	}
	matches, err := verifyPassword(encoded, password)
	if err != nil {
		t.Fatal(err)
	}
	if !matches {
		t.Fatal("correct password did not match its hash")
	}
	matches, err = verifyPassword(encoded, password+"x")
	if err != nil {
		t.Fatal(err)
	}
	if matches {
		t.Fatal("incorrect password matched its hash")
	}
}

func TestValidatePassword(t *testing.T) {
	for _, password := range []string{"short", "password1234", strings.Repeat("x", 129)} {
		if err := validatePassword(password); err == nil {
			t.Errorf("validatePassword(%q) unexpectedly succeeded", password)
		}
	}
	if err := validatePassword("a-strong-unique-password-739!"); err != nil {
		t.Fatalf("valid password rejected: %v", err)
	}
}

func TestTOTPUsesRFC6238SixDigitSHA1Vector(t *testing.T) {
	code, err := totpCode("GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ", time.Unix(59, 0))
	if err != nil {
		t.Fatal(err)
	}
	if code != "287082" {
		t.Fatalf("TOTP code = %s, want 287082", code)
	}
	if !verifyTOTP("GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ", code, time.Unix(59, 0)) {
		t.Fatal("valid TOTP was rejected")
	}
	if verifyTOTP("GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ", "not-a-code", time.Unix(59, 0)) {
		t.Fatal("invalid TOTP was accepted")
	}
}

func TestEncryptSecretAuthenticatesCiphertext(t *testing.T) {
	secret := "a-secret-that-is-long-enough-for-application-use"
	ciphertext, err := encryptSecret(secret, "totp-secret")
	if err != nil {
		t.Fatal(err)
	}
	plaintext, err := decryptSecret(secret, ciphertext)
	if err != nil {
		t.Fatal(err)
	}
	if plaintext != "totp-secret" {
		t.Fatalf("decrypted secret = %q", plaintext)
	}
	if _, err := decryptSecret(secret+"different", ciphertext); err == nil {
		t.Fatal("ciphertext was accepted with the wrong key")
	}
}

func TestCSRFTokenMustBeSignedByApplicationSecret(t *testing.T) {
	token, err := newCSRFToken("csrf-test-secret")
	if err != nil {
		t.Fatal(err)
	}
	if !validCSRFToken("csrf-test-secret", token) {
		t.Fatal("valid signed CSRF token was rejected")
	}
	if validCSRFToken("different-secret", token) {
		t.Fatal("CSRF token signed by a different secret was accepted")
	}
	if validCSRFToken("csrf-test-secret", "attacker-controlled-token") {
		t.Fatal("unsigned CSRF token was accepted")
	}
}

func TestAccessTokenVerifiesSignatureAndExpiry(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	claims := accessClaims{
		Subject: "2f7f8e71-5fe3-492c-88a5-d8f753dbd899",
		Session: "04cb8e6e-311d-4c0a-a31a-2f9841cb773b",
		Issuer:  "zaminflow-api", Audience: "zaminflow-clients",
		IssuedAt: now.Unix(), Expires: now.Add(time.Minute).Unix(),
	}
	token, err := signAccessToken([]byte("test-secret"), claims)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := verifyAccessToken([]byte("test-secret"), token, claims.Issuer, claims.Audience, now); err != nil {
		t.Fatalf("valid token rejected: %v", err)
	}
	if _, err := verifyAccessToken([]byte("wrong-secret"), token, claims.Issuer, claims.Audience, now); err == nil {
		t.Fatal("token with invalid signature was accepted")
	}
	if _, err := verifyAccessToken([]byte("test-secret"), token, claims.Issuer, claims.Audience, now.Add(2*time.Minute)); err == nil {
		t.Fatal("expired token was accepted")
	}
}

func TestIsUUIDRejectsMalformedIdentifiers(t *testing.T) {
	for _, id := range []string{"", "not-a-uuid", "zzzzzzzz-zzzz-zzzz-zzzz-zzzzzzzzzzzz", "12345678/1234-1234-1234-123456789012"} {
		if isUUID(id) {
			t.Errorf("isUUID(%q) unexpectedly succeeded", id)
		}
	}
	if !isUUID("2f7f8e71-5fe3-492c-88a5-d8f753dbd899") {
		t.Fatal("valid UUID rejected")
	}
}

func FuzzValidCSRFToken(f *testing.F) {
	f.Add("test-secret", "nonce.signature")
	f.Add("", "")
	f.Fuzz(func(t *testing.T, secret, token string) {
		_ = validCSRFToken(secret, token)
	})
}

func FuzzVerifyAccessToken(f *testing.F) {
	f.Add("test-secret", "not-a-token", "zaminflow-api", "zaminflow-clients")
	f.Add("", "", "", "")
	f.Fuzz(func(t *testing.T, secret, token, issuer, audience string) {
		_, _ = verifyAccessToken([]byte(secret), token, issuer, audience, time.Now())
	})
}
