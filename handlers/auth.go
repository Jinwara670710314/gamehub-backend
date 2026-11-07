package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"time"

	"gamehub-backend/middleware"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type AuthHandler struct {
	DB *sql.DB
}

type RegisterRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type ChangePasswordRequest struct {
	OldPassword string `json:"old_password"`
	NewPassword string `json:"new_password"`
}

type RegisterResponse struct {
	Status   string `json:"status"`
	Message  string `json:"message"`
	ID       int64  `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Role     string `json:"role"`
}

type LoginResponse struct {
	Status   string `json:"status"`
	Message  string `json:"message"`
	Token    string `json:"token"`
	ID       int64  `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Role     string `json:"role"`
}

type Claims struct {
	UserID   int64  `json:"user_id"`
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

// API สมัครสมาชิก
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "ข้อมูลไม่ถูกต้อง", http.StatusBadRequest)
		return
	}

	req.Username = strings.TrimSpace(req.Username)
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))

	if req.Username == "" || req.Email == "" || strings.TrimSpace(req.Password) == "" {
		http.Error(w, "กรุณากรอกข้อมูลให้ครบ", http.StatusBadRequest)
		return
	}

	if len(req.Password) < 8 {
		http.Error(w, "รหัสผ่านต้องมีอย่างน้อย 8 ตัวอักษร", http.StatusBadRequest)
		return
	}

	var exists bool

	err := h.DB.QueryRowContext(r.Context(), `
		SELECT EXISTS (
			SELECT 1
			FROM public.users
			WHERE LOWER(TRIM(username)) = LOWER($1)
			   OR LOWER(TRIM(email)) = LOWER($2)
		)
	`, req.Username, req.Email).Scan(&exists)

	if err != nil {
		http.Error(w, "ไม่สามารถตรวจสอบข้อมูลสมาชิกได้", http.StatusInternalServerError)
		return
	}

	if exists {
		http.Error(w, "ชื่อผู้ใช้หรืออีเมลนี้ถูกใช้งานแล้ว", http.StatusConflict)
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword(
		[]byte(req.Password),
		bcrypt.DefaultCost,
	)

	if err != nil {
		http.Error(w, "ไม่สามารถเข้ารหัสรหัสผ่านได้", http.StatusInternalServerError)
		return
	}

	var response RegisterResponse

	err = h.DB.QueryRowContext(r.Context(), `
		INSERT INTO public.users (username, email, password_hash)
		VALUES ($1, $2, $3)
		RETURNING id, username, email, role
	`,
		req.Username,
		req.Email,
		string(hashedPassword),
	).Scan(
		&response.ID,
		&response.Username,
		&response.Email,
		&response.Role,
	)

	if err != nil {
		http.Error(w, "ไม่สามารถสมัครสมาชิกได้", http.StatusInternalServerError)
		return
	}

	response.Status = "success"
	response.Message = "สมัครสมาชิกสำเร็จ"

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// API เข้าสู่ระบบ
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "ข้อมูลไม่ถูกต้อง", http.StatusBadRequest)
		return
	}

	req.Email = strings.ToLower(strings.TrimSpace(req.Email))

	if req.Email == "" || req.Password == "" {
		http.Error(w, "กรุณากรอกอีเมลและรหัสผ่าน", http.StatusBadRequest)
		return
	}

	var id int64
	var username string
	var email string
	var passwordHash string
	var role string
	var status string

	err := h.DB.QueryRowContext(r.Context(), `
		SELECT id, username, email, password_hash, role, status
		FROM public.users
		WHERE LOWER(TRIM(email)) = $1
	`, req.Email).Scan(
		&id,
		&username,
		&email,
		&passwordHash,
		&role,
		&status,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "อีเมลหรือรหัสผ่านไม่ถูกต้อง", http.StatusUnauthorized)
			return
		}

		http.Error(w, "ไม่สามารถเข้าสู่ระบบได้", http.StatusInternalServerError)
		return
	}

	if status != "active" {
		http.Error(w, "บัญชีนี้ไม่สามารถใช้งานได้", http.StatusForbidden)
		return
	}

	if err := bcrypt.CompareHashAndPassword(
		[]byte(passwordHash),
		[]byte(req.Password),
	); err != nil {
		http.Error(w, "อีเมลหรือรหัสผ่านไม่ถูกต้อง", http.StatusUnauthorized)
		return
	}

	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		http.Error(w, "ระบบยังไม่ได้ตั้งค่า JWT_SECRET", http.StatusInternalServerError)
		return
	}

	claims := Claims{
		UserID:   id,
		Username: username,
		Role:     role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		http.Error(w, "ไม่สามารถสร้าง Token ได้", http.StatusInternalServerError)
		return
	}

	response := LoginResponse{
		Status:   "success",
		Message:  "เข้าสู่ระบบสำเร็จ",
		Token:    tokenString,
		ID:       id,
		Username: username,
		Email:    email,
		Role:     role,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// API เปลี่ยนรหัสผ่าน
func (h *AuthHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req ChangePasswordRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "ข้อมูลไม่ถูกต้อง", http.StatusBadRequest)
		return
	}

	if req.OldPassword == "" || req.NewPassword == "" {
		http.Error(w, "กรุณากรอกรหัสผ่านเดิมและรหัสผ่านใหม่", http.StatusBadRequest)
		return
	}

	if len(req.NewPassword) < 8 {
		http.Error(w, "รหัสผ่านใหม่ต้องมีอย่างน้อย 8 ตัวอักษร", http.StatusBadRequest)
		return
	}

	if req.OldPassword == req.NewPassword {
		http.Error(w, "รหัสผ่านใหม่ต้องไม่ซ้ำกับรหัสผ่านเดิม", http.StatusBadRequest)
		return
	}

	// อ่าน User ID จาก JWT
	claims, ok := middleware.GetUser(r)
	if !ok {
		http.Error(w, "กรุณาเข้าสู่ระบบก่อน", http.StatusUnauthorized)
		return
	}

	// อ่านรหัสผ่านเดิมจาก Database
	var passwordHash string

	err := h.DB.QueryRowContext(r.Context(), `
		SELECT password_hash
		FROM public.users
		WHERE id = $1
	`, claims.UserID).Scan(&passwordHash)

	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "ไม่พบบัญชีผู้ใช้", http.StatusUnauthorized)
			return
		}

		http.Error(w, "ไม่สามารถตรวจสอบรหัสผ่านได้", http.StatusInternalServerError)
		return
	}

	// ตรวจสอบรหัสผ่านเดิม
	if err := bcrypt.CompareHashAndPassword(
		[]byte(passwordHash),
		[]byte(req.OldPassword),
	); err != nil {
		http.Error(w, "รหัสผ่านเดิมไม่ถูกต้อง", http.StatusUnauthorized)
		return
	}

	// เข้ารหัสรหัสผ่านใหม่
	newPasswordHash, err := bcrypt.GenerateFromPassword(
		[]byte(req.NewPassword),
		bcrypt.DefaultCost,
	)

	if err != nil {
		http.Error(w, "ไม่สามารถเข้ารหัสรหัสผ่านใหม่ได้", http.StatusInternalServerError)
		return
	}

	// บันทึกรหัสผ่านใหม่
	result, err := h.DB.ExecContext(r.Context(), `
		UPDATE public.users
		SET password_hash = $1,
		    updated_at = NOW()
		WHERE id = $2
	`, string(newPasswordHash), claims.UserID)

	if err != nil {
		http.Error(w, "ไม่สามารถเปลี่ยนรหัสผ่านได้", http.StatusInternalServerError)
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil || rowsAffected == 0 {
		http.Error(w, "ไม่สามารถเปลี่ยนรหัสผ่านได้", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"message": "เปลี่ยนรหัสผ่านสำเร็จ",
	})
}
