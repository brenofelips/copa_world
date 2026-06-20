package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"score-service/consumer"
	"score-service/redis"
)

func main() {
	brokers := getenv("KAFKA_BROKERS", "localhost:9092")
	redisAddr := getenv("REDIS_ADDR", "localhost:6379")

	rdb := redis.NewClient(redisAddr)
	c := consumer.New(brokers, "score-service", rdb)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	log.Println("Score Service starting...")
	if err := c.Run(ctx); err != nil {
		log.Fatal(err)
	}
	log.Println("Score Service stopped.")
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
