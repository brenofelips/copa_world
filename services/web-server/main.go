package main

import (
	"context"
	"log"
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"web-server/db"
	"web-server/handlers"
	"web-server/redis"
)

func main() {
	port := getenv("PORT", "8080")
	redisAddr := getenv("REDIS_ADDR", "localhost:6379")
	primaryDSN := getenv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/copaworld?sslmode=disable")
	replicaDSN := getenv("DATABASE_REPLICA_URL", primaryDSN)

	rdb := redis.NewClient(redisAddr)

	ctx := context.Background()
	store, err := db.NewStore(ctx, primaryDSN, replicaDSN)
	if err != nil {
		log.Fatal(err)
	}
	defer store.Close()

	app := fiber.New(fiber.Config{
		AppName: "Copa World Web Server",
	})
	app.Use(recover.New())
	app.Use(logger.New())
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowMethods: "GET",
		AllowHeaders: "Accept,Content-Type",
	}))

	matchH := handlers.NewMatchHandler(rdb, store)
	streamH := handlers.NewStreamHandler(rdb)
	historyH := handlers.NewHistoryHandler(store)

	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok", "service": "web-server"})
	})

	// Live match state (reads Redis)
	app.Get("/matches/:match_id", matchH.GetMatch)

	// Real-time SSE stream (Redis Pub/Sub)
	app.Get("/matches/:match_id/stream", streamH.Stream)

	// Historical statistics (reads PostgreSQL replica)
	app.Get("/matches/:match_id/statistic", historyH.GetMatchStatistic)
	app.Get("/teams/:team_id/history", historyH.GetTeamHistory)
	app.Get("/players/:player_id/history", historyH.GetPlayerHistory)

	log.Printf("Web Server listening on :%s", port)
	log.Fatal(app.Listen(":" + port))
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
