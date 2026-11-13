package handlers

import (
	"encoding/json"
	"net/http"

	"gamehub-backend/store"
)

// API สำหรับ Admin ดูเกมทั้งหมด รวมถึงเกมที่ถูกลบแล้ว
func (h *GameHandler) GetAdminGames(w http.ResponseWriter, r *http.Request) {
	games, err := store.GetAdminGames(r.Context(), h.DB)

	if err != nil {
		http.Error(w, "ไม่สามารถดึงข้อมูลเกมทั้งหมดได้", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "success",
		"count":  len(games),
		"data":   games,
	})
}
