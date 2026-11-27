-- +goose Up
-- The ticket log behind the track record (FR-7, SRS addendum A6).
--
-- One row per ticket per match, however often the ticket is analysed. The
-- user's pick (original_market) is the market they chose themselves; accepting
-- a suggestion changes only final_market. Outcomes are worked out from the
-- match score when results are read, so nothing here needs updating when a
-- match finishes.
CREATE TABLE analysed_legs (
    id                bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    -- Random id made in the browser (a UUID, or a fallback on old browsers).
    ticket_id         text   NOT NULL CHECK (ticket_id ~ '^[A-Za-z0-9-]{8,64}$'),
    origin            text   NOT NULL CHECK (origin IN ('manual', 'wizard')),
    match_id          bigint NOT NULL REFERENCES matches (id),
    original_market   text   NOT NULL,
    original_odds     numeric(7, 2),
    original_estimate double precision NOT NULL CHECK (original_estimate BETWEEN 0 AND 1),
    original_band     text   NOT NULL CHECK (original_band IN ('low', 'medium', 'high')),
    final_market      text   NOT NULL,
    -- False once the match is taken off the ticket; such legs aren't counted.
    on_ticket         boolean NOT NULL DEFAULT true,
    created_at        timestamptz NOT NULL DEFAULT now(),
    updated_at        timestamptz NOT NULL DEFAULT now(),
    UNIQUE (ticket_id, match_id),
    CHECK (original_market IN ('HOME_WIN', 'AWAY_WIN', 'DRAW', 'DC_1X', 'DC_X2', 'BTTS_YES', 'BTTS_NO',
                               'OVER_1_5', 'UNDER_1_5', 'OVER_2_5', 'UNDER_2_5', 'OVER_3_5', 'UNDER_3_5',
                               'HOME_TO_SCORE', 'AWAY_TO_SCORE')),
    CHECK (final_market IN ('HOME_WIN', 'AWAY_WIN', 'DRAW', 'DC_1X', 'DC_X2', 'BTTS_YES', 'BTTS_NO',
                            'OVER_1_5', 'UNDER_1_5', 'OVER_2_5', 'UNDER_2_5', 'OVER_3_5', 'UNDER_3_5',
                            'HOME_TO_SCORE', 'AWAY_TO_SCORE'))
);

-- The results page reads legs by match.
CREATE INDEX analysed_legs_match_idx ON analysed_legs (match_id) WHERE on_ticket;

-- Every suggestion shown for a leg, once per market (FR-5, FR-7.1c).
CREATE TABLE suggestions (
    id               bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    analysed_leg_id  bigint NOT NULL REFERENCES analysed_legs (id) ON DELETE CASCADE,
    market           text   NOT NULL CHECK (market IN ('HOME_WIN', 'AWAY_WIN', 'DRAW', 'DC_1X', 'DC_X2',
                                                       'BTTS_YES', 'BTTS_NO', 'OVER_1_5', 'UNDER_1_5',
                                                       'OVER_2_5', 'UNDER_2_5', 'OVER_3_5', 'UNDER_3_5',
                                                       'HOME_TO_SCORE', 'AWAY_TO_SCORE')),
    estimate         double precision NOT NULL CHECK (estimate BETWEEN 0 AND 1),
    risk_band        text   NOT NULL CHECK (risk_band IN ('low', 'medium', 'high')),
    accepted         boolean NOT NULL DEFAULT false,
    created_at       timestamptz NOT NULL DEFAULT now(),
    UNIQUE (analysed_leg_id, market)
);

-- +goose Down
DROP TABLE suggestions;
DROP TABLE analysed_legs;
