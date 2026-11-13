package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"gamehub-backend/middleware"
	"gamehub-backend/store"
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

	err = store.CheckReviewGame(r.Context(), h.DB, gameID)

	if errors.Is(err, sql.ErrNoRows) {
		reviewError(w, http.StatusNotFound, "ไม่พบเกมนี้")
		return
	}

	if err != nil {
		reviewError(w, http.StatusInternalServerError, "ตรวจสอบข้อมูลเกมไม่สำเร็จ")
		return
	}

	reviewID, err := store.CreateReview(
		r.Context(),
		h.DB,
		claims.UserID,
		gameID,
		content,
		req.Rating,
	)

	if store.IsReviewStage(err, "begin") {
		reviewError(w, http.StatusInternalServerError, "เริ่มบันทึกข้อมูลไม่สำเร็จ")
		return
	}

	if store.IsReviewStage(err, "insert") {
		reviewError(w, http.StatusConflict, "คุณเคยรีวิวเกมนี้แล้ว หรือไม่สามารถบันทึกรีวิวได้")
		return
	}

	if store.IsReviewStage(err, "rating") {
		reviewError(w, http.StatusInternalServerError, "บันทึกคะแนนไม่สำเร็จ")
		return
	}

	if store.IsReviewStage(err, "commit") {
		reviewError(w, http.StatusInternalServerError, "บันทึกรีวิวไม่สำเร็จ")
		return
	}

	if err != nil {
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

	results, err := store.GetReviews(r.Context(), h.DB, gameID)

	if store.IsReviewStage(err, "query") {
		reviewError(w, http.StatusInternalServerError, "ดึงรีวิวไม่สำเร็จ")
		return
	}

	if err != nil {
		reviewError(w, http.StatusInternalServerError, "อ่านข้อมูลรีวิวไม่สำเร็จ")
		return
	}

	reviews := make([]ReviewResponse, 0, len(results))

	for _, review := range results {
		reviews = append(reviews, ReviewResponse{
			ID:        review.ID,
			UserID:    review.UserID,
			Username:  review.Username,
			GameID:    review.GameID,
			Content:   review.Content,
			Rating:    review.Rating,
			Status:    review.Status,
			CreatedAt: review.CreatedAt,
			UpdatedAt: review.UpdatedAt,
		})
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

	gameID, err := store.UpdateReview(
		r.Context(),
		h.DB,
		reviewID,
		claims.UserID,
		content,
		req.Rating,
	)

	if errors.Is(err, sql.ErrNoRows) {
		reviewError(w, http.StatusNotFound, "ไม่พบรีวิวของคุณ")
		return
	}

	if store.IsReviewStage(err, "begin") {
		reviewError(w, http.StatusInternalServerError, "เริ่มแก้ไขข้อมูลไม่สำเร็จ")
		return
	}

	if store.IsReviewStage(err, "update") {
		reviewError(w, http.StatusInternalServerError, "แก้ไขรีวิวไม่สำเร็จ")
		return
	}

	if store.IsReviewStage(err, "rating") {
		reviewError(w, http.StatusInternalServerError, "แก้ไขคะแนนไม่สำเร็จ")
		return
	}

	if store.IsReviewStage(err, "commit") {
		reviewError(w, http.StatusInternalServerError, "บันทึกการแก้ไขไม่สำเร็จ")
		return
	}

	if err != nil {
		reviewError(w, http.StatusInternalServerError, "แก้ไขรีวิวไม่สำเร็จ")
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

	err = store.DeleteReview(
		r.Context(),
		h.DB,
		reviewID,
		claims.UserID,
	)

	if errors.Is(err, sql.ErrNoRows) {
		reviewError(w, http.StatusNotFound, "ไม่พบรีวิวของคุณ")
		return
	}

	if store.IsReviewStage(err, "begin") {
		reviewError(w, http.StatusInternalServerError, "เริ่มลบข้อมูลไม่สำเร็จ")
		return
	}

	if store.IsReviewStage(err, "delete") {
		reviewError(w, http.StatusInternalServerError, "ลบรีวิวไม่สำเร็จ")
		return
	}

	if store.IsReviewStage(err, "rating") {
		reviewError(w, http.StatusInternalServerError, "ลบคะแนนไม่สำเร็จ")
		return
	}

	if store.IsReviewStage(err, "commit") {
		reviewError(w, http.StatusInternalServerError, "บันทึกการลบไม่สำเร็จ")
		return
	}

	if err != nil {
		reviewError(w, http.StatusInternalServerError, "ลบรีวิวไม่สำเร็จ")
		return
	}

	reviewJSON(w, http.StatusOK, map[string]interface{}{
		"status":    "success",
		"message":   "ลบรีวิวและคะแนนสำเร็จ",
		"review_id": reviewID,
	})
}
