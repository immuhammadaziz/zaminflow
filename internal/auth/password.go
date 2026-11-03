package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"golang.org/x/crypto/argon2"
)

const (
	argonMemory      = 64 * 1024
	argonIterations  = 3
	argonParallelism = 1
	argonKeyLength   = 32
	argonSaltLength  = 16
)

var commonPasswords = map[string]struct{}{
	"password1234": {},
	"qwerty123456": {},
	"123456789012": {},
	"iloveyou1234": {},
}

func validatePassword(password string) error {
	if !utf8.ValidString(password) {
		return errors.New("password must be valid UTF-8")
	}
	if utf8.RuneCountInString(password) < 12 || len(password) > 128 {
		return errors.New("password must contain 12 to 128 characters")
	}
	if _, common := commonPasswords[strings.ToLower(password)]; common {
		return errors.New("password is too common")
	}
	return nil
}

func hashPassword(password string) (string, error) {
	salt := make([]byte, argonSaltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("generate password salt: %w", err)
	}
	hash := argon2.IDKey([]byte(password), salt, argonIterations, argonMemory, argonParallelism, argonKeyLength)
	return fmt.Sprintf("$argon2id$v=19$m=%d,t=%d,p=%d$%s$%s",
		argonMemory, argonIterations, argonParallelism,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(hash)), nil
}

func verifyPassword(encoded, password string) (bool, error) {
	var memory, iterations uint32
	var parallelism uint8
	var saltEncoded, hashEncoded string
	if _, err := fmt.Sscanf(encoded, "$argon2id$v=19$m=%d,t=%d,p=%d$%s", &memory, &iterations, &parallelism, &saltEncoded); err != nil {
		return false, fmt.Errorf("parse password hash: %w", err)
	}
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[0] != "" || parts[1] != "argon2id" || parts[2] != "v=19" {
		return false, errors.New("invalid password hash format")
	}
	hashEncoded = parts[5]
	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false, fmt.Errorf("decode password salt: %w", err)
	}
	want, err := base64.RawStdEncoding.DecodeString(hashEncoded)
	if err != nil {
		return false, fmt.Errorf("decode password hash: %w", err)
	}
	if memory < 8*1024 || memory > 1024*1024 || iterations == 0 || iterations > 10 || parallelism == 0 {
		return false, errors.New("password hash parameters are outside accepted bounds")
	}
	got := argon2.IDKey([]byte(password), salt, iterations, memory, parallelism, uint32(len(want)))
	return subtle.ConstantTimeCompare(got, want) == 1, nil
}
