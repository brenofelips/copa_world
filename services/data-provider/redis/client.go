package redis

import (
	"context"
	"fmt"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

const ttl = 48 * time.Hour

type Client struct {
	rdb *goredis.Client
}

func NewClient(addr string) *Client {
	rdb := goredis.NewClient(&goredis.Options{
		Addr:         addr,
		PoolSize:     10,
		MinIdleConns: 2,
	})
	return &Client{rdb: rdb}
}

// GetCursor returns the index of the last processed event for a fixture.
// Returns -1 if no cursor is stored.
func (c *Client) GetCursor(ctx context.Context, fixtureID int) (int, error) {
	key := fmt.Sprintf("data-provider:cursor:%d", fixtureID)
	val, err := c.rdb.Get(ctx, key).Int()
	if err == goredis.Nil {
		return -1, nil
	}
	return val, err
}

func (c *Client) SetCursor(ctx context.Context, fixtureID, cursor int) error {
	key := fmt.Sprintf("data-provider:cursor:%d", fixtureID)
	return c.rdb.Set(ctx, key, cursor, ttl).Err()
}

// GetStatus returns the last known API-Football status for a fixture.
// Returns "" if no status is stored.
func (c *Client) GetStatus(ctx context.Context, fixtureID int) (string, error) {
	key := fmt.Sprintf("data-provider:status:%d", fixtureID)
	val, err := c.rdb.Get(ctx, key).Result()
	if err == goredis.Nil {
		return "", nil
	}
	return val, err
}

func (c *Client) SetStatus(ctx context.Context, fixtureID int, status string) error {
	key := fmt.Sprintf("data-provider:status:%d", fixtureID)
	return c.rdb.Set(ctx, key, status, ttl).Err()
}

// GetDailyAPICount returns today's API-Football request count (UTC day).
func (c *Client) GetDailyAPICount(ctx context.Context) (int64, error) {
	key := dailyCountKey()
	count, err := c.rdb.Get(ctx, key).Int64()
	if err == goredis.Nil {
		return 0, nil
	}
	return count, err
}

// IncrDailyAPICount atomically increments today's request counter and returns the new value.
func (c *Client) IncrDailyAPICount(ctx context.Context) (int64, error) {
	key := dailyCountKey()
	count, err := c.rdb.Incr(ctx, key).Result()
	if err != nil {
		return 0, err
	}
	if count == 1 {
		// First call today — set a 48h TTL so the key self-cleans.
		c.rdb.Expire(ctx, key, 48*time.Hour)
	}
	return count, nil
}

func dailyCountKey() string {
	return fmt.Sprintf("data-provider:api-requests:%s", time.Now().UTC().Format("2006-01-02"))
}
