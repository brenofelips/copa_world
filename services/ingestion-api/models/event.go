package models

import "time"

type EventType string

const (
	Scheduled    EventType = "SCHEDULED"
	MatchStarted EventType = "MATCH_STARTED"
	Goal         EventType = "GOAL"
	RedCard      EventType = "RED_CARD"
	YellowCard   EventType = "YELLOW_CARD"
	VARDecision  EventType = "VAR_DECISION"
	CornerKick   EventType = "CORNER_KICK"
	Penalty      EventType = "PENALTY"
	Substitution EventType = "SUBSTITUTION"
	MatchEnded   EventType = "MATCH_ENDED"
)

// ProviderEvent is the format sent by external data providers.
type ProviderEvent struct {
	ID          string                 `json:"id"`
	Match       string                 `json:"match"`
	TeamA       string                 `json:"team_A"`
	TeamB       string                 `json:"team_B"`
	TeamALogo   string                 `json:"team_a_logo"`
	TeamBLogo   string                 `json:"team_b_logo"`
	Competition Competition            `json:"competition"`
	Event       EventType              `json:"event"`
	Minute      int                    `json:"minute"`
	Sequence    int                    `json:"sequence"`
	Payload     map[string]interface{} `json:"payload"`
}

type Competition struct {
	Title string `json:"title"`
	Stage string `json:"stage"`
}

// NormalizedEvent is the canonical format published to Kafka.
type NormalizedEvent struct {
	EventID          string                 `json:"event_id"`
	ExternalEventID  string                 `json:"external_event_id"`
	MatchID          string                 `json:"match_id"`
	TeamA            string                 `json:"team_a"`
	TeamB            string                 `json:"team_b"`
	TeamACode        string                 `json:"team_a_code"`
	TeamBCode        string                 `json:"team_b_code"`
	TeamALogo        string                 `json:"team_a_logo"`
	TeamBLogo        string                 `json:"team_b_logo"`
	CompetitionTitle string                 `json:"competition_title"`
	CompetitionStage string                 `json:"competition_stage"`
	EventType        EventType              `json:"event_type"`
	Minute           int                    `json:"minute"`
	Sequence         int                    `json:"sequence"`
	Payload          map[string]interface{} `json:"payload"`
	ReceivedAt       time.Time              `json:"received_at"`
	Source           string                 `json:"source"`
}
