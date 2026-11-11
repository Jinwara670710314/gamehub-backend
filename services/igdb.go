
package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

const (
	twitchTokenURL = "https://id.twitch.tv/oauth2/token"
	igdbGamesURL   = "https://api.igdb.com/v4/games"
)

// IGDBClient ใช้เชื่อมต่อกับ IGDB API
type IGDBClient struct {
	clientID     string
	clientSecret string
	httpClient   *http.Client

	mu          sync.Mutex
	accessToken string
	tokenExpiry time.Time
}

// IGDBGame ข้อมูลเกมที่ได้รับจาก IGDB
type IGDBGame struct {
	ID              int64       `json:"id"`
	Name            string      `json:"name"`
	FirstReleaseDate int64      `json:"first_release_date"`
	Summary         string      `json:"summary"`
	Cover           *IGDBCover  `json:"cover"`
	Genres          []IGDBGenre `json:"genres"`
}

type IGDBCover struct {
	ImageID string `json:"image_id"`
}

type IGDBGenre struct {
	Name string `json:"name"`
}

type twitchTokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int64  `json:"expires_in"`
}

// NewIGDBClient สร้างตัวเชื่อมต่อ IGDB
func NewIGDBClient(clientID, clientSecret string) *IGDBClient {
	return &IGDBClient{
		clientID:     clientID,
		clientSecret: clientSecret,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// getAccessToken ขอ Access Token จาก Twitch
func (c *IGDBClient) getAccessToken(ctx context.Context) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// ใช้ Token เดิมถ้ายังไม่หมดอายุ
	if c.accessToken != "" && time.Now().Before(c.tokenExpiry) {
		return c.accessToken, nil
	}

	form := url.Values{}
	form.Set("client_id", c.clientID)
	form.Set("client_secret", c.clientSecret)
	form.Set("grant_type", "client_credentials")

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		twitchTokenURL,
		strings.NewReader(form.Encode()),
	)
	if err != nil {
		return "", fmt.Errorf("สร้างคำขอ Token ไม่สำเร็จ: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("เชื่อมต่อ Twitch ไม่สำเร็จ: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Twitch ขอ Token ไม่สำเร็จ: HTTP %d", resp.StatusCode)
	}

	var tokenData twitchTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenData); err != nil {
		return "", fmt.Errorf("อ่านข้อมูล Token ไม่สำเร็จ: %w", err)
	}

	if tokenData.AccessToken == "" {
		return "", fmt.Errorf("Twitch ไม่ได้ส่ง Access Token กลับมา")
	}

	c.accessToken = tokenData.AccessToken
	c.tokenExpiry = time.Now().Add(
		time.Duration(tokenData.ExpiresIn-60) * time.Second,
	)

	return c.accessToken, nil
}

// FetchGames ดึงข้อมูลเกมจาก IGDB
func (c *IGDBClient) FetchGames(ctx context.Context, limit int) ([]IGDBGame, error) {
	if limit <= 0 {
		return nil, fmt.Errorf("จำนวนเกมต้องมากกว่า 0")
	}

	if limit > 500 {
		limit = 500
	}

	token, err := c.getAccessToken(ctx)
	if err != nil {
		return nil, err
	}

	query := fmt.Sprintf(
		"fields id,name,first_release_date,summary,cover.image_id,genres.name,total_rating_count; sort total_rating_count desc; limit %d;",
		limit,
	)

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		igdbGamesURL,
		bytes.NewBufferString(query),
	)
	if err != nil {
		return nil, fmt.Errorf("สร้างคำขอ IGDB ไม่สำเร็จ: %w", err)
	}

	req.Header.Set("Client-ID", c.clientID)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "text/plain")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("เชื่อมต่อ IGDB ไม่สำเร็จ: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("IGDB ส่งข้อมูลไม่สำเร็จ: HTTP %d", resp.StatusCode)
	}

	var games []IGDBGame
	if err := json.NewDecoder(resp.Body).Decode(&games); err != nil {
		return nil, fmt.Errorf("อ่านข้อมูลเกมจาก IGDB ไม่สำเร็จ: %w", err)
	}

	return games, nil
}

// GetGenre คืนชื่อแนวเกมแรกที่พบ
func (g IGDBGame) GetGenre() string {
	if len(g.Genres) == 0 {
		return ""
	}

	return g.Genres[0].Name
}

// GetYear คืนปีที่เกมวางจำหน่าย
func (g IGDBGame) GetYear() *int {
	if g.FirstReleaseDate <= 0 {
		return nil
	}

	year := time.Unix(g.FirstReleaseDate, 0).UTC().Year()
	return &year
}

// GetImageURL คืน URL รูปปกเกม
func (g IGDBGame) GetImageURL() string {
	if g.Cover == nil || g.Cover.ImageID == "" {
		return ""
	}

	return fmt.Sprintf(
		"https://images.igdb.com/igdb/image/upload/t_cover_big/%s.jpg",
		g.Cover.ImageID,
	)
}
