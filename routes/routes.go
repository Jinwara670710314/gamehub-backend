package routes

import (
	"database/sql"
	"net/http"

	"gamehub-backend/handlers"
)

func Setup(db *sql.DB) http.Handler {
	mux := http.NewServeMux()

	gameHandler := &handlers.GameHandler{
		DB: db,
	}

	// API ตรวจสอบ Backend และ Database
	mux.HandleFunc("/api/health", gameHandler.HealthCheck)

	// API ดึงรายการเกมทั้งหมด
	mux.HandleFunc("GET /api/games", gameHandler.GetGames)

	// API ดูรายละเอียดเกมตาม ID
	mux.HandleFunc("GET /api/games/{id}", gameHandler.GetGameByID)

	return mux
}
