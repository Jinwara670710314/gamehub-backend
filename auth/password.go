package auth

import "golang.org/x/crypto/bcrypt"

// เข้ารหัสรหัสผ่านก่อนบันทึกลง Database
func HashPassword(password string) (string, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword(
		[]byte(password),
		bcrypt.DefaultCost,
	)

	if err != nil {
		return "", err
	}

	return string(hashedPassword), nil
}

// ตรวจสอบรหัสผ่านกับ Hash ที่เก็บไว้ใน Database
func CheckPassword(hashedPassword string, password string) error {
	return bcrypt.CompareHashAndPassword(
		[]byte(hashedPassword),
		[]byte(password),
	)
}
