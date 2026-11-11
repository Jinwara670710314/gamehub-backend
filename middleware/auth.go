
package middleware

import (
	"context"
	"database/sql"
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

		if claims.UserID <= 0 {
			writeError(w, http.StatusUnauthorized, "ข้อมูลผู้ใช้ใน Token ไม่ถูกต้อง")
			return
		}

		ctx := context.WithValue(r.Context(), userContextKey, claims)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// ตรวจสอบว่าบัญชีผู้ใช้ยัง Active อยู่หรือไม่
// ต้องใช้ร่วมกับ Auth Middleware
func Active(db *sql.DB) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			claims, ok := GetUser(r)

			if !ok {
				writeError(w, http.StatusUnauthorized, "กรุณาเข้าสู่ระบบก่อน")
				return
			}

			var status string

			err := db.QueryRowContext(r.Context(), `
				SELECT status
				FROM public.users
				WHERE id = $1
			`, claims.UserID).Scan(&status)

			if err != nil {
				if err == sql.ErrNoRows {
					writeError(w, http.StatusUnauthorized, "ไม่พบบัญชีผู้ใช้")
					return
				}

				writeError(w, http.StatusInternalServerError, "ไม่สามารถตรวจสอบสถานะบัญชีได้")
				return
			}

			if status != "active" {
				writeError(w, http.StatusForbidden, "บัญชีนี้ถูกระงับการใช้งาน")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// ตรวจสอบสิทธิ์ Admin จากข้อมูลล่าสุดใน Database
// ต้องใช้ร่วมกับ Auth Middleware
func Admin(db *sql.DB) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			claims, ok := GetUser(r)

			if !ok {
				writeError(w, http.StatusUnauthorized, "กรุณาเข้าสู่ระบบก่อน")
				return
			}

			var role string
			var status string

			err := db.QueryRowContext(r.Context(), `
				SELECT role, status
				FROM public.users
				WHERE id = $1
			`, claims.UserID).Scan(&role, &status)

			if err != nil {
				if err == sql.ErrNoRows {
					writeError(w, http.StatusUnauthorized, "ไม่พบบัญชีผู้ใช้")
					return
				}

				writeError(w, http.StatusInternalServerError, "ไม่สามารถตรวจสอบสิทธิ์ผู้ใช้ได้")
				return
			}

			if status != "active" {
				writeError(w, http.StatusForbidden, "บัญชีนี้ไม่สามารถใช้งานได้")
				return
			}

			if role != "admin" {
				writeError(w, http.StatusForbidden, "คุณไม่มีสิทธิ์เข้าถึงส่วนนี้")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
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
