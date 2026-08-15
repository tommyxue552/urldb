// Package credential provides the repository boundary for provider secrets.
package credential

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
)

const (
	encryptionKeyEnv  = "CREDENTIAL_ENCRYPTION_KEY"
	fingerprintKeyEnv = "CREDENTIAL_FINGERPRINT_KEY"
	envelopePrefix    = "enc:v1:"
)

type Protector struct {
	encryptionKey  []byte
	fingerprintKey []byte
}

func LoadFromEnvironment() (*Protector, error) {
	encryptionKey, err := decodeKey(encryptionKeyEnv)
	if err != nil {
		return nil, err
	}
	fingerprintKey, err := decodeKey(fingerprintKeyEnv)
	if err != nil {
		return nil, err
	}
	return &Protector{encryptionKey: encryptionKey, fingerprintKey: fingerprintKey}, nil
}

func ValidateConfiguration() error {
	_, err := LoadFromEnvironment()
	return err
}

func decodeKey(name string) ([]byte, error) {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return nil, fmt.Errorf("%s must be set to a base64-encoded 32-byte secret", name)
	}
	key, err := base64.StdEncoding.DecodeString(value)
	if err != nil || len(key) != 32 {
		return nil, fmt.Errorf("%s must be a base64-encoded 32-byte secret", name)
	}
	return key, nil
}

func (p *Protector) Encrypt(value string, accountID uint, column string) (string, error) {
	if value == "" {
		return "", nil
	}
	block, err := aes.NewCipher(p.encryptionKey)
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
	ciphertext := gcm.Seal(nil, nonce, []byte(value), p.aad(accountID, column))
	return envelopePrefix + base64.RawStdEncoding.EncodeToString(append(nonce, ciphertext...)), nil
}

func (p *Protector) Decrypt(value string, accountID uint, column string) (string, error) {
	if value == "" {
		return "", nil
	}
	if !strings.HasPrefix(value, envelopePrefix) {
		return "", errors.New("credential is not an encrypted envelope")
	}
	payload, err := base64.RawStdEncoding.DecodeString(strings.TrimPrefix(value, envelopePrefix))
	if err != nil {
		return "", fmt.Errorf("decode credential envelope: %w", err)
	}
	block, err := aes.NewCipher(p.encryptionKey)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	if len(payload) < gcm.NonceSize() {
		return "", errors.New("credential envelope is truncated")
	}
	plaintext, err := gcm.Open(nil, payload[:gcm.NonceSize()], payload[gcm.NonceSize():], p.aad(accountID, column))
	if err != nil {
		return "", fmt.Errorf("decrypt credential envelope: %w", err)
	}
	return string(plaintext), nil
}

func (p *Protector) Fingerprint(value string) string {
	if value == "" {
		return ""
	}
	mac := hmac.New(sha256.New, p.fingerprintKey)
	_, _ = mac.Write([]byte(value))
	return base64.RawStdEncoding.EncodeToString(mac.Sum(nil))
}

func (p *Protector) aad(accountID uint, column string) []byte {
	return []byte(fmt.Sprintf("cks:%d:%s:v1", accountID, column))
}
