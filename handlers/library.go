package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"gamehub-backend/middleware"
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

type LibraryGame struct {
	ID          int64   `json:"id"`
	GameID      int64   `json:"game_id"`
	Name        string  `json:"name"`
	Genre       *string `json:"genre"`
	Year        *int    `json:"year"`
	Image       *string `json:"image"`
	Description *string `json:"description"`
	Status      string  `json:"status"`
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

	var libraryID int64

	err := h.DB.QueryRowContext(r.Context(),
		`INSERT INTO user_game_library (user_id, game_id, status)
		 VALUES ($1, $2, $3)
		 RETURNING id`,
		claims.UserID,
		req.GameID,
		req.Status,
	).Scan(&libraryID)

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

	rows, err := h.DB.QueryContext(r.Context(),
		`SELECT
			ugl.id,
			g.id,
			g.name,
			g.genre,
			g.year,
			g.image,
			g.description,
			ugl.status
		FROM user_game_library ugl
		JOIN games g ON g.id = ugl.game_id
		WHERE ugl.user_id = $1
		  AND g.is_deleted = FALSE
		ORDER BY ugl.created_at DESC`,
		claims.UserID,
	)

	if err != nil {
		http.Error(w, `{"status":"error","message":"ดึงข้อมูลคลังเกมไม่สำเร็จ"}`, http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	games := []LibraryGame{}

	for rows.Next() {
		var game LibraryGame

		err := rows.Scan(
			&game.ID,
			&game.GameID,
			&game.Name,
			&game.Genre,
			&game.Year,
			&game.Image,
			&game.Description,
			&game.Status,
		)

		if err != nil {
			http.Error(w, `{"status":"error","message":"อ่านข้อมูลเกมไม่สำเร็จ"}`, http.StatusInternalServerError)
			return
		}

		games = append(games, game)
	}

	if err := rows.Err(); err != nil {
		http.Error(w, `{"status":"error","message":"อ่านข้อมูลคลังเกมไม่สำเร็จ"}`, http.StatusInternalServerError)
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

	result, err := h.DB.ExecContext(r.Context(),
		`UPDATE user_game_library
		 SET status = $1, updated_at = NOW()
		 WHERE id = $2 AND user_id = $3`,
		req.Status,
		libraryID,
		claims.UserID,
	)

	if err != nil {
		http.Error(w, `{"status":"error","message":"เปลี่ยนสถานะเกมไม่สำเร็จ"}`, http.StatusInternalServerError)
		return
	}

	affected, err := result.RowsAffected()
	if err != nil {
		http.Error(w, `{"status":"error","message":"ตรวจสอบผลการแก้ไขไม่สำเร็จ"}`, http.StatusInternalServerError)
		return
	}

	if affected == 0 {
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

	result, err := h.DB.ExecContext(r.Context(),
		`DELETE FROM user_game_library
		 WHERE id = $1 AND user_id = $2`,
		libraryID,
		claims.UserID,
	)

	if err != nil {
		http.Error(w, `{"status":"error","message":"นำเกมออกจากคลังไม่สำเร็จ"}`, http.StatusInternalServerError)
		return
	}

	affected, err := result.RowsAffected()
	if err != nil {
		http.Error(w, `{"status":"error","message":"ตรวจสอบผลการลบไม่สำเร็จ"}`, http.StatusInternalServerError)
		return
	}

	if affected == 0 {
		http.Error(w, `{"status":"error","message":"ไม่พบเกมในคลังของคุณ"}`, http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "success",
		"message": "นำเกมออกจากคลังสำเร็จ",
		"id":      libraryID,
	})
}
