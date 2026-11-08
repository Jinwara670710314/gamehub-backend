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
	libraryHandler := &handlers.LibraryHandler{DB: db}
	reviewHandler := &handlers.ReviewHandler{DB: db}
	ratingHandler := &handlers.RatingHandler{DB: db}
	gameDetailHandler := &handlers.GameDetailHandler{DB: db}

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

	// User Game Library
	mux.Handle("POST /api/library",
		middleware.Auth(http.HandlerFunc(libraryHandler.AddGame)),
	)
	mux.Handle("GET /api/library",
		middleware.Auth(http.HandlerFunc(libraryHandler.GetLibrary)),
	)
	mux.Handle("PUT /api/library/{id}",
		middleware.Auth(http.HandlerFunc(libraryHandler.UpdateGameStatus)),
	)
	mux.Handle("DELETE /api/library/{id}",
		middleware.Auth(http.HandlerFunc(libraryHandler.RemoveGame)),
	)

	// Public Game APIs
	mux.HandleFunc("GET /api/games", gameHandler.GetGames)
	mux.HandleFunc("GET /api/games/{id}", gameHandler.GetGameByID)

	// Game Detail: ข้อมูลเกม + คะแนน + รีวิว
	mux.HandleFunc("GET /api/games/{id}/detail",
		gameDetailHandler.GetGameDetail,
	)

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

	// Game Reviews
	mux.Handle("POST /api/games/{id}/reviews",
		middleware.Auth(http.HandlerFunc(reviewHandler.CreateReview)),
	)
	mux.HandleFunc("GET /api/games/{id}/reviews", reviewHandler.GetReviews)
	mux.Handle("PUT /api/reviews/{id}",
		middleware.Auth(http.HandlerFunc(reviewHandler.UpdateReview)),
	)
	mux.Handle("DELETE /api/reviews/{id}",
		middleware.Auth(http.HandlerFunc(reviewHandler.DeleteReview)),
	)

	// Rating Summary
	mux.HandleFunc("GET /api/games/{id}/rating-summary",
		ratingHandler.GetRatingSummary,
	)

	return mux
}
