package poller

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"data-provider/apifootball"
	dataredis "data-provider/redis"
)

var terminalStatuses = map[string]bool{
	"FT":  true,
	"AET": true,
	"PEN": true,
}

type Poller struct {
	api          *apifootball.Client
	rdb          *dataredis.Client
	ingestionURL string
	interval     time.Duration
	httpClient   *http.Client
}

func New(api *apifootball.Client, rdb *dataredis.Client, ingestionURL string, interval time.Duration) *Poller {
	return &Poller{
		api:          api,
		rdb:          rdb,
		ingestionURL: ingestionURL,
		interval:     interval,
		httpClient:   &http.Client{Timeout: 10 * time.Second},
	}
}

func (p *Poller) Run(ctx context.Context) {
	log.Printf("Poller running, interval=%s, ingestion_url=%s", p.interval, p.ingestionURL)
	p.poll(ctx)
	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			p.poll(ctx)
		}
	}
}

func (p *Poller) poll(ctx context.Context) {
	fixtures, err := p.api.GetLiveFixtures(ctx)
	if err != nil {
		log.Printf("poller: fetch live fixtures error: %v", err)
		return
	}
	log.Printf("poller: %d live fixture(s)", len(fixtures))
	for _, f := range fixtures {
		if err := p.processFixture(ctx, f); err != nil {
			log.Printf("poller: fixture %d error: %v", f.Fixture.ID, err)
		}
	}
}

func (p *Poller) processFixture(ctx context.Context, fixture apifootball.FixtureData) error {
	fixtureID := fixture.Fixture.ID
	currentStatus := fixture.Fixture.Status.Short
	matchID := buildMatchID(fixture)

	prevStatus, err := p.rdb.GetStatus(ctx, fixtureID)
	if err != nil {
		return fmt.Errorf("get status: %w", err)
	}

	if err := p.handleStatusTransition(ctx, fixture, matchID, prevStatus, currentStatus); err != nil {
		log.Printf("poller: fixture %d status transition error: %v", fixtureID, err)
	}

	if prevStatus != currentStatus {
		if err := p.rdb.SetStatus(ctx, fixtureID, currentStatus); err != nil {
			log.Printf("poller: fixture %d set status error: %v", fixtureID, err)
		}
	}

	events, err := p.api.GetFixtureEvents(ctx, fixtureID)
	if err != nil {
		return fmt.Errorf("fetch events: %w", err)
	}

	cursor, err := p.rdb.GetCursor(ctx, fixtureID)
	if err != nil {
		return fmt.Errorf("get cursor: %w", err)
	}

	newCursor := cursor
	for i := cursor + 1; i < len(events); i++ {
		if err := p.processGameEvent(ctx, fixture, matchID, events[i], i); err != nil {
			log.Printf("poller: fixture %d event %d error: %v", fixtureID, i, err)
			break
		}
		newCursor = i
	}

	if newCursor != cursor {
		if err := p.rdb.SetCursor(ctx, fixtureID, newCursor); err != nil {
			log.Printf("poller: fixture %d set cursor error: %v", fixtureID, err)
		}
	}

	return nil
}

func (p *Poller) handleStatusTransition(ctx context.Context, fixture apifootball.FixtureData, matchID, prevStatus, currentStatus string) error {
	if prevStatus == currentStatus {
		return nil
	}

	fixtureID := fixture.Fixture.ID
	elapsed := fixture.Fixture.Status.Elapsed

	switch {
	case prevStatus == "":
		switch currentStatus {
		case "1H", "2H", "ET":
			return p.postStatusEvent(ctx, fixture, matchID, "MATCH_STARTED", 0,
				fmt.Sprintf("%d-MATCH_STARTED", fixtureID))
		case "HT":
			if err := p.postStatusEvent(ctx, fixture, matchID, "MATCH_STARTED", 0,
				fmt.Sprintf("%d-MATCH_STARTED", fixtureID)); err != nil {
				return err
			}
			return p.postStatusEvent(ctx, fixture, matchID, "HALF_TIME", 45,
				fmt.Sprintf("%d-HALF_TIME", fixtureID))
		}

	case currentStatus == "HT":
		return p.postStatusEvent(ctx, fixture, matchID, "HALF_TIME", 45,
			fmt.Sprintf("%d-HALF_TIME", fixtureID))

	case terminalStatuses[currentStatus] && !terminalStatuses[prevStatus]:
		return p.postStatusEvent(ctx, fixture, matchID, "MATCH_ENDED", elapsed,
			fmt.Sprintf("%d-MATCH_ENDED", fixtureID))
	}

	return nil
}

func (p *Poller) postStatusEvent(ctx context.Context, fixture apifootball.FixtureData, matchID, eventType string, minute int, eventID string) error {
	ev := providerEvent{
		ID:        eventID,
		Match:     matchID,
		TeamA:     fixture.Teams.Home.Name,
		TeamB:     fixture.Teams.Away.Name,
		TeamALogo: fixture.Teams.Home.Logo,
		TeamBLogo: fixture.Teams.Away.Logo,
		Competition: competition{
			Title: "world-cup-2026",
			Stage: normalizeRound(fixture.League.Round),
		},
		Event:    eventType,
		Minute:   minute,
		Sequence: 0,
		Payload:  map[string]interface{}{},
	}
	return p.postEvent(ctx, ev)
}

func (p *Poller) processGameEvent(ctx context.Context, fixture apifootball.FixtureData, matchID string, event apifootball.EventData, index int) error {
	eventType, payload := mapEventType(event)
	if eventType == "" {
		return nil
	}

	ev := providerEvent{
		ID:        fmt.Sprintf("%d-%d", fixture.Fixture.ID, index),
		Match:     matchID,
		TeamA:     fixture.Teams.Home.Name,
		TeamB:     fixture.Teams.Away.Name,
		TeamALogo: fixture.Teams.Home.Logo,
		TeamBLogo: fixture.Teams.Away.Logo,
		Competition: competition{
			Title: "world-cup-2026",
			Stage: normalizeRound(fixture.League.Round),
		},
		Event:    eventType,
		Minute:   event.Time.Elapsed,
		Sequence: index + 1,
		Payload:  payload,
	}
	return p.postEvent(ctx, ev)
}

func (p *Poller) postEvent(ctx context.Context, ev providerEvent) error {
	body, err := json.Marshal(ev)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", p.ingestionURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("ingestion-api returned %d for event %s", resp.StatusCode, ev.ID)
	}

	log.Printf("poller: posted event id=%s type=%s match=%s", ev.ID, ev.Event, ev.Match)
	return nil
}

type providerEvent struct {
	ID          string                 `json:"id"`
	Match       string                 `json:"match"`
	TeamA       string                 `json:"team_A"`
	TeamB       string                 `json:"team_B"`
	TeamALogo   string                 `json:"team_a_logo"`
	TeamBLogo   string                 `json:"team_b_logo"`
	Competition competition            `json:"competition"`
	Event       string                 `json:"event"`
	Minute      int                    `json:"minute"`
	Sequence    int                    `json:"sequence"`
	Payload     map[string]interface{} `json:"payload"`
}

type competition struct {
	Title string `json:"title"`
	Stage string `json:"stage"`
}

func mapEventType(event apifootball.EventData) (string, map[string]interface{}) {
	teamID := teamCode(event.Team.Name)
	playerIDStr := ""
	if event.Player.ID != nil {
		playerIDStr = fmt.Sprintf("%d", *event.Player.ID)
	}

	switch strings.ToLower(event.Type) {
	case "goal":
		payload := map[string]interface{}{
			"team_id":     teamID,
			"player_id":   playerIDStr,
			"player_name": event.Player.Name,
		}
		if event.Assist.Name != "" {
			payload["assist_name"] = event.Assist.Name
		}
		return "GOAL", payload

	case "card":
		payload := map[string]interface{}{
			"team_id":     teamID,
			"player_id":   playerIDStr,
			"player_name": event.Player.Name,
		}
		switch strings.ToLower(event.Detail) {
		case "yellow card":
			return "YELLOW_CARD", payload
		case "red card":
			return "RED_CARD", payload
		}

	case "subst":
		assistIDStr := ""
		if event.Assist.ID != nil {
			assistIDStr = fmt.Sprintf("%d", *event.Assist.ID)
		}
		return "SUBSTITUTION", map[string]interface{}{
			"player_out_id":   playerIDStr,
			"player_out_name": event.Player.Name,
			"player_in_id":    assistIDStr,
			"player_in_name":  event.Assist.Name,
		}
	}

	return "", nil
}

func buildMatchID(fixture apifootball.FixtureData) string {
	homeCode := teamCode(fixture.Teams.Home.Name)
	awayCode := teamCode(fixture.Teams.Away.Name)

	t, err := time.Parse(time.RFC3339, fixture.Fixture.Date)
	if err != nil {
		t = time.Now().UTC()
	}
	t = t.UTC()

	return fmt.Sprintf("%s-%s-%02d-%02d-%d", homeCode, awayCode, t.Day(), int(t.Month()), t.Year())
}

func teamCode(name string) string {
	name = strings.ToUpper(strings.TrimSpace(name))
	if len(name) >= 3 {
		return name[:3]
	}
	return name
}

// normalizeRound converts "Group Stage - 2" → "group_stage", "Round of 16" → "round_of_16"
func normalizeRound(round string) string {
	parts := strings.SplitN(round, " - ", 2)
	s := strings.TrimSpace(parts[0])
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, " ", "_")
	s = strings.ReplaceAll(s, "-", "_")
	return s
}
