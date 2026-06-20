package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"data-provider/apifootball"
	"data-provider/poller"
	dataredis "data-provider/redis"
)

func main() {
	apiKey := os.Getenv("API_FOOTBALL_KEY")
	if apiKey == "" {
		log.Fatal("API_FOOTBALL_KEY environment variable is required")
	}

	ingestionURL := getenv("INGESTION_URL", "http://nginx/events")
	redisAddr := getenv("REDIS_ADDR", "redis:6379")

	pollInterval, err := time.ParseDuration(getenv("POLL_INTERVAL", "30s"))
	if err != nil {
		log.Fatalf("invalid POLL_INTERVAL: %v", err)
	}

	scheduleInterval, err := time.ParseDuration(getenv("SCHEDULE_INTERVAL", "1h"))
	if err != nil {
		log.Fatalf("invalid SCHEDULE_INTERVAL: %v", err)
	}

	leagueID, err := strconv.Atoi(getenv("LEAGUE_ID", "1"))
	if err != nil {
		log.Fatalf("invalid LEAGUE_ID: %v", err)
	}

	season, err := strconv.Atoi(getenv("SEASON", "2026"))
	if err != nil {
		log.Fatalf("invalid SEASON: %v", err)
	}

	rdb := dataredis.NewClient(redisAddr)
	apiClient := apifootball.NewClient(apiKey, leagueID, season, rdb)
	p := poller.New(apiClient, rdb, ingestionURL, pollInterval, scheduleInterval, season)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	log.Printf("Data Provider starting (league=%d, season=%d)...", leagueID, season)
	p.Run(ctx)
	log.Println("Data Provider stopped.")
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
