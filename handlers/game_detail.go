package handlers

import (
	"database/sql"
	"log"
	"net/http"
	"strconv"

	"gamehub-backend/models"
	"gamehub-backend/store"
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
	game, err := store.GetGameDetail(r.Context(), h.DB, gameID)

	if err == sql.ErrNoRows {
		reviewError(w, http.StatusNotFound, "ไม่พบเกมนี้")
		return
	}

	if err != nil {
		log.Println("Game detail - game query:", err)
		reviewError(w, http.StatusInternalServerError, "ดึงข้อมูลเกมไม่สำเร็จ")
		return
	}

	// 2. ดึงคะแนนเฉลี่ยและจำนวนผู้ให้คะแนน
	storeSummary, err := store.GetGameRatingSummary(
		r.Context(),
		h.DB,
		gameID,
	)

	if err != nil {
		log.Println("Game detail - rating summary query:", err)
		reviewError(w, http.StatusInternalServerError, "ดึงคะแนนเกมไม่สำเร็จ")
		return
	}

	summary := RatingSummary{
		GameID:        storeSummary.GameID,
		AverageRating: storeSummary.AverageRating,
		TotalRatings:  storeSummary.TotalRatings,
		TotalReviews:  storeSummary.TotalReviews,
	}

	// 3. ดึงรายการรีวิว
	storeReviews, err := store.GetGameReviews(
		r.Context(),
		h.DB,
		gameID,
	)

	if err != nil {
		log.Println("Game detail - reviews query:", err)
		reviewError(w, http.StatusInternalServerError, "ดึงรีวิวไม่สำเร็จ")
		return
	}

	reviews := []ReviewResponse{}

	for _, item := range storeReviews {
		if !item.CreatedAt.Valid || !item.UpdatedAt.Valid {
			log.Println("Game detail - invalid review timestamp")
			reviewError(w, http.StatusInternalServerError, "อ่านข้อมูลรีวิวไม่สำเร็จ")
			return
		}

		review := ReviewResponse{
			ID:        item.ID,
			UserID:    item.UserID,
			Username:  item.Username,
			GameID:    item.GameID,
			Content:   item.Content,
			Rating:    item.Rating,
			Status:    item.Status,
			CreatedAt: item.CreatedAt.Time,
			UpdatedAt: item.UpdatedAt.Time,
		}

		reviews = append(reviews, review)
	}

	// 4. ส่งข้อมูลทั้งหมดกลับไป
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
