-- +goose Up
-- Precomputed home and away profiles (FR-2), rebuilt by every import, so a
-- ticket analysis only reads cached rows (SRS 2.3, NFR-1).
CREATE TABLE team_profiles (
    team_id          bigint   NOT NULL REFERENCES teams (id),
    venue            text     NOT NULL CHECK (venue IN ('home', 'away')),
    window_n         smallint NOT NULL CHECK (window_n > 0),        -- form window used (NFR-5)
    sample_size      smallint NOT NULL CHECK (sample_size >= 0),    -- matches actually found (FR-2.3)
    win_rate         double precision NOT NULL,
    draw_rate        double precision NOT NULL,
    loss_rate        double precision NOT NULL,
    gf_avg           double precision NOT NULL,
    ga_avg           double precision NOT NULL,
    scored_rate      double precision NOT NULL,
    clean_sheet_rate double precision NOT NULL,
    btts_rate        double precision NOT NULL,
    over15_rate      double precision NOT NULL,
    over25_rate      double precision NOT NULL,
    under35_rate     double precision NOT NULL,
    updated_at       timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (team_id, venue, window_n)
);

-- +goose Down
DROP TABLE team_profiles;
