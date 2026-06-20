package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const matchKeyTTL = 24 * time.Hour

type Client struct {
	rdb *redis.Client
}

type MatchState struct {
	MatchID          string    `json:"match_id"`
	TeamACode        string    `json:"team_a_code"`
	TeamBCode        string    `json:"team_b_code"`
	TeamA            string    `json:"team_a"`
	TeamB            string    `json:"team_b"`
	ScoreA           int       `json:"score_a"`
	ScoreB           int       `json:"score_b"`
	Status           string    `json:"status"`
	Minute           int       `json:"minute"`
	CompetitionTitle string    `json:"competition_title"`
	CompetitionStage string    `json:"competition_stage"`
	LastEvent        string    `json:"last_event"`
	LastUpdated      time.Time `json:"last_updated"`
}

func NewClient(addr string) *Client {
	rdb := redis.NewClient(&redis.Options{
		Addr:         addr,
		PoolSize:     20,
		MinIdleConns: 5,
	})
	return &Client{rdb: rdb}
}

func (c *Client) GetMatchState(ctx context.Context, matchID string) (*MatchState, error) {
	key := fmt.Sprintf("match:%s", matchID)
	data, err := c.rdb.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("redis get match state: %w", err)
	}
	var state MatchState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("unmarshal match state: %w", err)
	}
	return &state, nil
}

func (c *Client) SetMatchState(ctx context.Context, state *MatchState) error {
	key := fmt.Sprintf("match:%s", state.MatchID)
	data, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("marshal match state: %w", err)
	}
	return c.rdb.Set(ctx, key, data, matchKeyTTL).Err()
}

// PublishEvent broadcasts the updated match state to all SSE subscribers.
func (c *Client) PublishEvent(ctx context.Context, matchID string, state *MatchState) error {
	channel := fmt.Sprintf("match-events:%s", matchID)
	data, err := json.Marshal(state)
	if err != nil {
		return err
	}
	return c.rdb.Publish(ctx, channel, data).Err()
}
