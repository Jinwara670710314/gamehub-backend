package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"

	"gamehub-backend/auth"
	"gamehub-backend/middleware"
	"gamehub-backend/store"
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

	// ตรวจสอบชื่อผู้ใช้และอีเมลซ้ำ
	exists, err := store.CheckUserExists(
		r.Context(),
		h.DB,
		req.Username,
		req.Email,
	)

	if err != nil {
		http.Error(w, "ไม่สามารถตรวจสอบข้อมูลสมาชิกได้", http.StatusInternalServerError)
		return
	}

	if exists {
		http.Error(w, "ชื่อผู้ใช้หรืออีเมลนี้ถูกใช้งานแล้ว", http.StatusConflict)
		return
	}

	// เข้ารหัสรหัสผ่าน
	hashedPassword, err := auth.HashPassword(req.Password)
	if err != nil {
		http.Error(w, "ไม่สามารถเข้ารหัสรหัสผ่านได้", http.StatusInternalServerError)
		return
	}

	// บันทึกสมาชิกใหม่
	user, err := store.CreateUser(
		r.Context(),
		h.DB,
		req.Username,
		req.Email,
		hashedPassword,
	)

	if err != nil {
		http.Error(w, "ไม่สามารถสมัครสมาชิกได้", http.StatusInternalServerError)
		return
	}

	response := RegisterResponse{
		Status:   "success",
		Message:  "สมัครสมาชิกสำเร็จ",
		ID:       user.ID,
		Username: user.Username,
		Email:    user.Email,
		Role:     user.Role,
	}

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

	// ค้นหาสมาชิกจากอีเมล
	user, err := store.GetUserByEmail(
		r.Context(),
		h.DB,
		req.Email,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "อีเมลหรือรหัสผ่านไม่ถูกต้อง", http.StatusUnauthorized)
			return
		}

		http.Error(w, "ไม่สามารถเข้าสู่ระบบได้", http.StatusInternalServerError)
		return
	}

	if user.Status != "active" {
		http.Error(w, "บัญชีนี้ไม่สามารถใช้งานได้", http.StatusForbidden)
		return
	}

	// ตรวจสอบรหัสผ่าน
	if err := auth.CheckPassword(user.PasswordHash, req.Password); err != nil {
		http.Error(w, "อีเมลหรือรหัสผ่านไม่ถูกต้อง", http.StatusUnauthorized)
		return
	}

	// สร้าง JWT Token
	tokenString, err := auth.GenerateToken(user.ID, user.Username, user.Role)
	if err != nil {
		http.Error(w, "ไม่สามารถสร้าง Token ได้", http.StatusInternalServerError)
		return
	}

	response := LoginResponse{
		Status:   "success",
		Message:  "เข้าสู่ระบบสำเร็จ",
		Token:    tokenString,
		ID:       user.ID,
		Username: user.Username,
		Email:    user.Email,
		Role:     user.Role,
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

	// อ่านรหัสผ่านเดิมจาก Store
	passwordHash, err := store.GetPasswordHash(
		r.Context(),
		h.DB,
		claims.UserID,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "ไม่พบบัญชีผู้ใช้", http.StatusUnauthorized)
			return
		}

		http.Error(w, "ไม่สามารถตรวจสอบรหัสผ่านได้", http.StatusInternalServerError)
		return
	}

	// ตรวจสอบรหัสผ่านเดิม
	if err := auth.CheckPassword(passwordHash, req.OldPassword); err != nil {
		http.Error(w, "รหัสผ่านเดิมไม่ถูกต้อง", http.StatusUnauthorized)
		return
	}

	// เข้ารหัสรหัสผ่านใหม่
	newPasswordHash, err := auth.HashPassword(req.NewPassword)
	if err != nil {
		http.Error(w, "ไม่สามารถเข้ารหัสรหัสผ่านใหม่ได้", http.StatusInternalServerError)
		return
	}

	// บันทึกรหัสผ่านใหม่ผ่าน Store
	updated, err := store.UpdatePassword(
		r.Context(),
		h.DB,
		claims.UserID,
		newPasswordHash,
	)

	if err != nil || !updated {
		http.Error(w, "ไม่สามารถเปลี่ยนรหัสผ่านได้", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"message": "เปลี่ยนรหัสผ่านสำเร็จ",
	})
}
