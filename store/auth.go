package store

import (
	"context"
	"database/sql"
)

// ข้อมูลสมาชิกที่ใช้สำหรับสมัครสมาชิกและเข้าสู่ระบบ
type AuthUser struct {
	ID           int64
	Username     string
	Email        string
	PasswordHash string
	Role         string
	Status       string
}

// ตรวจสอบว่าชื่อผู้ใช้หรืออีเมลถูกใช้งานแล้วหรือไม่
func CheckUserExists(
	ctx context.Context,
	db *sql.DB,
	username string,
	email string,
) (bool, error) {

	var exists bool

	err := db.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM public.users
			WHERE LOWER(TRIM(username)) = LOWER($1)
			   OR LOWER(TRIM(email)) = LOWER($2)
		)
	`, username, email).Scan(&exists)

	return exists, err
}

// เพิ่มสมาชิกใหม่
func CreateUser(
	ctx context.Context,
	db *sql.DB,
	username string,
	email string,
	passwordHash string,
) (AuthUser, error) {

	var user AuthUser

	err := db.QueryRowContext(ctx, `
		INSERT INTO public.users (username, email, password_hash)
		VALUES ($1, $2, $3)
		RETURNING id, username, email, role
	`,
		username,
		email,
		passwordHash,
	).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.Role,
	)

	return user, err
}

// ค้นหาสมาชิกด้วยอีเมลสำหรับเข้าสู่ระบบ
func GetUserByEmail(
	ctx context.Context,
	db *sql.DB,
	email string,
) (AuthUser, error) {

	var user AuthUser

	err := db.QueryRowContext(ctx, `
		SELECT id, username, email, password_hash, role, status
		FROM public.users
		WHERE LOWER(TRIM(email)) = $1
	`, email).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.PasswordHash,
		&user.Role,
		&user.Status,
	)

	return user, err
}

// ดึง Hash ของรหัสผ่านจาก User ID
func GetPasswordHash(
	ctx context.Context,
	db *sql.DB,
	userID int64,
) (string, error) {

	var passwordHash string

	err := db.QueryRowContext(ctx, `
		SELECT password_hash
		FROM public.users
		WHERE id = $1
	`, userID).Scan(&passwordHash)

	return passwordHash, err
}

// เปลี่ยนรหัสผ่านของสมาชิก
func UpdatePassword(
	ctx context.Context,
	db *sql.DB,
	userID int64,
	newPasswordHash string,
) (bool, error) {

	result, err := db.ExecContext(ctx, `
		UPDATE public.users
		SET password_hash = $1,
		    updated_at = NOW()
		WHERE id = $2
	`, newPasswordHash, userID)

	if err != nil {
		return false, err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, err
	}

	return rowsAffected > 0, nil
}
