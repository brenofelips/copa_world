package models

import "time"

// MatchState is the live state stored/read from Redis.
type MatchState struct {
	MatchID          string    `json:"match_id"`
	TeamACode        string    `json:"team_a_code"`
	TeamBCode        string    `json:"team_b_code"`
	TeamA            string    `json:"team_a"`
	TeamB            string    `json:"team_b"`
	TeamALogo        string    `json:"team_a_logo"`
	TeamBLogo        string    `json:"team_b_logo"`
	ScoreA           int       `json:"score_a"`
	ScoreB           int       `json:"score_b"`
	Status           string    `json:"status"`
	Minute           int       `json:"minute"`
	CompetitionTitle string    `json:"competition_title"`
	CompetitionStage string    `json:"competition_stage"`
	LastEvent        string    `json:"last_event"`
	LastUpdated      time.Time `json:"last_updated"`
}

// MatchStatistic aggregates event counts for a match.
type MatchStatistic struct {
	MatchID      string        `json:"match_id"`
	Goals        []GoalEvent   `json:"goals"`
	YellowCards  []CardEvent   `json:"yellow_cards"`
	RedCards     []CardEvent   `json:"red_cards"`
	Corners      int           `json:"corners"`
	Penalties    []PenaltyEvent `json:"penalties"`
	VARDecisions []VAREvent    `json:"var_decisions"`
	Substitutions []SubEvent   `json:"substitutions"`
	TotalEvents  int           `json:"total_events"`
}

type GoalEvent struct {
	Minute    int    `json:"minute"`
	TeamID    string `json:"team_id"`
	PlayerID  string `json:"player_id"`
	PlayerName string `json:"player_name"`
}

type CardEvent struct {
	Minute    int    `json:"minute"`
	TeamID    string `json:"team_id"`
	PlayerID  string `json:"player_id"`
	PlayerName string `json:"player_name"`
}

type PenaltyEvent struct {
	Minute   int    `json:"minute"`
	TeamID   string `json:"team_id"`
	PlayerID string `json:"player_id"`
}

type VAREvent struct {
	Minute   int    `json:"minute"`
	Decision string `json:"decision"`
	Reason   string `json:"reason"`
}

type SubEvent struct {
	Minute    int    `json:"minute"`
	TeamID    string `json:"team_id"`
	PlayerIn  string `json:"player_in"`
	PlayerOut string `json:"player_out"`
}

// MatchSummary for team/player history.
type MatchSummary struct {
	MatchID          string    `json:"match_id"`
	TeamAID          string    `json:"team_a_id"`
	TeamAName        string    `json:"team_a_name"`
	TeamBID          string    `json:"team_b_id"`
	TeamBName        string    `json:"team_b_name"`
	ScoreA           int       `json:"score_a"`
	ScoreB           int       `json:"score_b"`
	Status           string    `json:"status"`
	CompetitionTitle string    `json:"competition_title"`
	CompetitionStage string    `json:"competition_stage"`
	StartedAt        *time.Time `json:"started_at"`
	EndedAt          *time.Time `json:"ended_at"`
}

type PlayerHistoryEvent struct {
	EventID    string                 `json:"event_id"`
	MatchID    string                 `json:"match_id"`
	EventType  string                 `json:"event_type"`
	Minute     int                    `json:"minute"`
	Payload    map[string]interface{} `json:"payload"`
	ReceivedAt time.Time              `json:"received_at"`
}
