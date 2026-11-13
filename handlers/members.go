package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"gamehub-backend/middleware"
	"gamehub-backend/store"
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
	members, err := store.GetMembers(r.Context(), h.DB)

	if err != nil {
		http.Error(w, "ไม่สามารถดึงข้อมูลสมาชิกได้", http.StatusInternalServerError)
		return
	}

	responseMembers := make([]Member, 0, len(members))

	for _, member := range members {
		responseMembers = append(responseMembers, Member{
			ID:       member.ID,
			Username: member.Username,
			Email:    member.Email,
			Role:     member.Role,
			Status:   member.Status,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "success",
		"count":  len(responseMembers),
		"data":   responseMembers,
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

	updatedMember, err := store.UpdateMemberStatus(
		r.Context(),
		h.DB,
		id,
		req.Status,
	)

	if errors.Is(err, sql.ErrNoRows) {
		role, checkErr := store.GetMemberRole(r.Context(), h.DB, id)

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

	member := Member{
		ID:       updatedMember.ID,
		Username: updatedMember.Username,
		Email:    updatedMember.Email,
		Role:     updatedMember.Role,
		Status:   updatedMember.Status,
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
