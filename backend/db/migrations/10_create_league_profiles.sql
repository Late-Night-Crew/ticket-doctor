-- +goose Up
-- Each league's average record at home and away, over about its last season,
-- rebuilt by every import. Estimates pull each team's own (ten-game) rates
-- towards these (rule prior_games, agreed 2026-11-28 from the backtest).
CREATE TABLE league_profiles (
    league_id        text     NOT NULL REFERENCES leagues (id),
    venue            text     NOT NULL CHECK (venue IN ('home', 'away')),
    sample_size      integer  NOT NULL CHECK (sample_size >= 0),
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
    PRIMARY KEY (league_id, venue)
);

-- +goose Down
DROP TABLE league_profiles;
