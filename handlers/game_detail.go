
package handlers

import (
	"database/sql"
	"log"
	"net/http"
	"strconv"

	"gamehub-backend/models"
)

type GameDetailHandler struct {
	DB *sql.DB
}

type GameDetailResponse struct {
	Game          models.Game      `json:"game"`
	RatingSummary RatingSummary    `json:"rating_summary"`
	Reviews       []ReviewResponse `json:"reviews"`
}

func (h *GameDetailHandler) GetGameDetail(w http.ResponseWriter, r *http.Request) {
	gameID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || gameID <= 0 {
		reviewError(w, http.StatusBadRequest, "รหัสเกมไม่ถูกต้อง")
		return
	}

	// 1. ดึงข้อมูลเกม
	var game models.Game

	var genre sql.NullString
	var year sql.NullInt64
	var image sql.NullString
	var description sql.NullString
	var igdbID sql.NullInt64

	err = h.DB.QueryRow(`
		SELECT id, name, genre, year, image, description, igdb_id
		FROM games
		WHERE id = $1 AND is_deleted = FALSE
	`, gameID).Scan(
		&game.ID,
		&game.Name,
		&genre,
		&year,
		&image,
		&description,
		&igdbID,
	)

	if err == sql.ErrNoRows {
		reviewError(w, http.StatusNotFound, "ไม่พบเกมนี้")
		return
	}

	if err != nil {
		log.Println("Game detail - game query:", err)
		reviewError(w, http.StatusInternalServerError, "ดึงข้อมูลเกมไม่สำเร็จ")
		return
	}

	if genre.Valid {
		game.Genre = &genre.String
	}
	if year.Valid {
		yearValue := int(year.Int64)
		game.Year = &yearValue
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

	// 2. ดึงคะแนนเฉลี่ยและจำนวนผู้ให้คะแนน
	var summary RatingSummary

	err = h.DB.QueryRow(`
		SELECT
			$1::bigint,
			COALESCE(ROUND(AVG(rating), 2), 0)::float8,
			COUNT(*)
		FROM game_ratings
		WHERE game_id = $1
	`, gameID).Scan(
		&summary.GameID,
		&summary.AverageRating,
		&summary.TotalRatings,
	)

	if err != nil {
		log.Println("Game detail - rating query:", err)
		reviewError(w, http.StatusInternalServerError, "ดึงคะแนนเกมไม่สำเร็จ")
		return
	}

	// 3. นับจำนวนรีวิวที่แสดงได้
	err = h.DB.QueryRow(`
		SELECT COUNT(*)
		FROM game_reviews
		WHERE game_id = $1 AND status = 'visible'
	`, gameID).Scan(&summary.TotalReviews)

	if err != nil {
		log.Println("Game detail - review count query:", err)
		reviewError(w, http.StatusInternalServerError, "นับจำนวนรีวิวไม่สำเร็จ")
		return
	}

	// 4. ดึงรายการรีวิว
	rows, err := h.DB.Query(`
		SELECT
			gr.id,
			gr.user_id,
			u.username,
			gr.game_id,
			gr.content,
			COALESCE(rt.rating, 0),
			gr.status,
			gr.created_at,
			gr.updated_at
		FROM game_reviews gr
		JOIN users u ON u.id = gr.user_id
		LEFT JOIN game_ratings rt
			ON rt.user_id = gr.user_id
			AND rt.game_id = gr.game_id
		WHERE gr.game_id = $1
			AND gr.status = 'visible'
		ORDER BY gr.created_at DESC
	`, gameID)

	if err != nil {
		log.Println("Game detail - reviews query:", err)
		reviewError(w, http.StatusInternalServerError, "ดึงรีวิวไม่สำเร็จ")
		return
	}
	defer rows.Close()

	reviews := []ReviewResponse{}

	for rows.Next() {
		var review ReviewResponse

		err := rows.Scan(
			&review.ID,
			&review.UserID,
			&review.Username,
			&review.GameID,
			&review.Content,
			&review.Rating,
			&review.Status,
			&review.CreatedAt,
			&review.UpdatedAt,
		)

		if err != nil {
			log.Println("Game detail - scan review:", err)
			reviewError(w, http.StatusInternalServerError, "อ่านข้อมูลรีวิวไม่สำเร็จ")
			return
		}

		reviews = append(reviews, review)
	}

	if err := rows.Err(); err != nil {
		log.Println("Game detail - rows:", err)
		reviewError(w, http.StatusInternalServerError, "อ่านข้อมูลรีวิวไม่สำเร็จ")
		return
	}

	// 5. ส่งข้อมูลทั้งหมดกลับไป
	response := GameDetailResponse{
		Game:          game,
		RatingSummary: summary,
		Reviews:       reviews,
	}

	reviewJSON(w, http.StatusOK, map[string]interface{}{
		"status": "success",
		"data":   response,
	})
}
