-- +goose Up
-- Upcoming fixtures come from football-data.org (decision D4), which has its
-- own competition codes. Results still come from football-data.co.uk (fd_code).
ALTER TABLE leagues ADD COLUMN fdo_code text UNIQUE;

UPDATE leagues SET fdo_code = 'PL'  WHERE id = 'epl';
UPDATE leagues SET fdo_code = 'PD'  WHERE id = 'laliga';
UPDATE leagues SET fdo_code = 'BL1' WHERE id = 'bundesliga';

-- A fixture can be postponed (or suspended or cancelled). It keeps its row,
-- so a new date later updates it, but fixture lists leave it out.
-- Backward compatible: v0.2.0 only reads finished matches.
ALTER TABLE matches DROP CONSTRAINT matches_status_check;
ALTER TABLE matches ADD CONSTRAINT matches_status_check
    CHECK (status IN ('scheduled', 'postponed', 'finished'));

-- +goose Down
UPDATE matches SET status = 'scheduled' WHERE status = 'postponed';
ALTER TABLE matches DROP CONSTRAINT matches_status_check;
ALTER TABLE matches ADD CONSTRAINT matches_status_check
    CHECK (status IN ('scheduled', 'finished'));
ALTER TABLE leagues DROP COLUMN fdo_code;
