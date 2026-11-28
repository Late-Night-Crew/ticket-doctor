-- +goose Up
-- Quick score checks (addendum A10) run every 15 minutes on match days and
-- are recorded here too, so the dashboard can show when results were last
-- checked. kind keeps them apart from the full imports.
ALTER TABLE import_runs ADD COLUMN kind text NOT NULL DEFAULT 'import' CHECK (kind IN ('import', 'scores'));
CREATE INDEX import_runs_kind_started_idx ON import_runs (kind, started_at DESC);

-- +goose Down
DROP INDEX import_runs_kind_started_idx;
ALTER TABLE import_runs DROP COLUMN kind;
