package main

import (
	"context"
	"log"
	"os"
	"os/signal"
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
	pollIntervalStr := getenv("POLL_INTERVAL", "30s")

	pollInterval, err := time.ParseDuration(pollIntervalStr)
	if err != nil {
		log.Fatalf("invalid POLL_INTERVAL %q: %v", pollIntervalStr, err)
	}

	rdb := dataredis.NewClient(redisAddr)
	apiClient := apifootball.NewClient(apiKey, rdb)
	p := poller.New(apiClient, rdb, ingestionURL, pollInterval)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	log.Println("Data Provider starting...")
	p.Run(ctx)
	log.Println("Data Provider stopped.")
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
