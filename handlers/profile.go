package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"

	"gamehub-backend/middleware"
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

	var profile ProfileResponse
	var accountStatus string

	err := h.DB.QueryRowContext(r.Context(),
		`SELECT id, username, email, role, status
		 FROM users
		 WHERE id = $1`,
		claims.UserID,
	).Scan(
		&profile.ID,
		&profile.Username,
		&profile.Email,
		&profile.Role,
		&accountStatus,
	)

	if err == sql.ErrNoRows {
		http.Error(w, `{"status":"error","message":"ไม่พบบัญชีผู้ใช้"}`, http.StatusUnauthorized)
		return
	}

	if err != nil {
		http.Error(w, `{"status":"error","message":"เกิดข้อผิดพลาดจากฐานข้อมูล"}`, http.StatusInternalServerError)
		return
	}

	if accountStatus != "active" {
		http.Error(w, `{"status":"error","message":"บัญชีถูกระงับ"}`, http.StatusForbidden)
		return
	}

	profile.Status = "success"
	profile.Message = "ดึงข้อมูลผู้ใช้สำเร็จ"

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
	var usernameExists bool

	err := h.DB.QueryRowContext(r.Context(),
		`SELECT EXISTS(
			SELECT 1 FROM users
			WHERE LOWER(TRIM(username)) = LOWER($1)
			AND id <> $2
		)`,
		req.Username,
		claims.UserID,
	).Scan(&usernameExists)

	if err != nil {
		http.Error(w, `{"status":"error","message":"ตรวจสอบ Username ไม่สำเร็จ"}`, http.StatusInternalServerError)
		return
	}

	if usernameExists {
		http.Error(w, `{"status":"error","message":"Username นี้ถูกใช้งานแล้ว"}`, http.StatusConflict)
		return
	}

	// ตรวจสอบ Email ซ้ำ
	var emailExists bool

	err = h.DB.QueryRowContext(r.Context(),
		`SELECT EXISTS(
			SELECT 1 FROM users
			WHERE LOWER(TRIM(email)) = LOWER($1)
			AND id <> $2
		)`,
		req.Email,
		claims.UserID,
	).Scan(&emailExists)

	if err != nil {
		http.Error(w, `{"status":"error","message":"ตรวจสอบ Email ไม่สำเร็จ"}`, http.StatusInternalServerError)
		return
	}

	if emailExists {
		http.Error(w, `{"status":"error","message":"Email นี้ถูกใช้งานแล้ว"}`, http.StatusConflict)
		return
	}

	// อัปเดตข้อมูลผู้ใช้
	var profile ProfileResponse

	err = h.DB.QueryRowContext(r.Context(),
		`UPDATE users
		 SET username = $1,
		     email = $2,
		     updated_at = NOW()
		 WHERE id = $3
		 RETURNING id, username, email, role`,
		req.Username,
		req.Email,
		claims.UserID,
	).Scan(
		&profile.ID,
		&profile.Username,
		&profile.Email,
		&profile.Role,
	)

	if err == sql.ErrNoRows {
		http.Error(w, `{"status":"error","message":"ไม่พบบัญชีผู้ใช้"}`, http.StatusNotFound)
		return
	}

	if err != nil {
		http.Error(w, `{"status":"error","message":"แก้ไขข้อมูลไม่สำเร็จ"}`, http.StatusInternalServerError)
		return
	}

	profile.Status = "success"
	profile.Message = "แก้ไขข้อมูลโปรไฟล์สำเร็จ"

	json.NewEncoder(w).Encode(profile)
}
