package store

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

type ReviewRecord struct {
	ID        int64
	UserID    int64
	Username  string
	GameID    int64
	Content   string
	Rating    int
	Status    string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// ใช้ระบุขั้นตอนที่เกิดข้อผิดพลาดใน Transaction
type ReviewStoreError struct {
	Stage string
	Err   error
}

func (e *ReviewStoreError) Error() string {
	return e.Err.Error()
}

func (e *ReviewStoreError) Unwrap() error {
	return e.Err
}

func reviewStoreError(stage string, err error) error {
	if err == nil {
		return nil
	}

	return &ReviewStoreError{
		Stage: stage,
		Err:   err,
	}
}

// ตรวจสอบว่าเกมมีอยู่จริงและยังไม่ถูกลบ
func CheckReviewGame(ctx context.Context, db *sql.DB, gameID int64) error {
	var existingGameID int64

	err := db.QueryRowContext(ctx, `
		SELECT id
		FROM games
		WHERE id = $1
		  AND is_deleted = FALSE
	`, gameID).Scan(&existingGameID)

	return err
}

// เพิ่มรีวิวและคะแนนภายใน Transaction เดียวกัน
func CreateReview(
	ctx context.Context,
	db *sql.DB,
	userID int64,
	gameID int64,
	content string,
	rating int,
) (int64, error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return 0, reviewStoreError("begin", err)
	}
	defer tx.Rollback()

	var reviewID int64

	err = tx.QueryRowContext(ctx, `
		INSERT INTO game_reviews (user_id, game_id, content)
		VALUES ($1, $2, $3)
		RETURNING id
	`, userID, gameID, content).Scan(&reviewID)

	if err != nil {
		return 0, reviewStoreError("insert", err)
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO game_ratings (user_id, game_id, rating)
		VALUES ($1, $2, $3)
		ON CONFLICT (user_id, game_id)
		DO UPDATE SET
			rating = EXCLUDED.rating,
			updated_at = NOW()
	`, userID, gameID, rating)

	if err != nil {
		return 0, reviewStoreError("rating", err)
	}

	if err := tx.Commit(); err != nil {
		return 0, reviewStoreError("commit", err)
	}

	return reviewID, nil
}

// ดึงรีวิวที่มีสถานะ visible
func GetReviews(
	ctx context.Context,
	db *sql.DB,
	gameID int64,
) ([]ReviewRecord, error) {
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
		return nil, reviewStoreError("query", err)
	}
	defer rows.Close()

	reviews := []ReviewRecord{}

	for rows.Next() {
		var review ReviewRecord

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
			return nil, reviewStoreError("scan", err)
		}

		reviews = append(reviews, review)
	}

	if err := rows.Err(); err != nil {
		return nil, reviewStoreError("rows", err)
	}

	return reviews, nil
}

// แก้ไขรีวิวและคะแนนภายใน Transaction เดียวกัน
func UpdateReview(
	ctx context.Context,
	db *sql.DB,
	reviewID int64,
	userID int64,
	content string,
	rating int,
) (int64, error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return 0, reviewStoreError("begin", err)
	}
	defer tx.Rollback()

	var gameID int64

	err = tx.QueryRowContext(ctx, `
		UPDATE game_reviews
		SET content = $1, updated_at = NOW()
		WHERE id = $2 AND user_id = $3
		RETURNING game_id
	`, content, reviewID, userID).Scan(&gameID)

	if err != nil {
		return 0, reviewStoreError("update", err)
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO game_ratings (user_id, game_id, rating)
		VALUES ($1, $2, $3)
		ON CONFLICT (user_id, game_id)
		DO UPDATE SET
			rating = EXCLUDED.rating,
			updated_at = NOW()
	`, userID, gameID, rating)

	if err != nil {
		return 0, reviewStoreError("rating", err)
	}

	if err := tx.Commit(); err != nil {
		return 0, reviewStoreError("commit", err)
	}

	return gameID, nil
}

// ลบรีวิวและคะแนนภายใน Transaction เดียวกัน
func DeleteReview(
	ctx context.Context,
	db *sql.DB,
	reviewID int64,
	userID int64,
) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return reviewStoreError("begin", err)
	}
	defer tx.Rollback()

	var gameID int64

	err = tx.QueryRowContext(ctx, `
		DELETE FROM game_reviews
		WHERE id = $1 AND user_id = $2
		RETURNING game_id
	`, reviewID, userID).Scan(&gameID)

	if err != nil {
		return reviewStoreError("delete", err)
	}

	_, err = tx.ExecContext(ctx, `
		DELETE FROM game_ratings
		WHERE user_id = $1 AND game_id = $2
	`, userID, gameID)

	if err != nil {
		return reviewStoreError("rating", err)
	}

	if err := tx.Commit(); err != nil {
		return reviewStoreError("commit", err)
	}

	return nil
}

// ตรวจสอบว่า Error เป็นข้อผิดพลาดจากขั้นตอนใด
func IsReviewStage(err error, stage string) bool {
	var storeErr *ReviewStoreError

	return errors.As(err, &storeErr) && storeErr.Stage == stage
}
