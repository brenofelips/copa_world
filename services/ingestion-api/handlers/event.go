package handlers

import (
	"context"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"ingestion-api/kafka"
	"ingestion-api/models"
)

type EventHandler struct {
	producer *kafka.Producer
}

func NewEventHandler(producer *kafka.Producer) *EventHandler {
	return &EventHandler{producer: producer}
}

func (h *EventHandler) HandleEvent(c *fiber.Ctx) error {
	var ev models.ProviderEvent
	if err := c.BodyParser(&ev); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid JSON: " + err.Error()})
	}

	if ev.Match == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "match is required"})
	}
	if ev.Event == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "event is required"})
	}

	teamACode, teamBCode := extractTeamCodes(ev.Match)

	normalized := models.NormalizedEvent{
		EventID:          uuid.New().String(),
		ExternalEventID:  ev.ID,
		MatchID:          ev.Match,
		TeamA:            ev.TeamA,
		TeamB:            ev.TeamB,
		TeamACode:        teamACode,
		TeamBCode:        teamBCode,
		CompetitionTitle: ev.Competition.Title,
		CompetitionStage: ev.Competition.Stage,
		EventType:        ev.Event,
		Minute:           ev.Minute,
		Sequence:         ev.Sequence,
		Payload:          ev.Payload,
		ReceivedAt:       time.Now().UTC(),
		Source:           "ingestion-api",
	}
	if normalized.Payload == nil {
		normalized.Payload = map[string]interface{}{}
	}

	if err := h.producer.Publish(context.Background(), normalized.MatchID, normalized); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to publish event: " + err.Error()})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"event_id":  normalized.EventID,
		"match_id":  normalized.MatchID,
		"event_type": normalized.EventType,
		"status":    "published",
	})
}

// extractTeamCodes parses "BRA-MAR-13-06-2026" → ("BRA", "MAR").
func extractTeamCodes(matchID string) (string, string) {
	parts := strings.SplitN(matchID, "-", 3)
	if len(parts) >= 2 {
		return parts[0], parts[1]
	}
	return "", ""
}
