-- +goose Up
-- People who can sign in to the admin pages (B6, open question 6). There is no
-- sign-up: accounts are made with `ticketdoctor admin create`. Public users
-- never get accounts (V2 saved tickets are link-based, SRS addendum B3), so
-- only a username is stored: no email or other personal data. role leaves
-- room for other kinds of account later.
CREATE TABLE users (
    id                  bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    username            text NOT NULL UNIQUE CHECK (username ~ '^[a-z0-9._-]{3,32}$'),
    password_hash       text NOT NULL,
    role                text NOT NULL DEFAULT 'admin' CHECK (role IN ('admin')),
    -- Sessions signed before this moment are refused, so changing a password
    -- signs that account out everywhere.
    password_changed_at timestamptz NOT NULL DEFAULT now(),
    disabled            boolean NOT NULL DEFAULT false,
    last_login_at       timestamptz,
    created_at          timestamptz NOT NULL DEFAULT now()
);

-- Every result typed in by an admin (FR-1.4), with what it replaced, so a
-- mistake can be traced and put right.
CREATE TABLE result_entries (
    id              bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    match_id        bigint   NOT NULL REFERENCES matches (id),
    user_id         bigint   NOT NULL REFERENCES users (id),
    home_goals      smallint NOT NULL CHECK (home_goals >= 0),
    away_goals      smallint NOT NULL CHECK (away_goals >= 0),
    -- The score before this entry; null when the match had none.
    previous_home   smallint,
    previous_away   smallint,
    created_at      timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX result_entries_match_idx ON result_entries (match_id, created_at DESC);

-- +goose Down
DROP TABLE result_entries;
DROP TABLE users;
