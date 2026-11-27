-- +goose Up
-- V2 saved tickets (SRS addendum B, FR-9). No accounts: a saved ticket is
-- reached by its random id (the private link), and only the browser that
-- holds the ticket's own ticket_id (never in the link) can update or delete
-- it. Saving takes a snapshot of the ticket log at that moment, so later
-- edits don't change what was saved; saving again replaces the snapshot and
-- keeps the link. Outcomes are worked out from the match scores when read.
CREATE TABLE saved_tickets (
    -- 128 random bits, base64url: the private link /t/<id>.
    id          text PRIMARY KEY CHECK (id ~ '^[A-Za-z0-9_-]{22}$'),
    ticket_id   text NOT NULL UNIQUE CHECK (ticket_id ~ '^[A-Za-z0-9-]{8,64}$'),
    origin      text NOT NULL CHECK (origin IN ('manual', 'wizard')),
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now(),
    -- Set by the daily clean-up once every game has a result; the ticket is
    -- deleted 90 days later (decided 2026-11-27).
    settled_at  timestamptz
);

CREATE TABLE saved_ticket_legs (
    id                bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    saved_ticket_id   text   NOT NULL REFERENCES saved_tickets (id) ON DELETE CASCADE,
    match_id          bigint NOT NULL REFERENCES matches (id),
    original_market   text   NOT NULL,
    original_odds     numeric(7, 2),
    original_estimate double precision NOT NULL,
    original_band     text   NOT NULL,
    final_market      text   NOT NULL,
    -- Every suggestion shown for this leg: [{market, estimate, risk_band, accepted}].
    suggestions       jsonb  NOT NULL DEFAULT '[]',
    position          int    NOT NULL,
    UNIQUE (saved_ticket_id, match_id)
);

CREATE INDEX saved_ticket_legs_ticket_idx ON saved_ticket_legs (saved_ticket_id, position);
CREATE INDEX saved_tickets_settled_idx ON saved_tickets (settled_at) WHERE settled_at IS NOT NULL;

-- +goose Down
DROP TABLE saved_ticket_legs;
DROP TABLE saved_tickets;
