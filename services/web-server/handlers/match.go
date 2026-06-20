package handlers

import (
	"github.com/gofiber/fiber/v2"
	"web-server/db"
	"web-server/redis"
)

type MatchHandler struct {
	rdb   *redis.Client
	store *db.Store
}

func NewMatchHandler(rdb *redis.Client, store *db.Store) *MatchHandler {
	return &MatchHandler{rdb: rdb, store: store}
}

// GET /matches/:match_id
// Reads live state from Redis; falls back to PostgreSQL for finished matches.
func (h *MatchHandler) GetMatch(c *fiber.Ctx) error {
	matchID := c.Params("match_id")
	ctx := c.Context()

	state, err := h.rdb.GetMatchState(ctx, matchID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	if state != nil {
		return c.JSON(state)
	}

	// Fall back to PostgreSQL (match finished or Redis was restarted)
	match, err := h.store.GetMatchByID(ctx, matchID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "match not found"})
	}
	return c.JSON(match)
}
