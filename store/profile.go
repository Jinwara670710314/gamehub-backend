package store

import (
	"context"
	"database/sql"
)

type UserProfile struct {
	ID       int64
	Username string
	Email    string
	Role     string
	Status   string
}

// ดึงข้อมูลโปรไฟล์ผู้ใช้
func GetUserProfile(ctx context.Context, db *sql.DB, userID int64) (UserProfile, error) {
	var profile UserProfile

	err := db.QueryRowContext(ctx, `
		SELECT id, username, email, role, status
		FROM users
		WHERE id = $1
	`, userID).Scan(
		&profile.ID,
		&profile.Username,
		&profile.Email,
		&profile.Role,
		&profile.Status,
	)

	return profile, err
}

// ตรวจสอบ Username ซ้ำ โดยไม่นับบัญชีของผู้ใช้ปัจจุบัน
func UsernameExists(ctx context.Context, db *sql.DB, username string, userID int64) (bool, error) {
	var exists bool

	err := db.QueryRowContext(ctx, `
		SELECT EXISTS(
			SELECT 1
			FROM users
			WHERE LOWER(TRIM(username)) = LOWER($1)
			  AND id <> $2
		)
	`, username, userID).Scan(&exists)

	return exists, err
}

// ตรวจสอบ Email ซ้ำ โดยไม่นับบัญชีของผู้ใช้ปัจจุบัน
func EmailExists(ctx context.Context, db *sql.DB, email string, userID int64) (bool, error) {
	var exists bool

	err := db.QueryRowContext(ctx, `
		SELECT EXISTS(
			SELECT 1
			FROM users
			WHERE LOWER(TRIM(email)) = LOWER($1)
			  AND id <> $2
		)
	`, email, userID).Scan(&exists)

	return exists, err
}

// อัปเดตข้อมูลโปรไฟล์
func UpdateUserProfile(
	ctx context.Context,
	db *sql.DB,
	userID int64,
	username string,
	email string,
) (UserProfile, error) {
	var profile UserProfile

	err := db.QueryRowContext(ctx, `
		UPDATE users
		SET username = $1,
		    email = $2,
		    updated_at = NOW()
		WHERE id = $3
		RETURNING id, username, email, role
	`, username, email, userID).Scan(
		&profile.ID,
		&profile.Username,
		&profile.Email,
		&profile.Role,
	)

	return profile, err
}
