package handlers

import (
	"database/sql"
	"net/http"
	"strconv"
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
