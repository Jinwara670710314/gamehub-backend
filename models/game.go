package models

// โครงสร้างข้อมูลเกม
type Game struct {
    ID          int64   `json:"id"`
    Name        string  `json:"name"`
    Genre       *string `json:"genre"`
    Year        *int    `json:"year"`
    Image       *string `json:"image"`
    Description *string `json:"description"`
    RAWGID      *int64  `json:"rawg_id"`
}