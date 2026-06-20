package poller

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
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
	api              *apifootball.Client
	rdb              *dataredis.Client
	ingestionURL     string
	interval         time.Duration
	scheduleInterval time.Duration
	season           int
	httpClient       *http.Client
}

func New(api *apifootball.Client, rdb *dataredis.Client, ingestionURL string, interval time.Duration, scheduleInterval time.Duration, season int) *Poller {
	return &Poller{
		api:              api,
		rdb:              rdb,
		ingestionURL:     ingestionURL,
		interval:         interval,
		scheduleInterval: scheduleInterval,
		season:           season,
		httpClient:       &http.Client{Timeout: 10 * time.Second},
	}
}

func (p *Poller) competitionTitle() string {
	return fmt.Sprintf("world-cup-%d", p.season)
}

func (p *Poller) Run(ctx context.Context) {
	log.Printf("Poller running, interval=%s, schedule_interval=%s, ingestion_url=%s",
		p.interval, p.scheduleInterval, p.ingestionURL)

	// Fetch full day schedule on startup, then poll live games.
	p.pollSchedule(ctx)
	p.poll(ctx)

	liveTicker := time.NewTicker(p.interval)
	scheduleTicker := time.NewTicker(p.scheduleInterval)
	defer liveTicker.Stop()
	defer scheduleTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-liveTicker.C:
			p.poll(ctx)
		case <-scheduleTicker.C:
			p.pollSchedule(ctx)
		}
	}
}

// pollSchedule fetches all of today's fixtures and sends SCHEDULED events for
// upcoming games and MATCH_ENDED (with final score) for games we missed entirely.
// Requires a paid API plan; on free plans it logs a warning and returns early.
func (p *Poller) pollSchedule(ctx context.Context) {
	today := time.Now().UTC().Format("2006-01-02")
	fixtures, err := p.api.GetFixturesByDate(ctx, today)
	if err != nil {
		if errors.Is(err, apifootball.ErrAPIRestricted) {
			log.Printf("poller: schedule poll skipped — upgrade API plan to enable schedule fetching (live polling still active)")
			return
		}
		log.Printf("poller: fetch schedule error: %v", err)
		return
	}
	log.Printf("poller: %d fixture(s) in today's schedule", len(fixtures))

	for _, f := range fixtures {
		status := f.Fixture.Status.Short
		fixtureID := f.Fixture.ID
		matchID := buildMatchID(f)

		prevStatus, err := p.rdb.GetStatus(ctx, fixtureID)
		if err != nil {
			log.Printf("poller: schedule: get status fixture %d: %v", fixtureID, err)
			continue
		}

		// Already handled as a terminal state — skip.
		if terminalStatuses[prevStatus] {
			continue
		}

		switch {
		case status == "NS" || status == "TBD":
			// Send SCHEDULED once, unless we already recorded it.
			if prevStatus != "NS" && prevStatus != "TBD" {
				if err := p.postStatusEvent(ctx, f, matchID, "SCHEDULED", 0,
					fmt.Sprintf("%d-SCHEDULED", fixtureID)); err != nil {
					log.Printf("poller: schedule: post scheduled fixture %d: %v", fixtureID, err)
					continue
				}
				_ = p.rdb.SetStatus(ctx, fixtureID, "NS")
			}

		case terminalStatuses[status]:
			// Game is finished but we either never saw it or tracked it live.
			if prevStatus == "" || prevStatus == "NS" || prevStatus == "TBD" {
				// Missed the entire game — synthesize MATCH_STARTED + MATCH_ENDED with score.
				_ = p.postStatusEvent(ctx, f, matchID, "MATCH_STARTED", 0,
					fmt.Sprintf("%d-MATCH_STARTED", fixtureID))

				scoreA := 0
				if f.Goals.Home != nil {
					scoreA = *f.Goals.Home
				}
				scoreB := 0
				if f.Goals.Away != nil {
					scoreB = *f.Goals.Away
				}
				ev := providerEvent{
					ID:        fmt.Sprintf("%d-MATCH_ENDED", fixtureID),
					Match:     matchID,
					TeamA:     f.Teams.Home.Name,
					TeamB:     f.Teams.Away.Name,
					TeamALogo: f.Teams.Home.Logo,
					TeamBLogo: f.Teams.Away.Logo,
					Competition: competition{
						Title: p.competitionTitle(),
						Stage: normalizeRound(f.League.Round),
					},
					Event:    "MATCH_ENDED",
					Minute:   f.Fixture.Status.Elapsed,
					Sequence: 0,
					Payload: map[string]interface{}{
						"final_score_a": scoreA,
						"final_score_b": scoreB,
					},
				}
				if err := p.postEvent(ctx, ev); err != nil {
					log.Printf("poller: schedule: post ended fixture %d: %v", fixtureID, err)
					continue
				}
				_ = p.rdb.SetStatus(ctx, fixtureID, status)

			} else if !terminalStatuses[prevStatus] {
				// Was tracked as live, now finished — normal transition.
				if err := p.postStatusEvent(ctx, f, matchID, "MATCH_ENDED", f.Fixture.Status.Elapsed,
					fmt.Sprintf("%d-MATCH_ENDED", fixtureID)); err != nil {
					log.Printf("poller: schedule: post ended fixture %d: %v", fixtureID, err)
					continue
				}
				_ = p.rdb.SetStatus(ctx, fixtureID, status)
			}
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

	// "NS" and "TBD" are pre-match states written by the schedule poller;
	// treat them the same as an empty (unseen) previous status.
	noLiveHistory := prevStatus == "" || prevStatus == "NS" || prevStatus == "TBD"

	switch {
	case noLiveHistory:
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

	case currentStatus == "HT" && !noLiveHistory:
		return p.postStatusEvent(ctx, fixture, matchID, "HALF_TIME", 45,
			fmt.Sprintf("%d-HALF_TIME", fixtureID))

	case terminalStatuses[currentStatus] && !terminalStatuses[prevStatus] && !noLiveHistory:
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
			Title: p.competitionTitle(),
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
			Title: p.competitionTitle(),
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
