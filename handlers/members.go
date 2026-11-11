
package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"gamehub-backend/middleware"
)

type MemberHandler struct {
	DB *sql.DB
}

type Member struct {
	ID       int64  `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Role     string `json:"role"`
	Status   string `json:"status"`
}

type UpdateMemberStatusRequest struct {
	Status string `json:"status"`
}

// API สำหรับ Admin ดูรายชื่อสมาชิกทั้งหมด
func (h *MemberHandler) GetMembers(w http.ResponseWriter, r *http.Request) {
	rows, err := h.DB.QueryContext(r.Context(), `
		SELECT id, username, email, role, status
		FROM public.users
		ORDER BY id ASC
	`)

	if err != nil {
		http.Error(w, "ไม่สามารถดึงข้อมูลสมาชิกได้", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	members := []Member{}

	for rows.Next() {
		var member Member

		if err := rows.Scan(
			&member.ID,
			&member.Username,
			&member.Email,
			&member.Role,
			&member.Status,
		); err != nil {
			http.Error(w, "เกิดข้อผิดพลาดในการอ่านข้อมูลสมาชิก", http.StatusInternalServerError)
			return
		}

		members = append(members, member)
	}

	if err := rows.Err(); err != nil {
		http.Error(w, "เกิดข้อผิดพลาดในการดึงข้อมูลสมาชิก", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "success",
		"count":  len(members),
		"data":   members,
	})
}

// API สำหรับ Admin ระงับหรือเปิดใช้งานบัญชีสมาชิก
func (h *MemberHandler) UpdateMemberStatus(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetUser(r)
	if !ok {
		http.Error(w, "กรุณาเข้าสู่ระบบก่อน", http.StatusUnauthorized)
		return
	}

	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id < 1 {
		http.Error(w, "รหัสสมาชิกไม่ถูกต้อง", http.StatusBadRequest)
		return
	}

	var req UpdateMemberStatusRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "รูปแบบ JSON ไม่ถูกต้อง", http.StatusBadRequest)
		return
	}

	req.Status = strings.ToLower(strings.TrimSpace(req.Status))

	if req.Status != "active" && req.Status != "suspended" {
		http.Error(w, "สถานะต้องเป็น active หรือ suspended เท่านั้น", http.StatusBadRequest)
		return
	}

	if claims.UserID == id && req.Status == "suspended" {
		http.Error(w, "ไม่สามารถระงับบัญชีของตนเองได้", http.StatusForbidden)
		return
	}

	var member Member

	err = h.DB.QueryRowContext(r.Context(), `
		UPDATE public.users
		SET status = $1,
		    updated_at = NOW()
		WHERE id = $2
		  AND role = 'member'
		RETURNING id, username, email, role, status
	`, req.Status, id).Scan(
		&member.ID,
		&member.Username,
		&member.Email,
		&member.Role,
		&member.Status,
	)

	if errors.Is(err, sql.ErrNoRows) {
		var role string

		checkErr := h.DB.QueryRowContext(r.Context(), `
			SELECT role
			FROM public.users
			WHERE id = $1
		`, id).Scan(&role)

		if errors.Is(checkErr, sql.ErrNoRows) {
			http.Error(w, "ไม่พบสมาชิกที่ต้องการ", http.StatusNotFound)
			return
		}

		if checkErr != nil {
			http.Error(w, "ไม่สามารถตรวจสอบข้อมูลสมาชิกได้", http.StatusInternalServerError)
			return
		}

		if role != "member" {
			http.Error(w, "ไม่อนุญาตให้เปลี่ยนสถานะบัญชี Admin", http.StatusForbidden)
			return
		}

		http.Error(w, "ไม่สามารถเปลี่ยนสถานะสมาชิกได้", http.StatusInternalServerError)
		return
	}

	if err != nil {
		http.Error(w, "ไม่สามารถเปลี่ยนสถานะสมาชิกได้", http.StatusInternalServerError)
		return
	}

	message := "เปิดใช้งานบัญชีสมาชิกสำเร็จ"
	if member.Status == "suspended" {
		message = "ระงับบัญชีสมาชิกสำเร็จ"
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "success",
		"message": message,
		"data":    member,
	})
}
