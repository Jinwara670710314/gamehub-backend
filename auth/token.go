package auth

import (
	"errors"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// ข้อผิดพลาดเมื่อไม่พบ JWT_SECRET
var ErrMissingJWTSecret = errors.New("ไม่พบ JWT_SECRET")

// ข้อมูลผู้ใช้ที่จัดเก็บใน JWT
type Claims struct {
	UserID   int64  `json:"user_id"`
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

// สร้าง JWT Token สำหรับผู้ใช้
func GenerateToken(userID int64, username string, role string) (string, error) {
	secret := os.Getenv("JWT_SECRET")

	if secret == "" {
		return "", ErrMissingJWTSecret
	}

	now := time.Now()

	claims := Claims{
		UserID:   userID,
		Username: username,
		Role:     role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	return token.SignedString([]byte(secret))
}

// ตรวจสอบ JWT Token และอ่านข้อมูลผู้ใช้
func ParseToken(tokenString string) (*Claims, error) {
	secret := os.Getenv("JWT_SECRET")

	if secret == "" {
		return nil, ErrMissingJWTSecret
	}

	claims := &Claims{}

	token, err := jwt.ParseWithClaims(
		tokenString,
		claims,
		func(token *jwt.Token) (interface{}, error) {
			if token.Method != jwt.SigningMethodHS256 {
				return nil, jwt.ErrSignatureInvalid
			}

			return []byte(secret), nil
		},
	)

	if err != nil {
		return nil, err
	}

	if token == nil || !token.Valid {
		return nil, errors.New("Token ไม่ถูกต้อง")
	}

	return claims, nil
}
