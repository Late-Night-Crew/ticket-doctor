-- Queries for sqlc (see sqlc.yaml). Each "-- name:" becomes a Go method on
-- dbsqlc.Queries: :one returns a row, :many a slice, :exec nothing.

-- ---------- leagues ----------

-- name: ListLeagues :many
SELECT * FROM leagues ORDER BY name;

-- name: GetLeague :one
SELECT * FROM leagues WHERE id = $1;

-- ---------- teams ----------

-- name: ListTeamsByLeague :many
SELECT * FROM teams WHERE league_id = $1 ORDER BY name;

-- name: InsertTeam :one
INSERT INTO teams (league_id, name, name_key)
VALUES ($1, $2, $3)
RETURNING *;

-- name: AddTeamAlias :exec
-- Records another spelling of a club (normalised), once.
UPDATE teams
SET aliases = array_append(aliases, sqlc.arg(alias)::text)
WHERE id = $1 AND NOT (sqlc.arg(alias)::text = ANY (aliases));

-- name: UpdateTeamForm :exec
UPDATE teams SET form = $2 WHERE id = $1;

-- ---------- matches ----------

-- name: UpsertMatch :one
-- Inserts a match, or updates it if anything changed. Returns whether a row
-- was inserted; returns no row at all when nothing changed.
INSERT INTO matches (league_id, season, kickoff, home_team_id, away_team_id, status,
                     home_goals, away_goals, odds_home, odds_draw, odds_away, source)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
ON CONFLICT (league_id, season, home_team_id, away_team_id) DO UPDATE
SET kickoff    = EXCLUDED.kickoff,
    status     = EXCLUDED.status,
    home_goals = EXCLUDED.home_goals,
    away_goals = EXCLUDED.away_goals,
    odds_home  = EXCLUDED.odds_home,
    odds_draw  = EXCLUDED.odds_draw,
    odds_away  = EXCLUDED.odds_away,
    source     = EXCLUDED.source,
    updated_at = now()
WHERE (matches.kickoff, matches.status, matches.home_goals, matches.away_goals,
       matches.odds_home, matches.odds_draw, matches.odds_away)
      IS DISTINCT FROM
      (EXCLUDED.kickoff, EXCLUDED.status, EXCLUDED.home_goals, EXCLUDED.away_goals,
       EXCLUDED.odds_home, EXCLUDED.odds_draw, EXCLUDED.odds_away)
RETURNING id, (xmax = 0)::boolean AS inserted;

-- name: UpsertFixture :one
-- Inserts an upcoming fixture, or updates its kick-off or status. A finished
-- match is never touched: its result comes from the results import. Returns
-- whether a row was inserted; returns no row at all when nothing changed.
INSERT INTO matches (league_id, season, kickoff, home_team_id, away_team_id, status, source)
VALUES ($1, $2, $3, $4, $5, $6, $7)
ON CONFLICT (league_id, season, home_team_id, away_team_id) DO UPDATE
SET kickoff    = EXCLUDED.kickoff,
    status     = EXCLUDED.status,
    source     = EXCLUDED.source,
    updated_at = now()
WHERE matches.status <> 'finished'
  AND (matches.kickoff, matches.status) IS DISTINCT FROM (EXCLUDED.kickoff, EXCLUDED.status)
RETURNING id, (xmax = 0)::boolean AS inserted;

-- name: ListFinishedMatchesByLeague :many
-- Every finished match in a league, newest first, for building profiles.
SELECT home_team_id, away_team_id, kickoff, home_goals, away_goals
FROM matches
WHERE league_id = $1 AND status = 'finished'
ORDER BY kickoff DESC;

-- name: CountMatchesByLeague :many
SELECT league_id, season, status, count(*) AS matches
FROM matches
GROUP BY league_id, season, status
ORDER BY league_id, season, status;

-- ---------- team profiles ----------

-- name: UpsertTeamProfile :exec
INSERT INTO team_profiles (team_id, venue, window_n, sample_size, win_rate, draw_rate, loss_rate,
                           gf_avg, ga_avg, scored_rate, clean_sheet_rate, btts_rate,
                           over15_rate, over25_rate, under35_rate)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
ON CONFLICT (team_id, venue, window_n) DO UPDATE
SET sample_size      = EXCLUDED.sample_size,
    win_rate         = EXCLUDED.win_rate,
    draw_rate        = EXCLUDED.draw_rate,
    loss_rate        = EXCLUDED.loss_rate,
    gf_avg           = EXCLUDED.gf_avg,
    ga_avg           = EXCLUDED.ga_avg,
    scored_rate      = EXCLUDED.scored_rate,
    clean_sheet_rate = EXCLUDED.clean_sheet_rate,
    btts_rate        = EXCLUDED.btts_rate,
    over15_rate      = EXCLUDED.over15_rate,
    over25_rate      = EXCLUDED.over25_rate,
    under35_rate     = EXCLUDED.under35_rate,
    updated_at       = now();

-- name: ListProfilesByLeague :many
SELECT t.name AS team, p.*
FROM team_profiles p
JOIN teams t ON t.id = p.team_id
WHERE t.league_id = $1 AND p.window_n = $2
ORDER BY t.name, p.venue;

-- ---------- import runs ----------

-- name: StartImportRun :one
INSERT INTO import_runs DEFAULT VALUES RETURNING id;

-- name: FinishImportRun :exec
UPDATE import_runs
SET finished_at = now(), status = $2, summary = $3, error = $4
WHERE id = $1;

-- ---------- fixtures (B3) ----------

-- name: ListFixtures :many
-- Games not yet played with kick-off in [from, to), optionally one league.
SELECT m.id, m.league_id, m.kickoff,
       h.id AS home_id, h.name AS home_name, h.form AS home_form,
       a.id AS away_id, a.name AS away_name, a.form AS away_form
FROM matches m
JOIN teams h ON h.id = m.home_team_id
JOIN teams a ON a.id = m.away_team_id
WHERE m.status = 'scheduled'
  AND m.kickoff >= sqlc.arg(kickoff_from)::timestamptz
  AND (sqlc.narg(kickoff_to)::timestamptz IS NULL OR m.kickoff < sqlc.narg(kickoff_to)::timestamptz)
  AND (sqlc.narg(league_id)::text IS NULL OR m.league_id = sqlc.narg(league_id)::text)
ORDER BY m.kickoff, m.league_id, h.name;

-- name: GetMatchesByIDs :many
-- Matches by id in any status, for analysing a ticket.
SELECT m.id, m.league_id, m.kickoff, m.status,
       h.id AS home_id, h.name AS home_name, h.form AS home_form,
       a.id AS away_id, a.name AS away_name, a.form AS away_form
FROM matches m
JOIN teams h ON h.id = m.home_team_id
JOIN teams a ON a.id = m.away_team_id
WHERE m.id = ANY (sqlc.arg(ids)::bigint[]);

-- name: ListProfilesForTeams :many
SELECT * FROM team_profiles
WHERE team_id = ANY (sqlc.arg(team_ids)::bigint[]) AND window_n = sqlc.arg(window_n);

-- ---------- ticket log (B3, SRS addendum A6) ----------

-- name: ListTicketLegs :many
SELECT * FROM analysed_legs WHERE ticket_id = $1;

-- name: InsertTicketLeg :one
INSERT INTO analysed_legs (ticket_id, origin, match_id, original_market, original_odds,
                           original_estimate, original_band, final_market)
VALUES ($1, $2, $3, $4, $5, $6, $7, $4)
RETURNING id;

-- name: UpdateTicketLeg :exec
-- Saves the latest analysis of a leg and puts it back on the ticket.
UPDATE analysed_legs
SET origin            = $2,
    original_market   = $3,
    original_odds     = $4,
    original_estimate = $5,
    original_band     = $6,
    final_market      = $7,
    on_ticket         = true,
    updated_at        = now()
WHERE id = $1;

-- name: TakeLegsOffTicket :exec
-- Marks a ticket's legs that are no longer on it (FR-7.1d).
UPDATE analysed_legs
SET on_ticket = false, updated_at = now()
WHERE ticket_id = $1 AND on_ticket AND NOT (id = ANY (sqlc.arg(keep)::bigint[]));

-- name: DeleteLegSuggestions :exec
-- The user changed their own pick: suggestions for the old pick no longer apply.
DELETE FROM suggestions WHERE analysed_leg_id = $1;

-- name: ListLegSuggestions :many
SELECT * FROM suggestions WHERE analysed_leg_id = ANY (sqlc.arg(leg_ids)::bigint[]);

-- name: UpsertSuggestion :one
INSERT INTO suggestions (analysed_leg_id, market, estimate, risk_band)
VALUES ($1, $2, $3, $4)
ON CONFLICT (analysed_leg_id, market) DO UPDATE
SET estimate = EXCLUDED.estimate, risk_band = EXCLUDED.risk_band
RETURNING id;

-- name: SyncAcceptedSuggestions :exec
-- A suggestion counts as accepted while it is the leg's final pick.
UPDATE suggestions s
SET accepted = (s.market = l.final_market)
FROM analysed_legs l
WHERE l.id = s.analysed_leg_id AND l.id = $1 AND s.accepted IS DISTINCT FROM (s.market = l.final_market);

-- name: AcceptSuggestion :execrows
UPDATE suggestions SET accepted = true WHERE id = $1;

-- ---------- results (B3) ----------

-- name: ListSettledLegs :many
-- Legs still on their ticket whose match has finished, with the score.
SELECT l.id, m.league_id, l.original_market, m.home_goals, m.away_goals,
       EXISTS (SELECT 1 FROM suggestions s WHERE s.analysed_leg_id = l.id) AS had_suggestion
FROM analysed_legs l
JOIN matches m ON m.id = l.match_id
WHERE l.on_ticket AND m.status = 'finished';

-- name: ListSettledSuggestions :many
SELECT s.market, m.league_id, m.home_goals, m.away_goals
FROM suggestions s
JOIN analysed_legs l ON l.id = s.analysed_leg_id
JOIN matches m ON m.id = l.match_id
WHERE l.on_ticket AND m.status = 'finished';

-- name: LastSuccessfulImport :one
SELECT finished_at FROM import_runs
WHERE status = 'succeeded' AND finished_at IS NOT NULL
ORDER BY finished_at DESC
LIMIT 1;
