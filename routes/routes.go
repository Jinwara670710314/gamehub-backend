
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
	memberHandler := &handlers.MemberHandler{DB: db}

	// ตรวจสอบ Token และสถานะบัญชีผู้ใช้
	authActive := func(handler http.Handler) http.Handler {
		return middleware.Auth(
			middleware.Active(db)(handler),
		)
	}

	// Health Check
	mux.HandleFunc("/api/health", gameHandler.HealthCheck)

	// Authentication
	mux.HandleFunc("POST /api/register", authHandler.Register)
	mux.HandleFunc("POST /api/login", authHandler.Login)

	// User Profile
	mux.Handle("GET /api/me",
		authActive(http.HandlerFunc(profileHandler.GetMe)),
	)
	mux.Handle("PUT /api/me",
		authActive(http.HandlerFunc(profileHandler.UpdateMe)),
	)

	// Change Password
	mux.Handle("POST /api/change-password",
		authActive(http.HandlerFunc(authHandler.ChangePassword)),
	)

	// User Game Library
	mux.Handle("POST /api/library",
		authActive(http.HandlerFunc(libraryHandler.AddGame)),
	)
	mux.Handle("GET /api/library",
		authActive(http.HandlerFunc(libraryHandler.GetLibrary)),
	)
	mux.Handle("PUT /api/library/{id}",
		authActive(http.HandlerFunc(libraryHandler.UpdateGameStatus)),
	)
	mux.Handle("DELETE /api/library/{id}",
		authActive(http.HandlerFunc(libraryHandler.RemoveGame)),
	)

	// Public Game APIs
	mux.HandleFunc("GET /api/games", gameHandler.GetGames)
	mux.HandleFunc("GET /api/games/{id}", gameHandler.GetGameByID)

	// Game Detail
	mux.HandleFunc("GET /api/games/{id}/detail",
		gameDetailHandler.GetGameDetail,
	)

	// Admin Game APIs
	mux.Handle("GET /api/admin/games",
		middleware.Auth(
			middleware.Admin(db)(
				http.HandlerFunc(gameHandler.GetAdminGames),
			),
		),
	)

	mux.Handle("POST /api/games",
		middleware.Auth(
			middleware.Admin(db)(
				http.HandlerFunc(gameHandler.CreateGame),
			),
		),
	)

	mux.Handle("PUT /api/games/{id}",
		middleware.Auth(
			middleware.Admin(db)(
				http.HandlerFunc(gameHandler.UpdateGame),
			),
		),
	)

	mux.Handle("DELETE /api/games/{id}",
		middleware.Auth(
			middleware.Admin(db)(
				http.HandlerFunc(gameHandler.DeleteGame),
			),
		),
	)

	mux.Handle("PATCH /api/games/{id}/restore",
		middleware.Auth(
			middleware.Admin(db)(
				http.HandlerFunc(gameHandler.RestoreGame),
			),
		),
	)

	// Admin Member APIs
	mux.Handle("GET /api/admin/members",
		middleware.Auth(
			middleware.Admin(db)(
				http.HandlerFunc(memberHandler.GetMembers),
			),
		),
	)

	mux.Handle("PATCH /api/admin/members/{id}/status",
		middleware.Auth(
			middleware.Admin(db)(
				http.HandlerFunc(memberHandler.UpdateMemberStatus),
			),
		),
	)

	// Game Reviews
	mux.Handle("POST /api/games/{id}/reviews",
		authActive(http.HandlerFunc(reviewHandler.CreateReview)),
	)
	mux.HandleFunc("GET /api/games/{id}/reviews", reviewHandler.GetReviews)

	mux.Handle("PUT /api/reviews/{id}",
		authActive(http.HandlerFunc(reviewHandler.UpdateReview)),
	)
	mux.Handle("DELETE /api/reviews/{id}",
		authActive(http.HandlerFunc(reviewHandler.DeleteReview)),
	)

	// Game Rating
	mux.Handle("POST /api/games/{id}/rating",
		authActive(http.HandlerFunc(ratingHandler.SubmitRating)),
	)

	// Rating Summary
	mux.HandleFunc("GET /api/games/{id}/rating-summary",
		ratingHandler.GetRatingSummary,
	)

	// เปิดใช้งาน CORS และ Logging
	return middleware.Logging(corsMiddleware(mux))
}

// CORS Middleware
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")

		allowedOrigins := map[string]bool{
			"http://127.0.0.1:3000": true,
			"http://localhost:3000": true,
		}

		if allowedOrigins[origin] {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
		}

		w.Header().Set(
			"Access-Control-Allow-Methods",
			"GET, POST, PUT, PATCH, DELETE, OPTIONS",
		)

		w.Header().Set(
			"Access-Control-Allow-Headers",
			"Content-Type, Authorization",
		)

		// Handle browser preflight request
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}
