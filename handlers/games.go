package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"gamehub-backend/models"
)

type GameHandler struct {
	DB *sql.DB
}

// API ดึงรายการเกมทั้งหมด และค้นหาเกม
func (h *GameHandler) GetGames(w http.ResponseWriter, r *http.Request) {
	// รับคำค้นหาจาก URL
	search := r.URL.Query().Get("q")

	rows, err := h.DB.QueryContext(r.Context(), `
		SELECT id, name, genre, year, image, description, rawg_id
		FROM public.games
		WHERE is_deleted = FALSE
		AND ($1 = '' OR name ILIKE '%' || $1 || '%')
		ORDER BY id ASC
	`, search)

	if err != nil {
		http.Error(w, "ไม่สามารถดึงข้อมูลเกมได้", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	games := []models.Game{}

	for rows.Next() {
		var game models.Game
		var genre, image, description sql.NullString
		var year, rawgID sql.NullInt64

		err := rows.Scan(
			&game.ID,
			&game.Name,
			&genre,
			&year,
			&image,
			&description,
			&rawgID,
		)
		if err != nil {
			http.Error(w, "เกิดข้อผิดพลาดในการอ่านข้อมูลเกม", http.StatusInternalServerError)
			return
		}

		if genre.Valid {
			game.Genre = &genre.String
		}
		if year.Valid {
			value := int(year.Int64)
			game.Year = &value
		}
		if image.Valid {
			game.Image = &image.String
		}
		if description.Valid {
			game.Description = &description.String
		}
		if rawgID.Valid {
			game.RAWGID = &rawgID.Int64
		}

		games = append(games, game)
	}

	if err := rows.Err(); err != nil {
		http.Error(w, "เกิดข้อผิดพลาดในการดึงข้อมูลเกม", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "success",
		"count":  len(games),
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
	if err != nil {
		http.Error(w, "รหัสเกมไม่ถูกต้อง", http.StatusBadRequest)
		return
	}

	var game models.Game
	var genre, image, description sql.NullString
	var year, rawgID sql.NullInt64

	err = h.DB.QueryRowContext(r.Context(), `
		SELECT id, name, genre, year, image, description, rawg_id
		FROM public.games
		WHERE id = $1 AND is_deleted = FALSE
	`, id).Scan(
		&game.ID,
		&game.Name,
		&genre,
		&year,
		&image,
		&description,
		&rawgID,
	)

	if err == sql.ErrNoRows {
		http.Error(w, "ไม่พบเกมที่ต้องการ", http.StatusNotFound)
		return
	}

	if err != nil {
		http.Error(w, "ไม่สามารถดึงรายละเอียดเกมได้", http.StatusInternalServerError)
		return
	}

	if genre.Valid {
		game.Genre = &genre.String
	}
	if year.Valid {
		value := int(year.Int64)
		game.Year = &value
	}
	if image.Valid {
		game.Image = &image.String
	}
	if description.Valid {
		game.Description = &description.String
	}
	if rawgID.Valid {
		game.RAWGID = &rawgID.Int64
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "success",
		"data":   game,
	})
}
