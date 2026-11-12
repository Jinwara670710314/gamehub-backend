package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"

	"gamehub-backend/middleware"
)

type RatingHandler struct {
	DB *sql.DB
}

type RatingSummary struct {
	GameID        int64   `json:"game_id"`
	AverageRating float64 `json:"average_rating"`
	TotalRatings  int64   `json:"total_ratings"`
	TotalReviews  int64   `json:"total_reviews"`
}

type RatingRequest struct {
	Rating int `json:"rating"`
}

// POST /api/games/{id}/rating
// ให้คะแนนเกมหรือแก้ไขคะแนนเดิม
func (h *RatingHandler) SubmitRating(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetUser(r)
	if !ok {
		reviewError(w, http.StatusUnauthorized, "กรุณาเข้าสู่ระบบก่อน")
		return
	}

	gameID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || gameID <= 0 {
		reviewError(w, http.StatusBadRequest, "รหัสเกมไม่ถูกต้อง")
		return
	}

	var req RatingRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		reviewError(w, http.StatusBadRequest, "ข้อมูลที่ส่งมาไม่ถูกต้อง")
		return
	}

	if req.Rating < 1 || req.Rating > 5 {
		reviewError(w, http.StatusBadRequest, "คะแนนต้องอยู่ระหว่าง 1-5")
		return
	}

	var existingGameID int64

	err = h.DB.QueryRow(`
		SELECT id
		FROM games
		WHERE id = $1
		AND is_deleted = FALSE
	`, gameID).Scan(&existingGameID)

	if err == sql.ErrNoRows {
		reviewError(w, http.StatusNotFound, "ไม่พบเกมนี้")
		return
	}

	if err != nil {
		reviewError(w, http.StatusInternalServerError, "ตรวจสอบข้อมูลเกมไม่สำเร็จ")
		return
	}

	_, err = h.DB.Exec(`
		INSERT INTO game_ratings (user_id, game_id, rating)
		VALUES ($1, $2, $3)
		ON CONFLICT (user_id, game_id)
		DO UPDATE SET
			rating = EXCLUDED.rating,
			updated_at = NOW()
	`, claims.UserID, gameID, req.Rating)

	if err != nil {
		reviewError(w, http.StatusInternalServerError, "บันทึกคะแนนไม่สำเร็จ")
		return
	}

	reviewJSON(w, http.StatusOK, map[string]interface{}{
		"status":  "success",
		"message": "บันทึกคะแนนสำเร็จ",
		"game_id": gameID,
		"rating":  req.Rating,
	})
}

// GET /api/games/{id}/rating-summary
// สรุปคะแนนและจำนวนรีวิวของเกม
func (h *RatingHandler) GetRatingSummary(w http.ResponseWriter, r *http.Request) {
	gameID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || gameID <= 0 {
		reviewError(w, http.StatusBadRequest, "รหัสเกมไม่ถูกต้อง")
		return
	}

	var summary RatingSummary

	err = h.DB.QueryRow(`
		SELECT
			g.id,

			COALESCE(
				(
					SELECT ROUND(AVG(rating)::numeric, 2)
					FROM game_ratings
					WHERE game_id = g.id
				),
				0
			)::float8,

			(
				SELECT COUNT(*)
				FROM game_ratings
				WHERE game_id = g.id
			),

			(
				SELECT COUNT(*)
				FROM game_reviews
				WHERE game_id = g.id
				AND status = 'visible'
			)

		FROM games g
		WHERE g.id = $1
		AND g.is_deleted = FALSE
	`, gameID).Scan(
		&summary.GameID,
		&summary.AverageRating,
		&summary.TotalRatings,
		&summary.TotalReviews,
	)

	if err == sql.ErrNoRows {
		reviewError(w, http.StatusNotFound, "ไม่พบเกมนี้")
		return
	}

	if err != nil {
		reviewError(w, http.StatusInternalServerError, "ดึงข้อมูลสรุปคะแนนไม่สำเร็จ")
		return
	}

	reviewJSON(w, http.StatusOK, map[string]interface{}{
		"status": "success",
		"data":   summary,
	})
}
