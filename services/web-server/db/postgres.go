package db

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"
	"web-server/models"
)

type Store struct {
	primary *pgxpool.Pool
	replica *pgxpool.Pool
}

func NewStore(ctx context.Context, primaryDSN, replicaDSN string) (*Store, error) {
	primary, err := pgxpool.New(ctx, primaryDSN)
	if err != nil {
		return nil, fmt.Errorf("primary pool: %w", err)
	}
	if err := primary.Ping(ctx); err != nil {
		return nil, fmt.Errorf("primary ping: %w", err)
	}

	// Replica is optional; fall back to primary if unavailable.
	replica, err := pgxpool.New(ctx, replicaDSN)
	if err != nil {
		log.Printf("db: replica unavailable, falling back to primary: %v", err)
		replica = primary
	} else if err := replica.Ping(ctx); err != nil {
		log.Printf("db: replica ping failed, falling back to primary: %v", err)
		replica = primary
	}

	log.Println("Web Server connected to PostgreSQL")
	return &Store{primary: primary, replica: replica}, nil
}

func (s *Store) Close() {
	s.primary.Close()
	if s.replica != s.primary {
		s.replica.Close()
	}
}

// GetMatchStatistic aggregates events for a match from the replica.
func (s *Store) GetMatchStatistic(ctx context.Context, matchID string) (*models.MatchStatistic, error) {
	rows, err := s.replica.Query(ctx, `
		SELECT event_type, minute, payload
		FROM match_events
		WHERE match_id = $1
		ORDER BY sequence ASC, received_at ASC
	`, matchID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	stat := &models.MatchStatistic{MatchID: matchID}

	for rows.Next() {
		var evType string
		var minute int
		var rawPayload []byte

		if err := rows.Scan(&evType, &minute, &rawPayload); err != nil {
			return nil, err
		}

		var payload map[string]interface{}
		_ = json.Unmarshal(rawPayload, &payload)
		str := func(key string) string {
			v, _ := payload[key].(string)
			return v
		}

		stat.TotalEvents++

		switch evType {
		case "GOAL":
			stat.Goals = append(stat.Goals, models.GoalEvent{
				Minute:     minute,
				TeamID:     str("team_id"),
				PlayerID:   str("player_id"),
				PlayerName: str("player_name"),
			})
		case "YELLOW_CARD":
			stat.YellowCards = append(stat.YellowCards, models.CardEvent{
				Minute:     minute,
				TeamID:     str("team_id"),
				PlayerID:   str("player_id"),
				PlayerName: str("player_name"),
			})
		case "RED_CARD":
			stat.RedCards = append(stat.RedCards, models.CardEvent{
				Minute:     minute,
				TeamID:     str("team_id"),
				PlayerID:   str("player_id"),
				PlayerName: str("player_name"),
			})
		case "CORNER_KICK":
			stat.Corners++
		case "PENALTY":
			stat.Penalties = append(stat.Penalties, models.PenaltyEvent{
				Minute:   minute,
				TeamID:   str("team_id"),
				PlayerID: str("player_id"),
			})
		case "VAR_DECISION":
			stat.VARDecisions = append(stat.VARDecisions, models.VAREvent{
				Minute:   minute,
				Decision: str("decision"),
				Reason:   str("reason"),
			})
		case "SUBSTITUTION":
			stat.Substitutions = append(stat.Substitutions, models.SubEvent{
				Minute:    minute,
				TeamID:    str("team_id"),
				PlayerIn:  str("player_in"),
				PlayerOut: str("player_out"),
			})
		}
	}
	return stat, rows.Err()
}

// GetMatchByID fetches a match record from the replica.
func (s *Store) GetMatchByID(ctx context.Context, matchID string) (*models.MatchSummary, error) {
	row := s.replica.QueryRow(ctx, `
		SELECT id, team_a_id, team_a_name, team_b_id, team_b_name,
		       score_a, score_b, status, competition_title, competition_stage,
		       started_at, ended_at
		FROM matches
		WHERE id = $1
	`, matchID)

	var m models.MatchSummary
	err := row.Scan(
		&m.MatchID, &m.TeamAID, &m.TeamAName, &m.TeamBID, &m.TeamBName,
		&m.ScoreA, &m.ScoreB, &m.Status, &m.CompetitionTitle, &m.CompetitionStage,
		&m.StartedAt, &m.EndedAt,
	)
	if err != nil {
		return nil, err
	}
	return &m, nil
}

// GetTeamHistory returns all matches a team participated in.
func (s *Store) GetTeamHistory(ctx context.Context, teamID string) ([]models.MatchSummary, error) {
	rows, err := s.replica.Query(ctx, `
		SELECT id, team_a_id, team_a_name, team_b_id, team_b_name,
		       score_a, score_b, status, competition_title, competition_stage,
		       started_at, ended_at
		FROM matches
		WHERE team_a_id = $1 OR team_b_id = $1
		ORDER BY started_at DESC NULLS LAST
		LIMIT 100
	`, teamID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var matches []models.MatchSummary
	for rows.Next() {
		var m models.MatchSummary
		if err := rows.Scan(
			&m.MatchID, &m.TeamAID, &m.TeamAName, &m.TeamBID, &m.TeamBName,
			&m.ScoreA, &m.ScoreB, &m.Status, &m.CompetitionTitle, &m.CompetitionStage,
			&m.StartedAt, &m.EndedAt,
		); err != nil {
			return nil, err
		}
		matches = append(matches, m)
	}
	return matches, rows.Err()
}

// GetPlayerHistory returns all events involving a player.
func (s *Store) GetPlayerHistory(ctx context.Context, playerID string) ([]models.PlayerHistoryEvent, error) {
	rows, err := s.replica.Query(ctx, `
		SELECT event_id, match_id, event_type, minute, payload, received_at
		FROM match_events
		WHERE payload->>'player_id' = $1
		   OR payload->>'player_in'  = $1
		   OR payload->>'player_out' = $1
		ORDER BY received_at DESC
		LIMIT 200
	`, playerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []models.PlayerHistoryEvent
	for rows.Next() {
		var e models.PlayerHistoryEvent
		var rawPayload []byte
		if err := rows.Scan(&e.EventID, &e.MatchID, &e.EventType, &e.Minute, &rawPayload, &e.ReceivedAt); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(rawPayload, &e.Payload)
		events = append(events, e)
	}
	return events, rows.Err()
}
