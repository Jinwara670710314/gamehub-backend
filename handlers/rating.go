package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"

	"gamehub-backend/middleware"
	"gamehub-backend/store"
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

	err = store.CheckActiveGame(r.Context(), h.DB, gameID)

	if err == sql.ErrNoRows {
		reviewError(w, http.StatusNotFound, "ไม่พบเกมนี้")
		return
	}

	if err != nil {
		reviewError(w, http.StatusInternalServerError, "ตรวจสอบข้อมูลเกมไม่สำเร็จ")
		return
	}

	err = store.UpsertRating(
		r.Context(),
		h.DB,
		claims.UserID,
		gameID,
		req.Rating,
	)

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

	result, err := store.GetRatingOverview(r.Context(), h.DB, gameID)

	if err == sql.ErrNoRows {
		reviewError(w, http.StatusNotFound, "ไม่พบเกมนี้")
		return
	}

	if err != nil {
		reviewError(w, http.StatusInternalServerError, "ดึงข้อมูลสรุปคะแนนไม่สำเร็จ")
		return
	}

	summary := RatingSummary{
		GameID:        result.GameID,
		AverageRating: result.AverageRating,
		TotalRatings:  result.TotalRatings,
		TotalReviews:  result.TotalReviews,
	}

	reviewJSON(w, http.StatusOK, map[string]interface{}{
		"status": "success",
		"data":   summary,
	})
}
