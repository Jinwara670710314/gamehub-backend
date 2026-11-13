package store

import (
	"context"
	"database/sql"

	"gamehub-backend/models"
)

// ข้อมูลเกมสำหรับหน้า Admin
type AdminGame struct {
	models.Game
	IsDeleted bool `json:"isDeleted"`
}

// ดึงเกมทั้งหมด รวมถึงเกมที่ถูกลบแล้ว
func GetAdminGames(
	ctx context.Context,
	db *sql.DB,
) ([]AdminGame, error) {

	rows, err := db.QueryContext(ctx, `
		SELECT id, name, genre, year, image, description, igdb_id, is_deleted
		FROM public.games
		ORDER BY id ASC
	`)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	games := []AdminGame{}

	for rows.Next() {
		var game AdminGame
		var genre, image, description sql.NullString
		var year, igdbID sql.NullInt64

		err := rows.Scan(
			&game.ID,
			&game.Name,
			&genre,
			&year,
			&image,
			&description,
			&igdbID,
			&game.IsDeleted,
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

		if igdbID.Valid {
			game.IGDBID = &igdbID.Int64
		}

		games = append(games, game)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return games, nil
}
