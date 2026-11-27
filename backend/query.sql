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
