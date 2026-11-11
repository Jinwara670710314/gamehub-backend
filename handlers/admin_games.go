
package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"gamehub-backend/models"
)

// ข้อมูลเกมสำหรับหน้า Admin
type AdminGame struct {
	models.Game
	IsDeleted bool `json:"isDeleted"`
}

// API สำหรับ Admin ดูเกมทั้งหมด รวมถึงเกมที่ถูกลบแล้ว
func (h *GameHandler) GetAdminGames(w http.ResponseWriter, r *http.Request) {
	rows, err := h.DB.QueryContext(r.Context(), `
		SELECT id, name, genre, year, image, description, igdb_id, is_deleted
		FROM public.games
		ORDER BY id ASC
	`)

	if err != nil {
		http.Error(w, "ไม่สามารถดึงข้อมูลเกมทั้งหมดได้", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	games := []AdminGame{}

	for rows.Next() {
		var game AdminGame
		var genre, image, description sql.NullString
		var year, igdbID sql.NullInt64

		err := rows.Scan(
			&game.ID,
			&game.Name,
			&genre,
			&year,
			&image,
			&description,
			&igdbID,
			&game.IsDeleted,
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

		if igdbID.Valid {
			game.IGDBID = &igdbID.Int64
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
