package credential

import (
	"encoding/base64"
	"strings"
	"testing"
)

func TestProtectorEncryptDecryptAndFingerprint(t *testing.T) {
	key := base64.StdEncoding.EncodeToString(make([]byte, 32))
	fingerprintKey := base64.StdEncoding.EncodeToString([]byte(strings.Repeat("f", 32)))
	t.Setenv(encryptionKeyEnv, key)
	t.Setenv(fingerprintKeyEnv, fingerprintKey)

	protector, err := LoadFromEnvironment()
	if err != nil {
		t.Fatalf("LoadFromEnvironment() error = %v", err)
	}
	encrypted, err := protector.Encrypt("secret", 7, "ck")
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}
	if !strings.HasPrefix(encrypted, envelopePrefix) || strings.Contains(encrypted, "secret") {
		t.Fatalf("unexpected envelope %q", encrypted)
	}
	plain, err := protector.Decrypt(encrypted, 7, "ck")
	if err != nil || plain != "secret" {
		t.Fatalf("Decrypt() = %q, %v", plain, err)
	}
	if _, err := protector.Decrypt(encrypted, 8, "ck"); err == nil {
		t.Fatal("Decrypt() accepted an envelope for a different account")
	}
	if protector.Fingerprint("secret") == protector.Fingerprint("other") {
		t.Fatal("Fingerprint() collision for distinct test inputs")
	}
}
