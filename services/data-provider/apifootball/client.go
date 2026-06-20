package apifootball

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"
)

const (
	baseURL           = "https://v3.football.api-sports.io"
	DailyRequestLimit = 100
)

// ErrDailyLimitReached is returned when the 100 req/day cap is hit.
var ErrDailyLimitReached = errors.New("daily API request limit reached")

// APICounter tracks the number of requests made to the external API today.
// Implemented by *redis.Client in the main package.
type APICounter interface {
	GetDailyAPICount(ctx context.Context) (int64, error)
	IncrDailyAPICount(ctx context.Context) (int64, error)
}

type Client struct {
	apiKey     string
	leagueID   int
	season     int
	httpClient *http.Client
	counter    APICounter
}

func NewClient(apiKey string, leagueID int, season int, counter APICounter) *Client {
	return &Client{
		apiKey:     apiKey,
		leagueID:   leagueID,
		season:     season,
		httpClient: &http.Client{Timeout: 15 * time.Second},
		counter:    counter,
	}
}

// checkLimit returns ErrDailyLimitReached when the 100 req/day cap is hit.
// On Redis errors it logs and allows the request through to avoid blocking.
func (c *Client) checkLimit(ctx context.Context) error {
	if c.counter == nil {
		return nil
	}
	current, err := c.counter.GetDailyAPICount(ctx)
	if err != nil {
		log.Printf("apifootball: rate-limit check error: %v (proceeding)", err)
		return nil
	}
	if current >= DailyRequestLimit {
		return fmt.Errorf("%w (%d/%d used today)", ErrDailyLimitReached, current, DailyRequestLimit)
	}
	newCount, err := c.counter.IncrDailyAPICount(ctx)
	if err != nil {
		log.Printf("apifootball: rate-limit increment error: %v (proceeding)", err)
		return nil
	}
	log.Printf("apifootball: daily request %d/%d", newCount, DailyRequestLimit)
	return nil
}

// ErrAPIRestricted is returned when the API returns a plan/access error.
var ErrAPIRestricted = errors.New("API access restricted for plan/season")

type FixtureResponse struct {
	Errors   interface{}   `json:"errors"`
	Results  int           `json:"results"`
	Response []FixtureData `json:"response"`
}

type FixtureData struct {
	Fixture FixtureInfo `json:"fixture"`
	League  LeagueInfo  `json:"league"`
	Teams   TeamsInfo   `json:"teams"`
	Goals   GoalsInfo   `json:"goals"`
}

type FixtureInfo struct {
	ID     int        `json:"id"`
	Date   string     `json:"date"`
	Status StatusInfo `json:"status"`
}

type StatusInfo struct {
	Short   string `json:"short"`
	Elapsed int    `json:"elapsed"`
}

type LeagueInfo struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Round string `json:"round"`
}

type TeamsInfo struct {
	Home TeamInfo `json:"home"`
	Away TeamInfo `json:"away"`
}

type TeamInfo struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Logo string `json:"logo"`
}

type GoalsInfo struct {
	Home *int `json:"home"`
	Away *int `json:"away"`
}

type EventsResponse struct {
	Response []EventData `json:"response"`
}

type EventData struct {
	Time   TimeInfo   `json:"time"`
	Team   TeamInfo   `json:"team"`
	Player PlayerInfo `json:"player"`
	Assist PlayerInfo `json:"assist"`
	Type   string     `json:"type"`
	Detail string     `json:"detail"`
}

type TimeInfo struct {
	Elapsed int  `json:"elapsed"`
	Extra   *int `json:"extra"`
}

type PlayerInfo struct {
	ID   *int   `json:"id"`
	Name string `json:"name"`
}

// GetLiveFixtures returns all currently live fixtures for the configured league.
// Note: the free API plan does not support filtering by season on this endpoint,
// so we omit the season parameter intentionally.
func (c *Client) GetLiveFixtures(ctx context.Context) ([]FixtureData, error) {
	if err := c.checkLimit(ctx); err != nil {
		return nil, err
	}
	url := fmt.Sprintf("%s/fixtures?live=all&league=%d", baseURL, c.leagueID)
	return c.fetchFixtures(ctx, url)
}

// GetFixturesByDate returns all fixtures for a given date (YYYY-MM-DD).
// Requires a paid API plan that allows season-based queries.
func (c *Client) GetFixturesByDate(ctx context.Context, date string) ([]FixtureData, error) {
	if err := c.checkLimit(ctx); err != nil {
		return nil, err
	}
	url := fmt.Sprintf("%s/fixtures?league=%d&season=%d&date=%s", baseURL, c.leagueID, c.season, date)
	return c.fetchFixtures(ctx, url)
}

func (c *Client) fetchFixtures(ctx context.Context, url string) ([]FixtureData, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("x-apisports-key", c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result FixtureResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	// Check for API-level errors (plan restrictions, invalid params, etc.)
	if result.Errors != nil {
		switch v := result.Errors.(type) {
		case map[string]interface{}:
			if len(v) > 0 {
				for key, msg := range v {
					log.Printf("apifootball: API error [%s]: %v", key, msg)
				}
				return nil, fmt.Errorf("%w: %v", ErrAPIRestricted, result.Errors)
			}
		}
	}

	return result.Response, nil
}

func (c *Client) GetFixtureEvents(ctx context.Context, fixtureID int) ([]EventData, error) {
	if err := c.checkLimit(ctx); err != nil {
		return nil, err
	}
	url := fmt.Sprintf("%s/fixtures/events?fixture=%d", baseURL, fixtureID)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("x-apisports-key", c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result EventsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return result.Response, nil
}

func (c *Client) LeagueID() int { return c.leagueID }
func (c *Client) Season() int   { return c.season }
