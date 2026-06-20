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
