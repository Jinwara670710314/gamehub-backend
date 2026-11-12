package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"

	"gamehub-backend/database"
	"gamehub-backend/services"
)

func main() {
	dryRun := flag.Bool("dry-run", false, "ตรวจสอบเกมโดยไม่บันทึกลงฐานข้อมูล")
	flag.Parse()

	// โหลดค่าจาก .env
	if err := godotenv.Load(); err != nil {
		log.Println("ไม่พบไฟล์ .env จะลองอ่านค่าจาก Environment Variables")
	}

	clientID := os.Getenv("TWITCH_CLIENT_ID")
	clientSecret := os.Getenv("TWITCH_CLIENT_SECRET")
	databaseURL := os.Getenv("DATABASE_URL")

	if clientID == "" || clientSecret == "" || databaseURL == "" {
		log.Fatal("กรุณาตรวจสอบ TWITCH_CLIENT_ID, TWITCH_CLIENT_SECRET และ DATABASE_URL")
	}

	// เชื่อมต่อ Supabase
	db, err := database.Connect(databaseURL)
	if err != nil {
		log.Fatalf("เชื่อมต่อฐานข้อมูลไม่สำเร็จ: %v", err)
	}
	defer db.Close()

	log.Println("เชื่อมต่อ Supabase สำเร็จ")

	// เชื่อมต่อ IGDB
	igdb := services.NewIGDBClient(clientID, clientSecret)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	log.Println("กำลังดึงข้อมูลเกมจาก IGDB...")

	games, err := igdb.FetchGames(ctx, 100)
	if err != nil {
		log.Fatalf("ดึงข้อมูลจาก IGDB ไม่สำเร็จ: %v", err)
	}

	log.Printf("ได้รับข้อมูลเกม %d เกม", len(games))

	// โหมดตรวจสอบก่อนนำเข้าจริง
	if *dryRun {
		log.Println("========== DRY RUN ==========")
		log.Println("กำลังตรวจสอบเกมซ้ำกับ Supabase")
		log.Println("ยังไม่มีการเพิ่มหรือแก้ไขข้อมูล")

		var added int
		var matched int
		var skipped int
		var failed int

		for i, game := range games {
			if game.ID <= 0 || strings.TrimSpace(game.Name) == "" {
				skipped++
				log.Printf("[%d] SKIPPED | ข้อมูลเกมไม่สมบูรณ์", i+1)
				continue
			}

			status, err := previewGame(ctx, db, game)
			if err != nil {
				failed++
				log.Printf("[%d] ERROR | %s | %v", i+1, game.Name, err)
				continue
			}

			switch status {
			case "ADDED":
				added++
			case "MATCHED":
				matched++
			default:
				skipped++
			}

			log.Printf(
				"[%d] %s | %s | IGDB ID: %d",
				i+1,
				status,
				game.Name,
				game.ID,
			)
		}

		log.Println("--------------------------------")
		log.Println("สรุปผลการตรวจสอบ (ยังไม่บันทึกข้อมูล)")
		log.Printf("เกมใหม่ที่พร้อมเพิ่ม: %d", added)
		log.Printf("เกมเดิมที่จับคู่ IGDB ID: %d", matched)
		log.Printf("เกมที่ต้องข้าม: %d", skipped)
		log.Printf("รายการที่ตรวจสอบไม่สำเร็จ: %d", failed)
		log.Println("--------------------------------")
		return
	}

	// โหมดนำเข้าจริง
	var added int
	var matched int
	var skipped int
	var failed int

	for _, game := range games {
		if game.ID <= 0 || strings.TrimSpace(game.Name) == "" {
			skipped++
			continue
		}

		result, err := importGame(ctx, db, game)
		if err != nil {
			failed++
			log.Printf("นำเข้า %s ไม่สำเร็จ: %v", game.Name, err)
			continue
		}

		switch result {
		case "added":
			added++
			log.Printf("เพิ่มเกมใหม่: %s", game.Name)
		case "matched":
			matched++
			log.Printf("จับคู่เกมเดิม: %s", game.Name)
		default:
			skipped++
			log.Printf("ข้ามเกม: %s", game.Name)
		}
	}

	log.Println("--------------------------------")
	log.Println("สรุปผลการนำเข้าเกม")
	log.Printf("เกมที่เพิ่มใหม่: %d", added)
	log.Printf("เกมเดิมที่จับคู่ IGDB ID: %d", matched)
	log.Printf("เกมที่ข้าม: %d", skipped)
	log.Printf("รายการที่เกิดข้อผิดพลาด: %d", failed)
	log.Println("--------------------------------")
}

// previewGame ตรวจสอบว่าเกมจะถูกเพิ่ม จับคู่ หรือข้าม
// ฟังก์ชันนี้อ่านข้อมูลอย่างเดียว ไม่แก้ไขฐานข้อมูล
func previewGame(
	ctx context.Context,
	db *sql.DB,
	game services.IGDBGame,
) (string, error) {

	// ตรวจสอบ IGDB ID ก่อน
	var existingID int64

	err := db.QueryRowContext(
		ctx,
		`SELECT id
		 FROM public.games
		 WHERE igdb_id = $1
		 LIMIT 1`,
		game.ID,
	).Scan(&existingID)

	if err == nil {
		return "SKIPPED", nil
	}

	if err != sql.ErrNoRows {
		return "", err
	}

	// ตรวจสอบชื่อเกมเดิมที่ยังไม่ถูกลบ
	var existingGameID int64
	var existingIGDBID sql.NullInt64

	err = db.QueryRowContext(
		ctx,
		`SELECT id, igdb_id
		 FROM public.games
		 WHERE LOWER(TRIM(name)) = LOWER(TRIM($1))
		   AND is_deleted = FALSE
		 ORDER BY id
		 LIMIT 1`,
		game.Name,
	).Scan(&existingGameID, &existingIGDBID)

	if err == nil {
		if existingIGDBID.Valid {
			return "SKIPPED", nil
		}

		return "MATCHED", nil
	}

	if err != sql.ErrNoRows {
		return "", err
	}

	return "ADDED", nil
}

// importGame นำเข้าเกมโดยตรวจสอบข้อมูลเดิมก่อน
func importGame(
	ctx context.Context,
	db *sql.DB,
	game services.IGDBGame,
) (string, error) {

	// ตรวจสอบ IGDB ID
	var existingID int64

	err := db.QueryRowContext(
		ctx,
		`SELECT id
		 FROM public.games
		 WHERE igdb_id = $1
		 LIMIT 1`,
		game.ID,
	).Scan(&existingID)

	if err == nil {
		return "skipped", nil
	}

	if err != sql.ErrNoRows {
		return "", err
	}

	// เตรียมข้อมูลเกมจาก IGDB
	var genre any
	if value := game.GetGenre(); value != "" {
		genre = value
	}

	var year any
	if value := game.GetYear(); value != nil {
		year = *value
	}

	var image any
	if value := game.GetImageURL(); value != "" {
		image = value
	}

	var description any
	if value := strings.TrimSpace(game.Summary); value != "" {
		description = value
	}

	// ตรวจสอบชื่อเกมเดิมที่ยังไม่ถูกลบ
	var existingGameID int64
	var existingIGDBID sql.NullInt64

	err = db.QueryRowContext(
		ctx,
		`SELECT id, igdb_id
		 FROM public.games
		 WHERE LOWER(TRIM(name)) = LOWER(TRIM($1))
		   AND is_deleted = FALSE
		 ORDER BY id
		 LIMIT 1`,
		game.Name,
	).Scan(&existingGameID, &existingIGDBID)

	if err == nil {
		if existingIGDBID.Valid {
			return "skipped", nil
		}

		// เพิ่มเฉพาะ IGDB ID ไม่เขียนทับข้อมูลเดิม
		_, err = db.ExecContext(
			ctx,
			`UPDATE public.games
			 SET igdb_id = $1
			 WHERE id = $2
			   AND igdb_id IS NULL`,
			game.ID,
			existingGameID,
		)

		if err != nil {
			return "", err
		}

		return "matched", nil
	}

	if err != sql.ErrNoRows {
		return "", err
	}

	// เพิ่มเกมใหม่
	_, err = db.ExecContext(
		ctx,
		`INSERT INTO public.games
		 (name, genre, year, image, description, igdb_id, is_deleted)
		 VALUES ($1, $2, $3, $4, $5, $6, FALSE)
		 ON CONFLICT (igdb_id)
		 WHERE igdb_id IS NOT NULL
		 DO NOTHING`,
		game.Name,
		genre,
		year,
		image,
		description,
		game.ID,
	)

	if err != nil {
		return "", fmt.Errorf("เพิ่มข้อมูลลงฐานข้อมูลไม่สำเร็จ: %w", err)
	}

	return "added", nil
}
