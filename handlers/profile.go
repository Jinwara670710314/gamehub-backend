package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"

	"gamehub-backend/middleware"
	"gamehub-backend/store"
)

type ProfileHandler struct {
	DB *sql.DB
}

type ProfileResponse struct {
	Status   string `json:"status"`
	Message  string `json:"message"`
	ID       int64  `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Role     string `json:"role"`
}

type UpdateProfileRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
}

// GET /api/me
func (h *ProfileHandler) GetMe(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	claims, ok := middleware.GetUser(r)
	if !ok {
		http.Error(w, `{"status":"error","message":"กรุณาเข้าสู่ระบบ"}`, http.StatusUnauthorized)
		return
	}

	user, err := store.GetUserProfile(r.Context(), h.DB, claims.UserID)

	if err == sql.ErrNoRows {
		http.Error(w, `{"status":"error","message":"ไม่พบบัญชีผู้ใช้"}`, http.StatusUnauthorized)
		return
	}

	if err != nil {
		http.Error(w, `{"status":"error","message":"เกิดข้อผิดพลาดจากฐานข้อมูล"}`, http.StatusInternalServerError)
		return
	}

	if user.Status != "active" {
		http.Error(w, `{"status":"error","message":"บัญชีถูกระงับ"}`, http.StatusForbidden)
		return
	}

	profile := ProfileResponse{
		Status:   "success",
		Message:  "ดึงข้อมูลผู้ใช้สำเร็จ",
		ID:       user.ID,
		Username: user.Username,
		Email:    user.Email,
		Role:     user.Role,
	}

	json.NewEncoder(w).Encode(profile)
}

// PUT /api/me
func (h *ProfileHandler) UpdateMe(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	claims, ok := middleware.GetUser(r)
	if !ok {
		http.Error(w, `{"status":"error","message":"กรุณาเข้าสู่ระบบ"}`, http.StatusUnauthorized)
		return
	}

	var req UpdateProfileRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"status":"error","message":"ข้อมูลไม่ถูกต้อง"}`, http.StatusBadRequest)
		return
	}

	req.Username = strings.TrimSpace(req.Username)
	req.Email = strings.TrimSpace(req.Email)

	if req.Username == "" || req.Email == "" {
		http.Error(w, `{"status":"error","message":"กรุณากรอก Username และ Email"}`, http.StatusBadRequest)
		return
	}

	if !strings.Contains(req.Email, "@") {
		http.Error(w, `{"status":"error","message":"รูปแบบ Email ไม่ถูกต้อง"}`, http.StatusBadRequest)
		return
	}

	// ตรวจสอบ Username ซ้ำ
	usernameExists, err := store.UsernameExists(
		r.Context(),
		h.DB,
		req.Username,
		claims.UserID,
	)

	if err != nil {
		http.Error(w, `{"status":"error","message":"ตรวจสอบ Username ไม่สำเร็จ"}`, http.StatusInternalServerError)
		return
	}

	if usernameExists {
		http.Error(w, `{"status":"error","message":"Username นี้ถูกใช้งานแล้ว"}`, http.StatusConflict)
		return
	}

	// ตรวจสอบ Email ซ้ำ
	emailExists, err := store.EmailExists(
		r.Context(),
		h.DB,
		req.Email,
		claims.UserID,
	)

	if err != nil {
		http.Error(w, `{"status":"error","message":"ตรวจสอบ Email ไม่สำเร็จ"}`, http.StatusInternalServerError)
		return
	}

	if emailExists {
		http.Error(w, `{"status":"error","message":"Email นี้ถูกใช้งานแล้ว"}`, http.StatusConflict)
		return
	}

	// อัปเดตข้อมูลผู้ใช้
	user, err := store.UpdateUserProfile(
		r.Context(),
		h.DB,
		claims.UserID,
		req.Username,
		req.Email,
	)

	if err == sql.ErrNoRows {
		http.Error(w, `{"status":"error","message":"ไม่พบบัญชีผู้ใช้"}`, http.StatusNotFound)
		return
	}

	if err != nil {
		http.Error(w, `{"status":"error","message":"แก้ไขข้อมูลไม่สำเร็จ"}`, http.StatusInternalServerError)
		return
	}

	profile := ProfileResponse{
		Status:   "success",
		Message:  "แก้ไขข้อมูลโปรไฟล์สำเร็จ",
		ID:       user.ID,
		Username: user.Username,
		Email:    user.Email,
		Role:     user.Role,
	}

	json.NewEncoder(w).Encode(profile)
}
