package db

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"
	"event-service/models"
)

type Store struct {
	pool *pgxpool.Pool
}

func NewStore(ctx context.Context, dsn string) (*Store, error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("pgxpool.New: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("db ping: %w", err)
	}
	log.Println("Event Service connected to PostgreSQL")
	return &Store{pool: pool}, nil
}

func (s *Store) Close() {
	s.pool.Close()
}

func (s *Store) SaveEvent(ctx context.Context, event *models.NormalizedEvent) error {
	// Upsert match record first
	if err := s.upsertMatch(ctx, event); err != nil {
		return fmt.Errorf("upsert match: %w", err)
	}

	payload, err := json.Marshal(event.Payload)
	if err != nil {
		return err
	}

	_, err = s.pool.Exec(ctx, `
		INSERT INTO match_events
			(event_id, external_event_id, match_id, event_type, minute, sequence, payload, source, received_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (event_id) DO NOTHING
	`,
		event.EventID,
		event.ExternalEventID,
		event.MatchID,
		string(event.EventType),
		event.Minute,
		event.Sequence,
		payload,
		event.Source,
		event.ReceivedAt,
	)
	return err
}

func (s *Store) upsertMatch(ctx context.Context, event *models.NormalizedEvent) error {
	switch event.EventType {
	case models.Scheduled:
		_, err := s.pool.Exec(ctx, `
			INSERT INTO matches
				(id, team_a_id, team_a_name, team_b_id, team_b_name, competition_title, competition_stage, status)
			VALUES ($1, $2, $3, $4, $5, $6, $7, 'SCHEDULED')
			ON CONFLICT (id) DO NOTHING
		`,
			event.MatchID,
			event.TeamACode, event.TeamA,
			event.TeamBCode, event.TeamB,
			event.CompetitionTitle,
			event.CompetitionStage,
		)
		return err

	case models.MatchStarted:
		_, err := s.pool.Exec(ctx, `
			INSERT INTO matches
				(id, team_a_id, team_a_name, team_b_id, team_b_name, competition_title, competition_stage, status, started_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, 'LIVE', $8)
			ON CONFLICT (id) DO UPDATE SET
				status     = 'LIVE',
				started_at = EXCLUDED.started_at,
				updated_at = NOW()
		`,
			event.MatchID,
			event.TeamACode, event.TeamA,
			event.TeamBCode, event.TeamB,
			event.CompetitionTitle,
			event.CompetitionStage,
			event.ReceivedAt,
		)
		return err

	case models.MatchEnded:
		// Compute final scores from events in DB
		_, err := s.pool.Exec(ctx, `
			UPDATE matches SET
				status     = 'ENDED',
				ended_at   = $2,
				updated_at = NOW(),
				score_a    = (
					SELECT COUNT(*) FROM match_events
					WHERE match_id = $1 AND event_type = 'GOAL'
					AND payload->>'team_id' = matches.team_a_id
				),
				score_b    = (
					SELECT COUNT(*) FROM match_events
					WHERE match_id = $1 AND event_type = 'GOAL'
					AND payload->>'team_id' = matches.team_b_id
				)
			WHERE id = $1
		`, event.MatchID, event.ReceivedAt)
		return err

	default:
		// Ensure match exists even if MATCH_STARTED was missed
		_, err := s.pool.Exec(ctx, `
			INSERT INTO matches
				(id, team_a_id, team_a_name, team_b_id, team_b_name, competition_title, competition_stage)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
			ON CONFLICT (id) DO NOTHING
		`,
			event.MatchID,
			event.TeamACode, event.TeamA,
			event.TeamBCode, event.TeamB,
			event.CompetitionTitle,
			event.CompetitionStage,
		)
		return err
	}
}
