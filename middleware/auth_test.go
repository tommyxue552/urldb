package middleware

import (
	"os"
	"testing"

	"github.com/ctwj/urldb/db/entity"
)

const testJWTSecret = "0123456789abcdef0123456789abcdef"

func TestGenerateTokenRequiresEnvironmentSecret(t *testing.T) {
	t.Setenv(jwtSecretEnv, "")
	if _, err := GenerateToken(&entity.User{Username: "admin"}); err == nil {
		t.Fatal("GenerateToken accepted a missing JWT_SECRET")
	}
}

func TestJWTSecretRotationAcceptsPreviousSecret(t *testing.T) {
	t.Setenv(jwtSecretEnv, testJWTSecret)
	t.Setenv(jwtPreviousSecretEnv, "")
	token, err := GenerateToken(&entity.User{Username: "admin", Role: "admin"})
	if err != nil {
		t.Fatalf("GenerateToken: %v", err)
	}

	previous := os.Getenv(jwtSecretEnv)
	t.Setenv(jwtSecretEnv, "fedcba9876543210fedcba9876543210")
	t.Setenv(jwtPreviousSecretEnv, previous)
	claims, err := parseToken(token)
	if err != nil {
		t.Fatalf("parseToken with previous key: %v", err)
	}
	if claims.Username != "admin" {
		t.Fatalf("username = %q, want admin", claims.Username)
	}
}
