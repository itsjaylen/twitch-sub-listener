package utils

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
	"twitchsublistener/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var (
	POSTGRES_USER     = os.Getenv("POSTGRES_USER")
	POSTGRES_PASSWORD = os.Getenv("POSTGRES_PASSWORD")
	POSTGRES_DB       = os.Getenv("POSTGRES_DB")
	POSTGRES_PORT     string
	POSTGRES_HOST     string
)

type DB = gorm.DB

func InitEnvDefaults() {
	if POSTGRES_PORT == "" {
		POSTGRES_PORT = "5432"
	}
	if POSTGRES_HOST == "" {
		POSTGRES_HOST = "postgres"
	}
}

func ParseArgs(host *string, port *string) {
	for _, arg := range os.Args[1:] {
		switch {
		case arg == "-h" || arg == "--help":
			fmt.Println("Usage: go run main.go [--host=HOST] [--port=PORT]")
			os.Exit(0)
		case len(arg) > 7 && arg[:7] == "--port=":
			*port = arg[7:]
		case len(arg) > 7 && arg[:7] == "--host=":
			*host = arg[7:]
		}
	}
}

func InitDB() *DB {
	port, err := strconv.Atoi(POSTGRES_PORT)
	if err != nil {
		log.Fatalf("invalid POSTGRES_PORT: %v", err)
	}
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%d sslmode=disable",
		POSTGRES_HOST, POSTGRES_USER, POSTGRES_PASSWORD, POSTGRES_DB, port)

	var db *DB
	for i := 0; i < 5; i++ {
		db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
		if err == nil {
			break
		}
		log.Printf("Retrying DB connection in 3s... attempt #%d", i+1)
		time.Sleep(3 * time.Second)
	}
	if err != nil {
		log.Fatalf("failed to connect database: %v", err)
	}

	return db
}

func AutoMigrateDB(db *DB) {
	if err := db.AutoMigrate(&models.UnknownFollowage{}, &models.SubscriptionEvent{}); err != nil {
		log.Fatalf("auto migrate failed: %v", err)
	}
}

func LogUnknownFollowage(db *DB, fromUser, toChannel string) {
	record := models.UnknownFollowage{FromUser: fromUser, ToChannel: toChannel}
	if err := db.Create(&record).Error; err != nil {
		log.Printf("[ERROR] logging unknown followage to DB: %v", err)
	}
}

func GetFollowDuration(fromUser, toChannel string) (time.Duration, error) {
	url := fmt.Sprintf("https://api.ivr.fi/v2/twitch/subage/%s/%s", fromUser, toChannel)
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("HTTP request error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("IVR API error %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var result struct {
		FollowedAt string `json:"followedAt"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, fmt.Errorf("failed to decode IVR response: %w", err)
	}
	if result.FollowedAt == "" {
		return 0, fmt.Errorf("user does not follow channel or follow date is hidden")
	}

	followedAt, err := time.Parse(time.RFC3339, result.FollowedAt)
	if err != nil {
		return 0, fmt.Errorf("invalid follow date format: %w", err)
	}
	return time.Since(followedAt), nil
}

func ComputeSusScore(subType string, dur time.Duration, err error) string {
	if subType == "Twitch Prime" {
		if err != nil {
			return "max"
		}
		if dur < 24*time.Hour {
			return "medium"
		}
	}
	return "none"
}

func RetryForever(action func() error) {
	for {
		if err := action(); err != nil {
			log.Printf("[ERROR] %v", err)
			log.Println("[INFO] Retrying in 10 seconds...")
			time.Sleep(10 * time.Second)
			continue
		}
		break
	}
}

func FormatDuration(d time.Duration) string {
	years := int(d.Hours()) / (24 * 365)
	months := (int(d.Hours()) % (24 * 365)) / (24 * 30)
	days := (int(d.Hours()) % (24 * 30)) / 24

	var parts []string
	if years > 0 {
		parts = append(parts, fmt.Sprintf("%d year(s)", years))
	}
	if months > 0 {
		parts = append(parts, fmt.Sprintf("%d month(s)", months))
	}
	if days > 0 || len(parts) == 0 {
		parts = append(parts, fmt.Sprintf("%d day(s)", days))
	}
	return strings.Join(parts, ", ")
}

func FormatTimeDuration(d time.Duration) string {
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	s := int(d.Seconds()) % 60
	return fmt.Sprintf("%02dh %02dm %02ds", h, m, s)
}

var startTime = time.Now()

func GetUptime() string {
	duration := time.Since(startTime)
	h := int(duration.Hours())
	m := int(duration.Minutes()) % 60
	s := int(duration.Seconds()) % 60
	return fmt.Sprintf("uptime: %02dh %02dm %02ds", h, m, s)
}
