package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"gamehub-backend/models"
	"gamehub-backend/store"
)

type GameHandler struct {
	DB *sql.DB
}

// API ดึงรายการเกม ค้นหา และแบ่งหน้า
func (h *GameHandler) GetGames(w http.ResponseWriter, r *http.Request) {
	search := r.URL.Query().Get("q")

	limit := 20
	if value := r.URL.Query().Get("limit"); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil || parsed < 1 || parsed > 200 {
			http.Error(w, "limit ต้องอยู่ระหว่าง 1 ถึง 200", http.StatusBadRequest)
			return
		}
		limit = parsed
	}

	offset := 0
	if value := r.URL.Query().Get("offset"); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil || parsed < 0 {
			http.Error(w, "offset ต้องเป็นตัวเลขตั้งแต่ 0 ขึ้นไป", http.StatusBadRequest)
			return
		}
		offset = parsed
	}

	games, total, err := store.GetGames(
		r.Context(),
		h.DB,
		search,
		limit,
		offset,
	)

	if err != nil {
		http.Error(w, "ไม่สามารถดึงข้อมูลเกมได้", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "success",
		"total":  total,
		"count":  len(games),
		"limit":  limit,
		"offset": offset,
		"data":   games,
	})
}

// ใช้ตรวจสอบว่า Handler และ Database ทำงานได้
func (h *GameHandler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	if err := h.DB.PingContext(ctx); err != nil {
		log.Println("Database connection error:", err)
		http.Error(w, "Database connection failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":   "success",
		"message":  "GameHub Backend is running!",
		"database": "connected",
	})
}

// API ดูรายละเอียดเกมตาม ID
func (h *GameHandler) GetGameByID(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id < 1 {
		http.Error(w, "รหัสเกมไม่ถูกต้อง", http.StatusBadRequest)
		return
	}

	game, err := store.GetGameByID(r.Context(), h.DB, id)

	if err == sql.ErrNoRows {
		http.Error(w, "ไม่พบเกมที่ต้องการ", http.StatusNotFound)
		return
	}

	if err != nil {
		http.Error(w, "ไม่สามารถดึงรายละเอียดเกมได้", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "success",
		"data":   game,
	})
}

// API เพิ่มเกมใหม่
func (h *GameHandler) CreateGame(w http.ResponseWriter, r *http.Request) {
	var input models.Game

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "รูปแบบ JSON ไม่ถูกต้อง", http.StatusBadRequest)
		return
	}

	input.Name = strings.TrimSpace(input.Name)
	if input.Name == "" {
		http.Error(w, "กรุณาระบุชื่อเกม", http.StatusBadRequest)
		return
	}

	game, err := store.CreateGame(r.Context(), h.DB, input)

	if err != nil {
		log.Println("Create game error:", err)
		http.Error(w, "ไม่สามารถเพิ่มเกมได้", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "success",
		"message": "เพิ่มเกมสำเร็จ",
		"data":    game,
	})
}

// API แก้ไขข้อมูลเกม
func (h *GameHandler) UpdateGame(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id < 1 {
		http.Error(w, "รหัสเกมไม่ถูกต้อง", http.StatusBadRequest)
		return
	}

	var input models.Game

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "รูปแบบ JSON ไม่ถูกต้อง", http.StatusBadRequest)
		return
	}

	input.Name = strings.TrimSpace(input.Name)
	if input.Name == "" {
		http.Error(w, "กรุณาระบุชื่อเกม", http.StatusBadRequest)
		return
	}

	game, err := store.UpdateGame(r.Context(), h.DB, id, input)

	if err == sql.ErrNoRows {
		http.Error(w, "ไม่พบเกมที่ต้องการแก้ไข", http.StatusNotFound)
		return
	}

	if err != nil {
		log.Println("Update game error:", err)
		http.Error(w, "ไม่สามารถแก้ไขข้อมูลเกมได้", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "success",
		"message": "แก้ไขเกมสำเร็จ",
		"data":    game,
	})
}

// API ลบเกมแบบ Soft Delete
func (h *GameHandler) DeleteGame(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id < 1 {
		http.Error(w, "รหัสเกมไม่ถูกต้อง", http.StatusBadRequest)
		return
	}

	deleted, err := store.DeleteGame(r.Context(), h.DB, id)

	if err != nil {
		log.Println("Delete game error:", err)
		http.Error(w, "ไม่สามารถลบเกมได้", http.StatusInternalServerError)
		return
	}

	if !deleted {
		http.Error(w, "ไม่พบเกมที่ต้องการลบ", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "success",
		"message": "ลบเกมสำเร็จ",
		"id":      id,
	})
}

// API กู้คืนเกมที่ถูก Soft Delete
func (h *GameHandler) RestoreGame(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id < 1 {
		http.Error(w, "รหัสเกมไม่ถูกต้อง", http.StatusBadRequest)
		return
	}

	restored, err := store.RestoreGame(r.Context(), h.DB, id)

	if err != nil {
		log.Println("Restore game error:", err)
		http.Error(w, "ไม่สามารถกู้คืนเกมได้", http.StatusInternalServerError)
		return
	}

	if !restored {
		http.Error(w, "ไม่พบเกมที่ถูกลบหรือเกมนี้ยังไม่ได้ถูกลบ", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "success",
		"message": "กู้คืนเกมสำเร็จ",
		"id":      id,
	})
}
