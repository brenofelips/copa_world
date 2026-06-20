-- Enable UUID generation
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ─── Teams ────────────────────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS teams (
    id         VARCHAR(20)  PRIMARY KEY,
    name       VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

-- ─── Players ──────────────────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS players (
    id         VARCHAR(50)  PRIMARY KEY,
    team_id    VARCHAR(20)  REFERENCES teams(id) ON DELETE SET NULL,
    name       VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

-- ─── Matches ──────────────────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS matches (
    id                  VARCHAR(100) PRIMARY KEY,
    team_a_id           VARCHAR(20)  NOT NULL,
    team_a_name         VARCHAR(255) NOT NULL,
    team_b_id           VARCHAR(20)  NOT NULL,
    team_b_name         VARCHAR(255) NOT NULL,
    competition_title   VARCHAR(255),
    competition_stage   VARCHAR(100),
    status              VARCHAR(20)  NOT NULL DEFAULT 'SCHEDULED',
    score_a             INT          NOT NULL DEFAULT 0,
    score_b             INT          NOT NULL DEFAULT 0,
    started_at          TIMESTAMPTZ,
    ended_at            TIMESTAMPTZ,
    created_at          TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_matches_status     ON matches(status);
CREATE INDEX IF NOT EXISTS idx_matches_team_a_id  ON matches(team_a_id);
CREATE INDEX IF NOT EXISTS idx_matches_team_b_id  ON matches(team_b_id);

-- ─── Match Events ─────────────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS match_events (
    id                UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id          VARCHAR(255) UNIQUE NOT NULL,
    external_event_id VARCHAR(255),
    match_id          VARCHAR(100) NOT NULL REFERENCES matches(id) ON DELETE CASCADE,
    event_type        VARCHAR(50)  NOT NULL,
    minute            INT          NOT NULL DEFAULT 0,
    sequence          INT          NOT NULL DEFAULT 0,
    payload           JSONB,
    source            VARCHAR(255),
    received_at       TIMESTAMPTZ,
    created_at        TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_match_events_match_id   ON match_events(match_id);
CREATE INDEX IF NOT EXISTS idx_match_events_event_type ON match_events(event_type);
CREATE INDEX IF NOT EXISTS idx_match_events_received   ON match_events(received_at DESC);
CREATE INDEX IF NOT EXISTS idx_match_events_player     ON match_events((payload->>'player_id'));
CREATE INDEX IF NOT EXISTS idx_match_events_team       ON match_events((payload->>'team_id'));
