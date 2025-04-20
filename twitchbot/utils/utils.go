package utils

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

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