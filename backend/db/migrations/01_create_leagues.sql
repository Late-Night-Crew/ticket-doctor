-- +goose Up
-- Supported leagues (SRS 2.5). The MVP's three leagues are reference data, so
-- they're inserted here rather than by an import.
CREATE TABLE leagues (
    id         text PRIMARY KEY,                                   -- e.g. 'epl', used in URLs
    name       text NOT NULL,
    country    text NOT NULL,
    source     text NOT NULL CHECK (source IN ('csv', 'api', 'manual')),
    fd_code    text UNIQUE,                                        -- football-data.co.uk division, e.g. 'E0'
    created_at timestamptz NOT NULL DEFAULT now()
);

INSERT INTO leagues (id, name, country, source, fd_code) VALUES
    ('epl',        'Premier League', 'England', 'csv', 'E0'),
    ('laliga',     'La Liga',        'Spain',   'csv', 'SP1'),
    ('bundesliga', 'Bundesliga',     'Germany', 'csv', 'D1');

-- +goose Down
DROP TABLE leagues;
