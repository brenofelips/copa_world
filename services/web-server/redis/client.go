package redis

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/redis/go-redis/v9"
	"web-server/models"
)

type Client struct {
	rdb *redis.Client
}

func NewClient(addr string) *Client {
	rdb := redis.NewClient(&redis.Options{
		Addr:         addr,
		PoolSize:     50,
		MinIdleConns: 10,
	})
	return &Client{rdb: rdb}
}

func (c *Client) GetLiveMatches(ctx context.Context) ([]*models.MatchState, error) {
	var cursor uint64
	var keys []string
	for {
		batch, next, err := c.rdb.Scan(ctx, cursor, "match:*", 100).Result()
		if err != nil {
			return nil, fmt.Errorf("redis scan: %w", err)
		}
		keys = append(keys, batch...)
		cursor = next
		if cursor == 0 {
			break
		}
	}
	if len(keys) == 0 {
		return []*models.MatchState{}, nil
	}
	pipe := c.rdb.Pipeline()
	cmds := make([]*redis.StringCmd, len(keys))
	for i, key := range keys {
		cmds[i] = pipe.Get(ctx, key)
	}
	if _, err := pipe.Exec(ctx); err != nil && err != redis.Nil {
		return nil, fmt.Errorf("redis pipeline: %w", err)
	}
	var matches []*models.MatchState
	for _, cmd := range cmds {
		data, err := cmd.Bytes()
		if err != nil {
			continue
		}
		var state models.MatchState
		if err := json.Unmarshal(data, &state); err != nil {
			continue
		}
		if state.Status == "SCHEDULED" {
			continue
		}
		matches = append(matches, &state)
	}
	return matches, nil
}

func (c *Client) GetMatchState(ctx context.Context, matchID string) (*models.MatchState, error) {
	key := fmt.Sprintf("match:%s", matchID)
	data, err := c.rdb.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("redis get: %w", err)
	}
	var state models.MatchState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("unmarshal: %w", err)
	}
	return &state, nil
}

// Subscribe returns a PubSub handle for real-time match events.
func (c *Client) Subscribe(ctx context.Context, matchID string) *redis.PubSub {
	channel := fmt.Sprintf("match-events:%s", matchID)
	return c.rdb.Subscribe(ctx, channel)
}

func (c *Client) Ping(ctx context.Context) error {
	return c.rdb.Ping(ctx).Err()
}
