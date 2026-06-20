package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"event-service/consumer"
	"event-service/db"
)

func main() {
	brokers := getenv("KAFKA_BROKERS", "localhost:9092")
	dsn := getenv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/copaworld?sslmode=disable")

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	store, err := db.NewStore(ctx, dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer store.Close()

	c := consumer.New(brokers, "event-service", store)

	log.Println("Event Service starting...")
	if err := c.Run(ctx); err != nil {
		log.Fatal(err)
	}
	log.Println("Event Service stopped.")
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
