-- +goose Up
-- One row per `ticketdoctor import`, so it's easy to see whether the daily job
-- worked and what it changed.
CREATE TABLE import_runs (
    id          bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    started_at  timestamptz NOT NULL DEFAULT now(),
    finished_at timestamptz,
    status      text  NOT NULL DEFAULT 'running' CHECK (status IN ('running', 'succeeded', 'failed')),
    summary     jsonb NOT NULL DEFAULT '{}',
    error       text
);

-- +goose Down
DROP TABLE import_runs;
