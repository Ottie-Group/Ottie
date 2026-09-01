package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"strings"

	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/bcrypt"
)

// Argon2id parameters for zero-knowledge per-user Key Encryption Key (KEK) derivation.
const (
	argonMemory      = 64 * 1024 // 64 MB
	argonIterations  = 3
	argonParallelism = 2
	keyLength        = 32 // 256-bit AES key
	saltLength       = 16
)

// GenerateRandomBytes returns cryptographically secure random bytes of given length.
func GenerateRandomBytes(n int) ([]byte, error) {
	b := make([]byte, n)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		return nil, err
	}
	return b, nil
}

// GenerateSalt returns a base64 encoded random salt for password key derivation.
func GenerateSalt() (string, error) {
	b, err := GenerateRandomBytes(saltLength)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(b), nil
}

// DeriveKEK derives a 32-byte Key Encryption Key from a user's password and salt using Argon2id.
func DeriveKEK(password string, saltB64 string) ([]byte, error) {
	salt, err := base64.StdEncoding.DecodeString(saltB64)
	if err != nil {
		return nil, errors.New("invalid salt encoding")
	}
	if len(salt) < 16 {
		return nil, errors.New("salt is too short")
	}
	kek := argon2.IDKey([]byte(password), salt, argonIterations, argonMemory, argonParallelism, keyLength)
	return kek, nil
}

// GenerateDEK generates a new random 32-byte Data Encryption Key (DEK) for a user.
func GenerateDEK() ([]byte, error) {
	return GenerateRandomBytes(keyLength)
}

// EncryptAESGCM encrypts plaintext bytes with a 32-byte key using AES-256-GCM.
// Returns base64(nonce + ciphertext).
func EncryptAESGCM(plaintext []byte, key []byte) (string, error) {
	if len(key) != 32 {
		return "", errors.New("key must be 32 bytes for AES-256")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// DecryptAESGCM decrypts a base64(nonce + ciphertext) string using a 32-byte key.
func DecryptAESGCM(encodedCiphertext string, key []byte) ([]byte, error) {
	if len(key) != 32 {
		return nil, errors.New("key must be 32 bytes for AES-256")
	}
	data, err := base64.StdEncoding.DecodeString(encodedCiphertext)
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return nil, errors.New("ciphertext too short")
	}
	nonce, ct := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ct, nil)
	if err != nil {
		return nil, err
	}
	return plaintext, nil
}

// WrapDEK encrypts the user's DEK with their password-derived KEK.
func WrapDEK(dek []byte, kek []byte) (string, error) {
	return EncryptAESGCM(dek, kek)
}

// UnwrapDEK decrypts the user's DEK with their password-derived KEK.
func UnwrapDEK(wrappedDEK string, kek []byte) ([]byte, error) {
	return DecryptAESGCM(wrappedDEK, kek)
}

// EncryptSecret encrypts a plaintext TOTP secret using the user's raw 32-byte DEK.
func EncryptSecret(secret string, userDEK []byte) (string, error) {
	return EncryptAESGCM([]byte(secret), userDEK)
}

// DecryptSecret decrypts an encrypted TOTP secret using the user's raw 32-byte DEK.
func DecryptSecret(encryptedSecret string, userDEK []byte) (string, error) {
	pt, err := DecryptAESGCM(encryptedSecret, userDEK)
	if err != nil {
		return "", err
	}
	return string(pt), nil
}

// GenerateRecoveryCode creates a formatted emergency recovery key (e.g. OT-8492-3841-9920-5512) and its bcrypt hash.
func GenerateRecoveryCode() (string, string, error) {
	b, err := GenerateRandomBytes(8)
	if err != nil {
		return "", "", err
	}
	hexStr := strings.ToUpper(hex.EncodeToString(b))
	formatted := fmt.Sprintf("OT-%s-%s-%s-%s", hexStr[0:4], hexStr[4:8], hexStr[8:12], hexStr[12:16])
	hash, err := bcrypt.GenerateFromPassword([]byte(formatted), bcrypt.DefaultCost)
	if err != nil {
		return "", "", err
	}
	return formatted, string(hash), nil
}

// VerifyRecoveryCode checks a raw recovery code against a stored bcrypt hash.
func VerifyRecoveryCode(rawCode, storedHash string) bool {
	if rawCode == "" || storedHash == "" {
		return false
	}
	clean := strings.ToUpper(strings.TrimSpace(rawCode))
	return bcrypt.CompareHashAndPassword([]byte(storedHash), []byte(clean)) == nil
}
