package consumer

import (
	"context"
	"encoding/json"
	"log"
	"strings"
	"time"

	"github.com/segmentio/kafka-go"
	"score-service/models"
	"score-service/redis"
)

type Consumer struct {
	reader *kafka.Reader
	rdb    *redis.Client
}

func New(brokers, groupID string, rdb *redis.Client) *Consumer {
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        strings.Split(brokers, ","),
		Topic:          "match-events",
		GroupID:        groupID,
		MinBytes:       1,
		MaxBytes:       10e6,
		CommitInterval: time.Second,
		StartOffset:    kafka.FirstOffset,
	})
	return &Consumer{reader: r, rdb: rdb}
}

func (c *Consumer) Run(ctx context.Context) error {
	defer c.reader.Close()
	log.Println("Score consumer running...")

	for {
		msg, err := c.reader.ReadMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			log.Printf("score consumer: read error: %v", err)
			time.Sleep(time.Second)
			continue
		}

		var event models.NormalizedEvent
		if err := json.Unmarshal(msg.Value, &event); err != nil {
			log.Printf("score consumer: unmarshal error: %v", err)
			continue
		}

		if err := c.process(ctx, &event); err != nil {
			log.Printf("score consumer: process event %s error: %v", event.EventID, err)
		}
	}
}

func (c *Consumer) process(ctx context.Context, event *models.NormalizedEvent) error {
	state, err := c.rdb.GetMatchState(ctx, event.MatchID)
	if err != nil {
		return err
	}
	if state == nil {
		state = &redis.MatchState{
			MatchID:          event.MatchID,
			TeamACode:        event.TeamACode,
			TeamBCode:        event.TeamBCode,
			TeamA:            event.TeamA,
			TeamB:            event.TeamB,
			CompetitionTitle: event.CompetitionTitle,
			CompetitionStage: event.CompetitionStage,
			Status:           "SCHEDULED",
		}
	}

	state.Minute = event.Minute
	state.LastEvent = string(event.EventType)
	state.LastUpdated = time.Now().UTC()

	// Always refresh logos when the event carries them.
	if event.TeamALogo != "" {
		state.TeamALogo = event.TeamALogo
	}
	if event.TeamBLogo != "" {
		state.TeamBLogo = event.TeamBLogo
	}

	switch event.EventType {
	case models.Scheduled:
		state.Status = "SCHEDULED"
		state.TeamACode = event.TeamACode
		state.TeamBCode = event.TeamBCode
		state.TeamA = event.TeamA
		state.TeamB = event.TeamB
		state.CompetitionTitle = event.CompetitionTitle
		state.CompetitionStage = event.CompetitionStage

	case models.MatchStarted:
		state.Status = "LIVE"
		state.ScoreA = 0
		state.ScoreB = 0
		state.TeamACode = event.TeamACode
		state.TeamBCode = event.TeamBCode
		state.TeamA = event.TeamA
		state.TeamB = event.TeamB

	case models.Goal:
		teamID, _ := event.Payload["team_id"].(string)
		if teamID == state.TeamACode {
			state.ScoreA++
		} else if teamID == state.TeamBCode {
			state.ScoreB++
		}

	case models.MatchEnded:
		state.Status = "ENDED"
		// Use final score from payload for games missed entirely (from schedule poll).
		if finalA, ok := event.Payload["final_score_a"].(float64); ok {
			state.ScoreA = int(finalA)
		}
		if finalB, ok := event.Payload["final_score_b"].(float64); ok {
			state.ScoreB = int(finalB)
		}
	}

	if err := c.rdb.SetMatchState(ctx, state); err != nil {
		return err
	}
	return c.rdb.PublishEvent(ctx, event.MatchID, state)
}
