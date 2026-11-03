package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

type accessClaims struct {
	Subject        string `json:"sub"`
	Session        string `json:"sid"`
	Issuer         string `json:"iss"`
	Audience       string `json:"aud"`
	IssuedAt       int64  `json:"iat"`
	Expires        int64  `json:"exp"`
	EnrollmentOnly bool   `json:"enrollmentOnly,omitempty"`
}

func signAccessToken(secret []byte, claims accessClaims) (string, error) {
	header, err := json.Marshal(map[string]string{"alg": "HS256", "typ": "JWT"})
	if err != nil {
		return "", fmt.Errorf("encode token header: %w", err)
	}
	payload, err := json.Marshal(claims)
	if err != nil {
		return "", fmt.Errorf("encode token claims: %w", err)
	}
	encoded := base64.RawURLEncoding.EncodeToString(header) + "." + base64.RawURLEncoding.EncodeToString(payload)
	mac := hmac.New(sha256.New, secret)
	if _, err := mac.Write([]byte(encoded)); err != nil {
		return "", fmt.Errorf("sign access token: %w", err)
	}
	return encoded + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil)), nil
}

func verifyAccessToken(secret []byte, token, issuer, audience string, now time.Time) (accessClaims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return accessClaims{}, errors.New("invalid access token format")
	}
	headerBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return accessClaims{}, errors.New("invalid access token header")
	}
	var header struct {
		Algorithm string `json:"alg"`
		Type      string `json:"typ"`
	}
	if err := json.Unmarshal(headerBytes, &header); err != nil || header.Algorithm != "HS256" || header.Type != "JWT" {
		return accessClaims{}, errors.New("invalid access token algorithm")
	}
	signature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return accessClaims{}, errors.New("invalid access token signature")
	}
	mac := hmac.New(sha256.New, secret)
	if _, err := mac.Write([]byte(parts[0] + "." + parts[1])); err != nil {
		return accessClaims{}, fmt.Errorf("verify access token signature: %w", err)
	}
	if !hmac.Equal(signature, mac.Sum(nil)) {
		return accessClaims{}, errors.New("invalid access token signature")
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return accessClaims{}, errors.New("invalid access token claims")
	}
	var claims accessClaims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return accessClaims{}, errors.New("invalid access token claims")
	}
	unix := now.Unix()
	if claims.Subject == "" || claims.Session == "" || claims.Issuer != issuer || claims.Audience != audience ||
		claims.IssuedAt <= 0 || claims.IssuedAt > unix+30 || claims.Expires <= unix || claims.Expires <= claims.IssuedAt {
		return accessClaims{}, errors.New("access token claims are invalid or expired")
	}
	return claims, nil
}
