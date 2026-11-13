package store

import (
	"context"
	"database/sql"

	"gamehub-backend/models"
)

// แปลงข้อมูลจาก Database เป็น Game Model
type rowScanner interface {
	Scan(dest ...any) error
}

func scanGame(row rowScanner) (models.Game, error) {
	var game models.Game
	var genre, image, description sql.NullString
	var year, igdbID sql.NullInt64

	err := row.Scan(
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

// ดึงรายการเกมและจำนวนเกมทั้งหมด
func GetGames(
	ctx context.Context,
	db *sql.DB,
	search string,
	limit int,
	offset int,
) ([]models.Game, int, error) {

	var total int

	err := db.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM public.games
		WHERE is_deleted = FALSE
		AND ($1 = '' OR name ILIKE '%' || $1 || '%')
	`, search).Scan(&total)

	if err != nil {
		return nil, 0, err
	}

	rows, err := db.QueryContext(ctx, `
		SELECT id, name, genre, year, image, description, igdb_id
		FROM public.games
		WHERE is_deleted = FALSE
		AND ($1 = '' OR name ILIKE '%' || $1 || '%')
		ORDER BY id ASC
		LIMIT $2 OFFSET $3
	`, search, limit, offset)

	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	games := []models.Game{}

	for rows.Next() {
		game, err := scanGame(rows)
		if err != nil {
			return nil, 0, err
		}

		games = append(games, game)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return games, total, nil
}

// ดึงรายละเอียดเกมตาม ID
func GetGameByID(
	ctx context.Context,
	db *sql.DB,
	id int64,
) (models.Game, error) {

	row := db.QueryRowContext(ctx, `
		SELECT id, name, genre, year, image, description, igdb_id
		FROM public.games
		WHERE id = $1 AND is_deleted = FALSE
	`, id)

	return scanGame(row)
}

// เพิ่มเกมใหม่
func CreateGame(
	ctx context.Context,
	db *sql.DB,
	input models.Game,
) (models.Game, error) {

	row := db.QueryRowContext(ctx, `
		INSERT INTO public.games
			(name, genre, year, image, description, igdb_id, is_deleted)
		VALUES
			($1, $2, $3, $4, $5, $6, FALSE)
		RETURNING id, name, genre, year, image, description, igdb_id
	`,
		input.Name,
		input.Genre,
		input.Year,
		input.Image,
		input.Description,
		input.IGDBID,
	)

	return scanGame(row)
}

// แก้ไขข้อมูลเกม
func UpdateGame(
	ctx context.Context,
	db *sql.DB,
	id int64,
	input models.Game,
) (models.Game, error) {

	row := db.QueryRowContext(ctx, `
		UPDATE public.games
		SET
			name = $1,
			genre = $2,
			year = $3,
			image = $4,
			description = $5,
			igdb_id = $6,
			updated_at = NOW()
		WHERE id = $7 AND is_deleted = FALSE
		RETURNING id, name, genre, year, image, description, igdb_id
	`,
		input.Name,
		input.Genre,
		input.Year,
		input.Image,
		input.Description,
		input.IGDBID,
		id,
	)

	return scanGame(row)
}

// ลบเกมแบบ Soft Delete
func DeleteGame(
	ctx context.Context,
	db *sql.DB,
	id int64,
) (bool, error) {

	result, err := db.ExecContext(ctx, `
		UPDATE public.games
		SET is_deleted = TRUE,
		    updated_at = NOW()
		WHERE id = $1 AND is_deleted = FALSE
	`, id)

	if err != nil {
		return false, err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, err
	}

	return rowsAffected > 0, nil
}

// กู้คืนเกมที่ถูก Soft Delete
func RestoreGame(
	ctx context.Context,
	db *sql.DB,
	id int64,
) (bool, error) {

	result, err := db.ExecContext(ctx, `
		UPDATE public.games
		SET is_deleted = FALSE,
		    updated_at = NOW()
		WHERE id = $1 AND is_deleted = TRUE
	`, id)

	if err != nil {
		return false, err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, err
	}

	return rowsAffected > 0, nil
}
