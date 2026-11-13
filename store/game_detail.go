package store

import (
	"context"
	"database/sql"

	"gamehub-backend/models"
)

// ข้อมูลสรุปคะแนนเกมจาก Database
type GameRatingSummary struct {
	GameID        int64
	AverageRating float64
	TotalRatings  int64
	TotalReviews  int64
}

// ข้อมูลรีวิวเกมจาก Database
type GameReview struct {
	ID        int64
	UserID    int64
	Username  string
	GameID    int64
	Content   string
	Rating    int
	Status    string
	CreatedAt sql.NullTime
	UpdatedAt sql.NullTime
}

// ดึงรายละเอียดเกม
func GetGameDetail(
	ctx context.Context,
	db *sql.DB,
	gameID int64,
) (models.Game, error) {

	var game models.Game
	var genre, image, description sql.NullString
	var year, igdbID sql.NullInt64

	err := db.QueryRowContext(ctx, `
		SELECT id, name, genre, year, image, description, igdb_id
		FROM games
		WHERE id = $1 AND is_deleted = FALSE
	`, gameID).Scan(
		&game.ID,
		&game.Name,
		&genre,
		&year,
		&image,
		&description,
		&igdbID,
	)

	if err != nil {
		return game, err
	}

	if genre.Valid {
		game.Genre = &genre.String
	}
	if year.Valid {
		value := int(year.Int64)
		game.Year = &value
	}
	if image.Valid {
		game.Image = &image.String
	}
	if description.Valid {
		game.Description = &description.String
	}
	if igdbID.Valid {
		game.IGDBID = &igdbID.Int64
	}

	return game, nil
}

// ดึงคะแนนเฉลี่ยและจำนวนผู้ให้คะแนน
func GetGameRatingSummary(
	ctx context.Context,
	db *sql.DB,
	gameID int64,
) (GameRatingSummary, error) {

	var summary GameRatingSummary

	err := db.QueryRowContext(ctx, `
		SELECT
			$1::bigint,
			COALESCE(ROUND(AVG(rating), 2), 0)::float8,
			COUNT(*)
		FROM game_ratings
		WHERE game_id = $1
	`, gameID).Scan(
		&summary.GameID,
		&summary.AverageRating,
		&summary.TotalRatings,
	)

	if err != nil {
		return summary, err
	}

	err = db.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM game_reviews
		WHERE game_id = $1 AND status = 'visible'
	`, gameID).Scan(&summary.TotalReviews)

	if err != nil {
		return summary, err
	}

	return summary, nil
}

// ดึงรายการรีวิวที่แสดงได้
func GetGameReviews(
	ctx context.Context,
	db *sql.DB,
	gameID int64,
) ([]GameReview, error) {

	rows, err := db.QueryContext(ctx, `
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
		return nil, err
	}
	defer rows.Close()

	reviews := []GameReview{}

	for rows.Next() {
		var review GameReview

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
			return nil, err
		}

		reviews = append(reviews, review)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return reviews, nil
}
