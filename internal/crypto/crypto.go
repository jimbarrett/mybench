package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"io"

	"golang.org/x/crypto/argon2"
)

const (
	saltLen    = 16
	keyLen     = 32 // AES-256
	argonTime  = 1
	argonMem   = 64 * 1024
	argonParts = 4
)

// Vault holds the derived encryption key in memory.
type Vault struct {
	key []byte
}

// NewVault derives an encryption key from the master password and salt.
func NewVault(password string, salt []byte) *Vault {
	key := argon2.IDKey([]byte(password), salt, argonTime, argonMem, argonParts, keyLen)
	return &Vault{key: key}
}

// GenerateSalt returns a random salt for key derivation.
func GenerateSalt() ([]byte, error) {
	salt := make([]byte, saltLen)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return nil, err
	}
	return salt, nil
}

// HashPassword creates a verification hash of the master password.
// This is stored so we can verify the password on subsequent launches
// without needing to decrypt anything.
func HashPassword(password string, salt []byte) string {
	hash := argon2.IDKey([]byte(password), salt, argonTime, argonMem, argonParts, keyLen)
	return base64.StdEncoding.EncodeToString(hash)
}

// VerifyPassword checks a password against a stored hash.
func VerifyPassword(password string, salt []byte, storedHash string) bool {
	hash := argon2.IDKey([]byte(password), salt, argonTime, argonMem, argonParts, keyLen)
	decoded, err := base64.StdEncoding.DecodeString(storedHash)
	if err != nil {
		return false
	}
	return subtle.ConstantTimeCompare(hash, decoded) == 1
}

// Encrypt encrypts plaintext using AES-256-GCM.
// Returns base64-encoded ciphertext with nonce prepended.
func (v *Vault) Encrypt(plaintext string) (string, error) {
	if len(plaintext) == 0 {
		return "", nil
	}

	block, err := aes.NewCipher(v.key)
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

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Decrypt decrypts base64-encoded ciphertext produced by Encrypt.
func (v *Vault) Decrypt(encoded string) (string, error) {
	if len(encoded) == 0 {
		return "", nil
	}

	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(v.key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", errors.New("ciphertext too short")
	}

	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", errors.New("decryption failed: wrong master password or corrupted data")
	}

	return string(plaintext), nil
}
