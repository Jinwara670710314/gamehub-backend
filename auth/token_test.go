package auth

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// ทดสอบสร้าง JWT และอ่านข้อมูลกลับมา
func TestGenerateAndParseToken(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret-key")

	userID := int64(1)
	username := "testuser"
	role := "member"

	tokenString, err := GenerateToken(userID, username, role)
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}

	if tokenString == "" {
		t.Fatal("GenerateToken() returned empty token")
	}

	claims, err := ParseToken(tokenString)
	if err != nil {
		t.Fatalf("ParseToken() error = %v", err)
	}

	if claims.UserID != userID {
		t.Errorf("UserID = %d, want %d", claims.UserID, userID)
	}

	if claims.Username != username {
		t.Errorf("Username = %s, want %s", claims.Username, username)
	}

	if claims.Role != role {
		t.Errorf("Role = %s, want %s", claims.Role, role)
	}

	if claims.ExpiresAt == nil || !claims.ExpiresAt.Time.After(time.Now()) {
		t.Error("Token expiration time is invalid")
	}
}

// ทดสอบกรณีไม่มี JWT_SECRET ตอนสร้าง Token
func TestGenerateTokenMissingSecret(t *testing.T) {
	t.Setenv("JWT_SECRET", "")

	tokenString, err := GenerateToken(1, "testuser", "member")

	if err != ErrMissingJWTSecret {
		t.Errorf("error = %v, want %v", err, ErrMissingJWTSecret)
	}

	if tokenString != "" {
		t.Error("Expected empty token when JWT_SECRET is missing")
	}
}

// ทดสอบกรณีไม่มี JWT_SECRET ตอนตรวจสอบ Token
func TestParseTokenMissingSecret(t *testing.T) {
	t.Setenv("JWT_SECRET", "")

	claims, err := ParseToken("some-token")

	if err != ErrMissingJWTSecret {
		t.Errorf("error = %v, want %v", err, ErrMissingJWTSecret)
	}

	if claims != nil {
		t.Error("Expected nil claims when JWT_SECRET is missing")
	}
}

// ทดสอบ Token ที่ไม่ถูกต้อง
func TestParseTokenInvalid(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret-key")

	claims, err := ParseToken("invalid-token")

	if err == nil {
		t.Fatal("Expected error for invalid token")
	}

	if claims != nil {
		t.Error("Expected nil claims for invalid token")
	}
}

// ทดสอบ Token ที่ลงลายเซ็นด้วย Secret คนละตัว
func TestParseTokenWrongSecret(t *testing.T) {
	t.Setenv("JWT_SECRET", "correct-secret")

	tokenString, err := GenerateToken(1, "testuser", "member")
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}

	t.Setenv("JWT_SECRET", "wrong-secret")

	claims, err := ParseToken(tokenString)

	if err == nil {
		t.Fatal("Expected error for token signed with another secret")
	}

	if claims != nil {
		t.Error("Expected nil claims for invalid signature")
	}
}

// ทดสอบ Token ที่หมดอายุแล้ว
func TestParseTokenExpired(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret-key")

	claims := Claims{
		UserID:   1,
		Username: "testuser",
		Role:     "member",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	tokenString, err := token.SignedString([]byte("test-secret-key"))
	if err != nil {
		t.Fatalf("SignedString() error = %v", err)
	}

	parsedClaims, err := ParseToken(tokenString)

	if err == nil {
		t.Fatal("Expected error for expired token")
	}

	if parsedClaims != nil {
		t.Error("Expected nil claims for expired token")
	}
}
