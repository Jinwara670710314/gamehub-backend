package store

import (
	"context"
	"database/sql"
)

type Member struct {
	ID       int64
	Username string
	Email    string
	Role     string
	Status   string
}

// ดึงรายชื่อสมาชิกทั้งหมด
func GetMembers(ctx context.Context, db *sql.DB) ([]Member, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT id, username, email, role, status
		FROM public.users
		ORDER BY id ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	members := []Member{}

	for rows.Next() {
		var member Member

		if err := rows.Scan(
			&member.ID,
			&member.Username,
			&member.Email,
			&member.Role,
			&member.Status,
		); err != nil {
			return nil, err
		}

		members = append(members, member)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return members, nil
}

// เปลี่ยนสถานะบัญชีสมาชิก โดยไม่อนุญาตให้แก้ไขบัญชี Admin
func UpdateMemberStatus(
	ctx context.Context,
	db *sql.DB,
	id int64,
	status string,
) (Member, error) {
	var member Member

	err := db.QueryRowContext(ctx, `
		UPDATE public.users
		SET status = $1,
		    updated_at = NOW()
		WHERE id = $2
		  AND role = 'member'
		RETURNING id, username, email, role, status
	`, status, id).Scan(
		&member.ID,
		&member.Username,
		&member.Email,
		&member.Role,
		&member.Status,
	)

	return member, err
}

// ตรวจสอบ Role ของบัญชีสมาชิก
func GetMemberRole(ctx context.Context, db *sql.DB, id int64) (string, error) {
	var role string

	err := db.QueryRowContext(ctx, `
		SELECT role
		FROM public.users
		WHERE id = $1
	`, id).Scan(&role)

	return role, err
}

// ตรวจสอบสถานะบัญชีผู้ใช้
func GetMemberStatus(ctx context.Context, db *sql.DB, id int64) (string, error) {
	var status string

	err := db.QueryRowContext(ctx, `
		SELECT status
		FROM public.users
		WHERE id = $1
	`, id).Scan(&status)

	return status, err
}

// ตรวจสอบ Role และ Status ของบัญชีผู้ใช้
func GetMemberAccess(ctx context.Context, db *sql.DB, id int64) (string, string, error) {
	var role string
	var status string

	err := db.QueryRowContext(ctx, `
		SELECT role, status
		FROM public.users
		WHERE id = $1
	`, id).Scan(&role, &status)

	return role, status, err
}
