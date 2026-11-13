package store

import (
	"context"
	"database/sql"
)

type LibraryGame struct {
	ID          int64   `json:"id"`
	GameID      int64   `json:"game_id"`
	Name        string  `json:"name"`
	Genre       *string `json:"genre"`
	Year        *int    `json:"year"`
	Image       *string `json:"image"`
	Description *string `json:"description"`
	Status      string  `json:"status"`
}

// เพิ่มเกมเข้าคลังส่วนตัว
func AddLibraryGame(
	ctx context.Context,
	db *sql.DB,
	userID int64,
	gameID int64,
	status string,
) (int64, error) {
	var libraryID int64

	err := db.QueryRowContext(ctx, `
		INSERT INTO user_game_library (user_id, game_id, status)
		VALUES ($1, $2, $3)
		RETURNING id
	`,
		userID,
		gameID,
		status,
	).Scan(&libraryID)

	return libraryID, err
}

// ดึงรายการเกมในคลังส่วนตัว
func GetLibraryGames(
	ctx context.Context,
	db *sql.DB,
	userID int64,
) ([]LibraryGame, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT
			ugl.id,
			g.id,
			g.name,
			g.genre,
			g.year,
			g.image,
			g.description,
			ugl.status
		FROM user_game_library ugl
		JOIN games g ON g.id = ugl.game_id
		WHERE ugl.user_id = $1
		  AND g.is_deleted = FALSE
		ORDER BY ugl.created_at DESC
	`,
		userID,
	)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	games := []LibraryGame{}

	for rows.Next() {
		var game LibraryGame

		var genre sql.NullString
		var year sql.NullInt64
		var image sql.NullString
		var description sql.NullString

		err := rows.Scan(
			&game.ID,
			&game.GameID,
			&game.Name,
			&genre,
			&year,
			&image,
			&description,
			&game.Status,
		)

		if err != nil {
			return nil, err
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

		games = append(games, game)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return games, nil
}

// เปลี่ยนสถานะเกมในคลัง
func UpdateLibraryGameStatus(
	ctx context.Context,
	db *sql.DB,
	libraryID int64,
	userID int64,
	status string,
) (bool, error) {
	result, err := db.ExecContext(ctx, `
		UPDATE user_game_library
		SET status = $1, updated_at = NOW()
		WHERE id = $2 AND user_id = $3
	`,
		status,
		libraryID,
		userID,
	)

	if err != nil {
		return false, err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return false, err
	}

	return affected > 0, nil
}

// นำเกมออกจากคลังส่วนตัว
func RemoveLibraryGame(
	ctx context.Context,
	db *sql.DB,
	libraryID int64,
	userID int64,
) (bool, error) {
	result, err := db.ExecContext(ctx, `
		DELETE FROM user_game_library
		WHERE id = $1 AND user_id = $2
	`,
		libraryID,
		userID,
	)

	if err != nil {
		return false, err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return false, err
	}

	return affected > 0, nil
}
