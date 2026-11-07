package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

// ข้อมูลผู้ใช้ที่อ่านได้จาก JWT
type Claims struct {
	UserID   int64  `json:"user_id"`
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

// ใช้เก็บข้อมูลผู้ใช้ใน Request Context
type contextKey string

const userContextKey contextKey = "user"

// อ่านข้อมูลผู้ใช้จาก Context
func GetUser(r *http.Request) (*Claims, bool) {
	claims, ok := r.Context().Value(userContextKey).(*Claims)
	return claims, ok
}

// ตรวจสอบ JWT Token
func Auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		authHeader := r.Header.Get("Authorization")

		if authHeader == "" {
			writeError(w, http.StatusUnauthorized, "กรุณาเข้าสู่ระบบก่อน")
			return
		}

		parts := strings.Fields(authHeader)

		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			writeError(w, http.StatusUnauthorized, "รูปแบบ Token ไม่ถูกต้อง")
			return
		}

		tokenString := parts[1]

		secret := os.Getenv("JWT_SECRET")

		if secret == "" {
			writeError(w, http.StatusInternalServerError, "ไม่พบ JWT_SECRET")
			return
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

		if err != nil || !token.Valid {
			writeError(w, http.StatusUnauthorized, "Token ไม่ถูกต้องหรือหมดอายุ")
			return
		}

		ctx := context.WithValue(r.Context(), userContextKey, claims)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// ตรวจสอบสิทธิ์ Admin
// ต้องใช้ร่วมกับ Auth Middleware
func Admin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		claims, ok := GetUser(r)

		if !ok {
			writeError(w, http.StatusUnauthorized, "กรุณาเข้าสู่ระบบก่อน")
			return
		}

		if claims.Role != "admin" {
			writeError(w, http.StatusForbidden, "คุณไม่มีสิทธิ์เข้าถึงส่วนนี้")
			return
		}

		next.ServeHTTP(w, r)
	})
}

// ส่ง Error ในรูปแบบ JSON
func writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	json.NewEncoder(w).Encode(map[string]string{
		"status":  "error",
		"message": message,
	})
}
