package routes

import (
	"database/sql"
	"net/http"

	"gamehub-backend/handlers"
	"gamehub-backend/middleware"
)

func Setup(db *sql.DB) http.Handler {
	mux := http.NewServeMux()

	gameHandler := &handlers.GameHandler{DB: db}
	authHandler := &handlers.AuthHandler{DB: db}
	profileHandler := &handlers.ProfileHandler{DB: db}

	// Health Check
	mux.HandleFunc("/api/health", gameHandler.HealthCheck)

	// Authentication
	mux.HandleFunc("POST /api/register", authHandler.Register)
	mux.HandleFunc("POST /api/login", authHandler.Login)

	// User Profile
	mux.Handle("GET /api/me",
		middleware.Auth(http.HandlerFunc(profileHandler.GetMe)),
	)

	mux.Handle("PUT /api/me",
		middleware.Auth(http.HandlerFunc(profileHandler.UpdateMe)),
	)

	// Change Password
	mux.Handle("POST /api/change-password",
		middleware.Auth(http.HandlerFunc(authHandler.ChangePassword)),
	)

	// Public Game APIs
	mux.HandleFunc("GET /api/games", gameHandler.GetGames)
	mux.HandleFunc("GET /api/games/{id}", gameHandler.GetGameByID)

	// Admin Game APIs
	mux.Handle("POST /api/games",
		middleware.Auth(middleware.Admin(http.HandlerFunc(gameHandler.CreateGame))),
	)

	mux.Handle("PUT /api/games/{id}",
		middleware.Auth(middleware.Admin(http.HandlerFunc(gameHandler.UpdateGame))),
	)

	mux.Handle("DELETE /api/games/{id}",
		middleware.Auth(middleware.Admin(http.HandlerFunc(gameHandler.DeleteGame))),
	)

	mux.Handle("PATCH /api/games/{id}/restore",
		middleware.Auth(middleware.Admin(http.HandlerFunc(gameHandler.RestoreGame))),
	)

	return mux
}
