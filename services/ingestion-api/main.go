package main

import (
	"log"
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"ingestion-api/handlers"
	"ingestion-api/kafka"
)

func main() {
	brokers := getenv("KAFKA_BROKERS", "localhost:9092")
	port := getenv("PORT", "8081")

	producer := kafka.NewProducer(brokers)
	defer producer.Close()

	app := fiber.New(fiber.Config{
		AppName: "Copa World Ingestion API",
	})
	app.Use(recover.New())
	app.Use(logger.New())

	h := handlers.NewEventHandler(producer)

	app.Post("/events", h.HandleEvent)
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok", "service": "ingestion-api"})
	})

	log.Printf("Ingestion API listening on :%s", port)
	log.Fatal(app.Listen(":" + port))
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
