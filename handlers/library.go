package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"gamehub-backend/middleware"
	"gamehub-backend/store"
)

type LibraryHandler struct {
	DB *sql.DB
}

type AddLibraryRequest struct {
	GameID int64  `json:"game_id"`
	Status string `json:"status"`
}

type UpdateLibraryRequest struct {
	Status string `json:"status"`
}

// ตรวจสอบสถานะเกม
func validLibraryStatus(status string) bool {
	switch status {
	case "want_to_play", "playing", "completed", "dropped":
		return true
	default:
		return false
	}
}

// POST /api/library
// เพิ่มเกมเข้าคลังส่วนตัว
func (h *LibraryHandler) AddGame(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	claims, ok := middleware.GetUser(r)
	if !ok {
		http.Error(w, `{"status":"error","message":"กรุณาเข้าสู่ระบบ"}`, http.StatusUnauthorized)
		return
	}

	var req AddLibraryRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"status":"error","message":"ข้อมูลไม่ถูกต้อง"}`, http.StatusBadRequest)
		return
	}

	if req.GameID <= 0 {
		http.Error(w, `{"status":"error","message":"Game ID ไม่ถูกต้อง"}`, http.StatusBadRequest)
		return
	}

	if req.Status == "" {
		req.Status = "want_to_play"
	}

	if !validLibraryStatus(req.Status) {
		http.Error(w, `{"status":"error","message":"สถานะเกมไม่ถูกต้อง"}`, http.StatusBadRequest)
		return
	}

	libraryID, err := store.AddLibraryGame(
		r.Context(),
		h.DB,
		claims.UserID,
		req.GameID,
		req.Status,
	)

	if err != nil {
		if strings.Contains(err.Error(), "user_game_library_user_id_game_id_key") {
			http.Error(w, `{"status":"error","message":"เกมนี้อยู่ในคลังของคุณแล้ว"}`, http.StatusConflict)
			return
		}

		if strings.Contains(err.Error(), "user_game_library_game_id_fkey") {
			http.Error(w, `{"status":"error","message":"ไม่พบเกมนี้ในระบบ"}`, http.StatusNotFound)
			return
		}

		http.Error(w, `{"status":"error","message":"เพิ่มเกมเข้าคลังไม่สำเร็จ"}`, http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":      "success",
		"message":     "เพิ่มเกมเข้าคลังสำเร็จ",
		"library_id":  libraryID,
		"game_id":     req.GameID,
		"game_status": req.Status,
	})
}

// GET /api/library
// ดูรายการเกมในคลังส่วนตัว
func (h *LibraryHandler) GetLibrary(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	claims, ok := middleware.GetUser(r)
	if !ok {
		http.Error(w, `{"status":"error","message":"กรุณาเข้าสู่ระบบ"}`, http.StatusUnauthorized)
		return
	}

	games, err := store.GetLibraryGames(
		r.Context(),
		h.DB,
		claims.UserID,
	)

	if err != nil {
		http.Error(w, `{"status":"error","message":"ดึงข้อมูลคลังเกมไม่สำเร็จ"}`, http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "success",
		"count":  len(games),
		"data":   games,
	})
}

// PUT /api/library/{id}
// เปลี่ยนสถานะเกมในคลัง
func (h *LibraryHandler) UpdateGameStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	claims, ok := middleware.GetUser(r)
	if !ok {
		http.Error(w, `{"status":"error","message":"กรุณาเข้าสู่ระบบ"}`, http.StatusUnauthorized)
		return
	}

	libraryID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || libraryID <= 0 {
		http.Error(w, `{"status":"error","message":"Library ID ไม่ถูกต้อง"}`, http.StatusBadRequest)
		return
	}

	var req UpdateLibraryRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"status":"error","message":"ข้อมูลไม่ถูกต้อง"}`, http.StatusBadRequest)
		return
	}

	if !validLibraryStatus(req.Status) {
		http.Error(w, `{"status":"error","message":"สถานะเกมไม่ถูกต้อง"}`, http.StatusBadRequest)
		return
	}

	updated, err := store.UpdateLibraryGameStatus(
		r.Context(),
		h.DB,
		libraryID,
		claims.UserID,
		req.Status,
	)

	if err != nil {
		http.Error(w, `{"status":"error","message":"เปลี่ยนสถานะเกมไม่สำเร็จ"}`, http.StatusInternalServerError)
		return
	}

	if !updated {
		http.Error(w, `{"status":"error","message":"ไม่พบเกมในคลังของคุณ"}`, http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":      "success",
		"message":     "เปลี่ยนสถานะเกมสำเร็จ",
		"id":          libraryID,
		"game_status": req.Status,
	})
}

// DELETE /api/library/{id}
// นำเกมออกจากคลังส่วนตัว
func (h *LibraryHandler) RemoveGame(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	claims, ok := middleware.GetUser(r)
	if !ok {
		http.Error(w, `{"status":"error","message":"กรุณาเข้าสู่ระบบ"}`, http.StatusUnauthorized)
		return
	}

	libraryID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || libraryID <= 0 {
		http.Error(w, `{"status":"error","message":"Library ID ไม่ถูกต้อง"}`, http.StatusBadRequest)
		return
	}

	removed, err := store.RemoveLibraryGame(
		r.Context(),
		h.DB,
		libraryID,
		claims.UserID,
	)

	if err != nil {
		http.Error(w, `{"status":"error","message":"นำเกมออกจากคลังไม่สำเร็จ"}`, http.StatusInternalServerError)
		return
	}

	if !removed {
		http.Error(w, `{"status":"error","message":"ไม่พบเกมในคลังของคุณ"}`, http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "success",
		"message": "นำเกมออกจากคลังสำเร็จ",
		"id":      libraryID,
	})
}
