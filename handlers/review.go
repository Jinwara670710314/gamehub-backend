package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"gamehub-backend/middleware"
)

type ReviewHandler struct {
	DB *sql.DB
}

type ReviewRequest struct {
	Content string `json:"content"`
	Rating  int    `json:"rating"`
}

type ReviewResponse struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"user_id"`
	Username  string    `json:"username"`
	GameID    int64     `json:"game_id"`
	Content   string    `json:"content"`
	Rating    int       `json:"rating"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func reviewJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func reviewError(w http.ResponseWriter, status int, message string) {
	reviewJSON(w, status, map[string]interface{}{
		"status":  "error",
		"message": message,
	})
}

func validReviewRequest(content string, rating int) (string, bool) {
	content = strings.TrimSpace(content)

	length := utf8.RuneCountInString(content)

	if length < 1 || length > 5000 {
		return content, false
	}

	if rating < 1 || rating > 5 {
		return content, false
	}

	return content, true
}

// POST /api/games/{id}/reviews
// เขียนรีวิวและให้คะแนนเกม
func (h *ReviewHandler) CreateReview(w http.ResponseWriter, r *http.Request) {
	claims, _ := middleware.GetUser(r)

	gameID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || gameID <= 0 {
		reviewError(w, http.StatusBadRequest, "รหัสเกมไม่ถูกต้อง")
		return
	}

	var req ReviewRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		reviewError(w, http.StatusBadRequest, "ข้อมูลที่ส่งมาไม่ถูกต้อง")
		return
	}

	content, valid := validReviewRequest(req.Content, req.Rating)
	if !valid {
		reviewError(w, http.StatusBadRequest, "รีวิวต้องมีข้อความ 1-5000 ตัวอักษร และคะแนนต้องอยู่ระหว่าง 1-5")
		return
	}

	var existingGameID int64

	err = h.DB.QueryRow(`
		SELECT id
		FROM games
		WHERE id = $1 AND is_deleted = FALSE
	`, gameID).Scan(&existingGameID)

	if err == sql.ErrNoRows {
		reviewError(w, http.StatusNotFound, "ไม่พบเกมนี้")
		return
	}

	if err != nil {
		reviewError(w, http.StatusInternalServerError, "ตรวจสอบข้อมูลเกมไม่สำเร็จ")
		return
	}

	tx, err := h.DB.Begin()
	if err != nil {
		reviewError(w, http.StatusInternalServerError, "เริ่มบันทึกข้อมูลไม่สำเร็จ")
		return
	}
	defer tx.Rollback()

	var reviewID int64

	err = tx.QueryRow(`
		INSERT INTO game_reviews (user_id, game_id, content)
		VALUES ($1, $2, $3)
		RETURNING id
	`, claims.UserID, gameID, content).Scan(&reviewID)

	if err != nil {
		reviewError(w, http.StatusConflict, "คุณเคยรีวิวเกมนี้แล้ว หรือไม่สามารถบันทึกรีวิวได้")
		return
	}

	_, err = tx.Exec(`
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

	if err := tx.Commit(); err != nil {
		reviewError(w, http.StatusInternalServerError, "บันทึกรีวิวไม่สำเร็จ")
		return
	}

	reviewJSON(w, http.StatusCreated, map[string]interface{}{
		"status":    "success",
		"message":   "บันทึกรีวิวและคะแนนสำเร็จ",
		"review_id": reviewID,
		"game_id":   gameID,
		"rating":    req.Rating,
	})
}

// GET /api/games/{id}/reviews
// ดูรีวิวของเกม
func (h *ReviewHandler) GetReviews(w http.ResponseWriter, r *http.Request) {
	gameID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || gameID <= 0 {
		reviewError(w, http.StatusBadRequest, "รหัสเกมไม่ถูกต้อง")
		return
	}

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
			reviewError(w, http.StatusInternalServerError, "อ่านข้อมูลรีวิวไม่สำเร็จ")
			return
		}

		reviews = append(reviews, review)
	}

	if err := rows.Err(); err != nil {
		reviewError(w, http.StatusInternalServerError, "อ่านข้อมูลรีวิวไม่สำเร็จ")
		return
	}

	reviewJSON(w, http.StatusOK, map[string]interface{}{
		"status": "success",
		"count":  len(reviews),
		"data":   reviews,
	})
}

// PUT /api/reviews/{id}
// แก้ไขรีวิวและคะแนนของตัวเอง
func (h *ReviewHandler) UpdateReview(w http.ResponseWriter, r *http.Request) {
	claims, _ := middleware.GetUser(r)

	reviewID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || reviewID <= 0 {
		reviewError(w, http.StatusBadRequest, "รหัสรีวิวไม่ถูกต้อง")
		return
	}

	var req ReviewRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		reviewError(w, http.StatusBadRequest, "ข้อมูลที่ส่งมาไม่ถูกต้อง")
		return
	}

	content, valid := validReviewRequest(req.Content, req.Rating)
	if !valid {
		reviewError(w, http.StatusBadRequest, "รีวิวต้องมีข้อความ 1-5000 ตัวอักษร และคะแนนต้องอยู่ระหว่าง 1-5")
		return
	}

	tx, err := h.DB.Begin()
	if err != nil {
		reviewError(w, http.StatusInternalServerError, "เริ่มแก้ไขข้อมูลไม่สำเร็จ")
		return
	}
	defer tx.Rollback()

	var gameID int64

	err = tx.QueryRow(`
		UPDATE game_reviews
		SET content = $1, updated_at = NOW()
		WHERE id = $2 AND user_id = $3
		RETURNING game_id
	`, content, reviewID, claims.UserID).Scan(&gameID)

	if err == sql.ErrNoRows {
		reviewError(w, http.StatusNotFound, "ไม่พบรีวิวของคุณ")
		return
	}

	if err != nil {
		reviewError(w, http.StatusInternalServerError, "แก้ไขรีวิวไม่สำเร็จ")
		return
	}

	_, err = tx.Exec(`
		INSERT INTO game_ratings (user_id, game_id, rating)
		VALUES ($1, $2, $3)
		ON CONFLICT (user_id, game_id)
		DO UPDATE SET
			rating = EXCLUDED.rating,
			updated_at = NOW()
	`, claims.UserID, gameID, req.Rating)

	if err != nil {
		reviewError(w, http.StatusInternalServerError, "แก้ไขคะแนนไม่สำเร็จ")
		return
	}

	if err := tx.Commit(); err != nil {
		reviewError(w, http.StatusInternalServerError, "บันทึกการแก้ไขไม่สำเร็จ")
		return
	}

	reviewJSON(w, http.StatusOK, map[string]interface{}{
		"status":    "success",
		"message":   "แก้ไขรีวิวและคะแนนสำเร็จ",
		"review_id": reviewID,
		"game_id":   gameID,
		"rating":    req.Rating,
	})
}

// DELETE /api/reviews/{id}
// ลบรีวิวและคะแนนของตัวเอง
func (h *ReviewHandler) DeleteReview(w http.ResponseWriter, r *http.Request) {
	claims, _ := middleware.GetUser(r)

	reviewID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || reviewID <= 0 {
		reviewError(w, http.StatusBadRequest, "รหัสรีวิวไม่ถูกต้อง")
		return
	}

	tx, err := h.DB.Begin()
	if err != nil {
		reviewError(w, http.StatusInternalServerError, "เริ่มลบข้อมูลไม่สำเร็จ")
		return
	}
	defer tx.Rollback()

	var gameID int64

	err = tx.QueryRow(`
		DELETE FROM game_reviews
		WHERE id = $1 AND user_id = $2
		RETURNING game_id
	`, reviewID, claims.UserID).Scan(&gameID)

	if err == sql.ErrNoRows {
		reviewError(w, http.StatusNotFound, "ไม่พบรีวิวของคุณ")
		return
	}

	if err != nil {
		reviewError(w, http.StatusInternalServerError, "ลบรีวิวไม่สำเร็จ")
		return
	}

	_, err = tx.Exec(`
		DELETE FROM game_ratings
		WHERE user_id = $1 AND game_id = $2
	`, claims.UserID, gameID)

	if err != nil {
		reviewError(w, http.StatusInternalServerError, "ลบคะแนนไม่สำเร็จ")
		return
	}

	if err := tx.Commit(); err != nil {
		reviewError(w, http.StatusInternalServerError, "บันทึกการลบไม่สำเร็จ")
		return
	}

	reviewJSON(w, http.StatusOK, map[string]interface{}{
		"status":    "success",
		"message":   "ลบรีวิวและคะแนนสำเร็จ",
		"review_id": reviewID,
	})
}
