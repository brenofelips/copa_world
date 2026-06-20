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

type NormalizedEvent struct {
	EventID          string                 `json:"event_id"`
	ExternalEventID  string                 `json:"external_event_id"`
	MatchID          string                 `json:"match_id"`
	TeamA            string                 `json:"team_a"`
	TeamB            string                 `json:"team_b"`
	TeamACode        string                 `json:"team_a_code"`
	TeamBCode        string                 `json:"team_b_code"`
	CompetitionTitle string                 `json:"competition_title"`
	CompetitionStage string                 `json:"competition_stage"`
	EventType        EventType              `json:"event_type"`
	Minute           int                    `json:"minute"`
	Sequence         int                    `json:"sequence"`
	Payload          map[string]interface{} `json:"payload"`
	ReceivedAt       time.Time              `json:"received_at"`
	Source           string                 `json:"source"`
}
