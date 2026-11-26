-- +goose Up
-- Clubs per league. name is the spelling football-data.co.uk uses (the primary
-- source); name_key is that name normalised by pkg/teamnames, and aliases holds
-- the normalised keys of other spellings, so the same club matches across
-- sources (FR-1.5).
CREATE TABLE teams (
    id         bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    league_id  text   NOT NULL REFERENCES leagues (id),
    name       text   NOT NULL,
    name_key   text   NOT NULL,
    aliases    text[] NOT NULL DEFAULT '{}',
    form       text   NOT NULL DEFAULT '',                         -- last 5 results, oldest first (FR-2.4)
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (league_id, name_key)
);

CREATE INDEX teams_aliases_idx ON teams USING gin (aliases);

-- +goose Down
DROP TABLE teams;
