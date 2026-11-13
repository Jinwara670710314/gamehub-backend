package store

import (
	"context"
	"database/sql"
)

type RatingOverview struct {
	GameID        int64
	AverageRating float64
	TotalRatings  int64
	TotalReviews  int64
}

// ตรวจสอบว่าเกมมีอยู่จริงและยังไม่ถูกลบ
func CheckActiveGame(ctx context.Context, db *sql.DB, gameID int64) error {
	var existingGameID int64

	err := db.QueryRowContext(ctx, `
		SELECT id
		FROM games
		WHERE id = $1
		  AND is_deleted = FALSE
	`, gameID).Scan(&existingGameID)

	return err
}

// บันทึกคะแนนใหม่ หรืออัปเดตคะแนนเดิมของผู้ใช้
func UpsertRating(
	ctx context.Context,
	db *sql.DB,
	userID int64,
	gameID int64,
	rating int,
) error {
	_, err := db.ExecContext(ctx, `
		INSERT INTO game_ratings (user_id, game_id, rating)
		VALUES ($1, $2, $3)
		ON CONFLICT (user_id, game_id)
		DO UPDATE SET
			rating = EXCLUDED.rating,
			updated_at = NOW()
	`, userID, gameID, rating)

	return err
}

// ดึงข้อมูลสรุปคะแนนและจำนวนรีวิวของเกม
func GetRatingOverview(
	ctx context.Context,
	db *sql.DB,
	gameID int64,
) (RatingOverview, error) {
	var summary RatingOverview

	err := db.QueryRowContext(ctx, `
		SELECT
			g.id,

			COALESCE(
				(
					SELECT ROUND(AVG(rating)::numeric, 2)
					FROM game_ratings
					WHERE game_id = g.id
				),
				0
			)::float8,

			(
				SELECT COUNT(*)
				FROM game_ratings
				WHERE game_id = g.id
			),

			(
				SELECT COUNT(*)
				FROM game_reviews
				WHERE game_id = g.id
				  AND status = 'visible'
			)

		FROM games g
		WHERE g.id = $1
		  AND g.is_deleted = FALSE
	`, gameID).Scan(
		&summary.GameID,
		&summary.AverageRating,
		&summary.TotalRatings,
		&summary.TotalReviews,
	)

	return summary, err
}
