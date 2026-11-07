
package main

import (
	"log"
	"net/http"
	"os"

	"gamehub-backend/database"
	"gamehub-backend/routes"

	"github.com/joho/godotenv"
)

func main() {
	// อ่านไฟล์ .env
	if err := godotenv.Load(); err != nil {
		log.Println("ไม่พบไฟล์ .env")
	}

	// อ่าน DATABASE_URL
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal("ไม่พบ DATABASE_URL")
	}

	// เชื่อมต่อ Supabase
	db, err := database.Connect(databaseURL)
	if err != nil {
		log.Fatal("เชื่อมต่อ Database ไม่สำเร็จ:", err)
	}
	defer db.Close()

	// กำหนด Routes
	router := routes.Setup(db)

	// กำหนด Port
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// เริ่มต้น Server
	log.Println("GameHub Backend running on port", port)

	if err := http.ListenAndServe("0.0.0.0:"+port, router); err != nil {
		log.Fatal(err)
	}
}
