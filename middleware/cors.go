package middleware

import (
	"net/http"
	"os"
	"strings"
)

// CORS Middleware
func CORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")

		// อ่านรายการ Origin ที่อนุญาตจาก Environment Variable
		allowedOrigins := make(map[string]bool)

		originsEnv := os.Getenv("ALLOWED_ORIGINS")

		if originsEnv == "" {
			// ค่าเริ่มต้นสำหรับการทดสอบบนเครื่องตัวเอง
			allowedOrigins["http://127.0.0.1:3000"] = true
			allowedOrigins["http://localhost:3000"] = true
		} else {
			// รองรับหลาย Origin โดยคั่นด้วยเครื่องหมายจุลภาค
			for _, allowedOrigin := range strings.Split(originsEnv, ",") {
				allowedOrigin = strings.TrimSpace(allowedOrigin)

				if allowedOrigin != "" {
					allowedOrigins[allowedOrigin] = true
				}
			}
		}

		// กำหนด CORS Headers เฉพาะ Origin ที่ได้รับอนุญาต
		if allowedOrigins[origin] {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Add("Vary", "Origin")

			w.Header().Set(
				"Access-Control-Allow-Methods",
				"GET, POST, PUT, PATCH, DELETE, OPTIONS",
			)

			w.Header().Set(
				"Access-Control-Allow-Headers",
				"Content-Type, Authorization",
			)
		}

		// Handle browser preflight request
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}
