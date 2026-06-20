package handlers

import (
	"github.com/gofiber/fiber/v2"
	"web-server/db"
	"web-server/models"
)

type HistoryHandler struct {
	store *db.Store
}

func NewHistoryHandler(store *db.Store) *HistoryHandler {
	return &HistoryHandler{store: store}
}

// GET /matches/:match_id/statistic
func (h *HistoryHandler) GetMatchStatistic(c *fiber.Ctx) error {
	matchID := c.Params("match_id")
	stat, err := h.store.GetMatchStatistic(c.Context(), matchID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(stat)
}

// GET /teams/:team_id/history
func (h *HistoryHandler) GetTeamHistory(c *fiber.Ctx) error {
	teamID := c.Params("team_id")
	matches, err := h.store.GetTeamHistory(c.Context(), teamID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	if matches == nil {
		matches = make([]models.MatchSummary, 0)
	}
	return c.JSON(fiber.Map{
		"team_id": teamID,
		"matches": matches,
		"total":   len(matches),
	})
}

// GET /players/:player_id/history
func (h *HistoryHandler) GetPlayerHistory(c *fiber.Ctx) error {
	playerID := c.Params("player_id")
	events, err := h.store.GetPlayerHistory(c.Context(), playerID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	if events == nil {
		events = make([]models.PlayerHistoryEvent, 0)
	}
	return c.JSON(fiber.Map{
		"player_id": playerID,
		"events":    events,
		"total":     len(events),
	})
}
