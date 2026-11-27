-- +goose Up
-- Results and fixtures. A league season has each home/away pairing exactly
-- once, so (league, season, home, away) identifies a match: re-imports and
-- rescheduled kick-offs update the same row instead of adding duplicates.
CREATE TABLE matches (
    id           bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    league_id    text     NOT NULL REFERENCES leagues (id),
    season       text     NOT NULL CHECK (season ~ '^[0-9]{4}$'),   -- '2627' = 2026/27
    kickoff      timestamptz NOT NULL,
    home_team_id bigint   NOT NULL REFERENCES teams (id),
    away_team_id bigint   NOT NULL REFERENCES teams (id),
    status       text     NOT NULL CHECK (status IN ('scheduled', 'finished')),
    home_goals   smallint CHECK (home_goals >= 0),
    away_goals   smallint CHECK (away_goals >= 0),
    -- Closing market-average decimal odds where the source has them.
    odds_home    numeric(7, 2),
    odds_draw    numeric(7, 2),
    odds_away    numeric(7, 2),
    source       text     NOT NULL,
    updated_at   timestamptz NOT NULL DEFAULT now(),
    CHECK (home_team_id <> away_team_id),
    -- Finished matches have a score; scheduled ones don't.
    CHECK ((status = 'finished') = (home_goals IS NOT NULL AND away_goals IS NOT NULL)),
    UNIQUE (league_id, season, home_team_id, away_team_id)
);

-- Profiles read each team's latest finished matches; fixtures are listed by date.
CREATE INDEX matches_home_team_idx ON matches (home_team_id, kickoff DESC) WHERE status = 'finished';
CREATE INDEX matches_away_team_idx ON matches (away_team_id, kickoff DESC) WHERE status = 'finished';
CREATE INDEX matches_league_kickoff_idx ON matches (league_id, status, kickoff);

-- +goose Down
DROP TABLE matches;
