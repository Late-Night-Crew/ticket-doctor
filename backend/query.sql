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

-- name: CountSettledLegs :many
-- Legs still on their ticket whose match has finished, counted per league,
-- user's pick, final score and whether we suggested a swap. Scores repeat a
-- lot, so this stays a few hundred rows however many tickets there are; the
-- engine settles each group once (B4).
SELECT m.league_id, l.original_market, m.home_goals, m.away_goals,
       EXISTS (SELECT 1 FROM suggestions s WHERE s.analysed_leg_id = l.id) AS had_suggestion,
       count(*) AS legs
FROM analysed_legs l
JOIN matches m ON m.id = l.match_id
WHERE l.on_ticket AND m.status = 'finished'
  AND (sqlc.narg(league_id)::text IS NULL OR m.league_id = sqlc.narg(league_id)::text)
  AND (sqlc.narg(market)::text IS NULL OR l.original_market = sqlc.narg(market)::text)
GROUP BY 1, 2, 3, 4, 5;

-- name: CountSettledSuggestions :many
-- Suggestions on finished matches, counted per league, market and final score.
SELECT m.league_id, s.market, m.home_goals, m.away_goals, count(*) AS suggestions
FROM suggestions s
JOIN analysed_legs l ON l.id = s.analysed_leg_id
JOIN matches m ON m.id = l.match_id
WHERE l.on_ticket AND m.status = 'finished'
  AND (sqlc.narg(league_id)::text IS NULL OR m.league_id = sqlc.narg(league_id)::text)
  AND (sqlc.narg(market)::text IS NULL OR s.market = sqlc.narg(market)::text)
GROUP BY 1, 2, 3, 4;

-- name: LastSuccessfulImport :one
SELECT finished_at FROM import_runs
WHERE status = 'succeeded' AND finished_at IS NOT NULL
ORDER BY finished_at DESC
LIMIT 1;

-- ---------- admin (B6) ----------

-- name: CreateUser :one
INSERT INTO users (username, password_hash) VALUES ($1, $2) RETURNING *;

-- name: GetUserByUsername :one
SELECT * FROM users WHERE username = $1;

-- name: GetUser :one
SELECT * FROM users WHERE id = $1;

-- name: ListUsers :many
SELECT * FROM users ORDER BY username;

-- name: SetUserPassword :execrows
UPDATE users SET password_hash = $2, password_changed_at = now() WHERE username = $1;

-- name: SetUserDisabled :execrows
UPDATE users SET disabled = $2 WHERE username = $1;

-- name: RecordLogin :exec
UPDATE users SET last_login_at = now() WHERE id = $1;

-- name: ListAdminMatches :many
-- Matches for the result entry page. Awaiting: kicked off but no result yet.
-- Recent: finished in the last two weeks, newest first, to correct a score.
SELECT m.id, m.league_id, m.kickoff, m.status, m.home_goals, m.away_goals, m.source,
       h.name AS home_name, a.name AS away_name
FROM matches m
JOIN teams h ON h.id = m.home_team_id
JOIN teams a ON a.id = m.away_team_id
WHERE (sqlc.narg(league_id)::text IS NULL OR m.league_id = sqlc.narg(league_id)::text)
  AND CASE WHEN sqlc.arg(awaiting)::boolean
           THEN m.status IN ('scheduled', 'postponed') AND m.kickoff < sqlc.arg(now)::timestamptz
           ELSE m.status = 'finished' AND m.kickoff >= sqlc.arg(now)::timestamptz - interval '14 days'
      END
ORDER BY CASE WHEN sqlc.arg(awaiting)::boolean THEN m.kickoff END ASC,
         m.kickoff DESC, m.id
LIMIT 300;

-- name: GetMatchForUpdate :one
SELECT id, league_id, kickoff, status, home_goals, away_goals FROM matches WHERE id = $1 FOR UPDATE;

-- name: SetManualResult :exec
UPDATE matches
SET status = 'finished', home_goals = $2, away_goals = $3, source = 'manual', updated_at = now()
WHERE id = $1;

-- name: InsertResultEntry :exec
INSERT INTO result_entries (match_id, user_id, home_goals, away_goals, previous_home, previous_away)
VALUES ($1, $2, $3, $4, $5, $6);

-- name: DashboardCounts :one
SELECT
  (SELECT count(*) FROM matches WHERE status = 'scheduled' AND kickoff >= sqlc.arg(now)::timestamptz) AS upcoming,
  (SELECT count(*) FROM matches WHERE status IN ('scheduled', 'postponed') AND kickoff < sqlc.arg(now)::timestamptz) AS awaiting_result,
  (SELECT count(*) FROM matches WHERE status = 'finished') AS finished,
  (SELECT count(DISTINCT ticket_id) FROM analysed_legs) AS tickets,
  (SELECT count(DISTINCT ticket_id) FROM analysed_legs WHERE created_at >= sqlc.arg(now)::timestamptz - interval '7 days') AS tickets_7d,
  (SELECT count(*) FROM analysed_legs WHERE on_ticket) AS legs,
  (SELECT count(*) FROM suggestions s JOIN analysed_legs l ON l.id = s.analysed_leg_id WHERE l.on_ticket) AS suggestions,
  (SELECT count(*) FROM suggestions s JOIN analysed_legs l ON l.id = s.analysed_leg_id WHERE l.on_ticket AND s.accepted) AS accepted;

-- name: ListImportRuns :many
SELECT * FROM import_runs ORDER BY started_at DESC LIMIT $1;

-- name: CountUsers :one
SELECT count(*) FROM users;

-- name: LockUsers :exec
-- Serialises first-admin setup, so two people can't both create "the first" account.
LOCK TABLE users IN SHARE ROW EXCLUSIVE MODE;

-- ---------- saved tickets (V2, FR-9) ----------

-- name: GetSavedByTicket :one
SELECT * FROM saved_tickets WHERE ticket_id = $1;

-- name: InsertSaved :exec
INSERT INTO saved_tickets (id, ticket_id, origin) VALUES ($1, $2, $3);

-- name: ResaveSaved :exec
-- Saving again replaces the snapshot; it may no longer be settled.
UPDATE saved_tickets SET origin = $2, updated_at = now(), settled_at = NULL WHERE id = $1;

-- name: DeleteSavedLegs :exec
DELETE FROM saved_ticket_legs WHERE saved_ticket_id = $1;

-- name: SnapshotLegs :execrows
-- Copies the ticket's current legs and the suggestions shown for them.
INSERT INTO saved_ticket_legs (saved_ticket_id, match_id, original_market, original_odds, original_estimate,
                               original_band, final_market, suggestions, position)
SELECT sqlc.arg(saved_id)::text, l.match_id, l.original_market, l.original_odds, l.original_estimate,
       l.original_band, l.final_market,
       COALESCE((SELECT jsonb_agg(jsonb_build_object('market', s.market, 'estimate', s.estimate,
                                                     'risk_band', s.risk_band, 'accepted', s.accepted)
                                  ORDER BY s.estimate DESC)
                 FROM suggestions s WHERE s.analysed_leg_id = l.id), '[]'::jsonb),
       row_number() OVER (ORDER BY l.id)::int
FROM analysed_legs l
WHERE l.ticket_id = sqlc.arg(ticket_id)::text AND l.on_ticket;

-- name: ListSaved :many
SELECT * FROM saved_tickets WHERE id = ANY (sqlc.arg(ids)::text[]);

-- name: ListSavedLegs :many
-- Legs of the given saved tickets with their match, teams and score.
SELECT l.saved_ticket_id, l.position, l.original_market, l.original_odds, l.original_estimate, l.original_band,
       l.final_market, l.suggestions,
       m.id AS match_id, m.league_id, m.kickoff, m.status, m.home_goals, m.away_goals,
       h.id AS home_id, h.name AS home_name, h.form AS home_form,
       a.id AS away_id, a.name AS away_name, a.form AS away_form
FROM saved_ticket_legs l
JOIN matches m ON m.id = l.match_id
JOIN teams h ON h.id = m.home_team_id
JOIN teams a ON a.id = m.away_team_id
WHERE l.saved_ticket_id = ANY (sqlc.arg(ids)::text[])
ORDER BY l.saved_ticket_id, l.position;

-- name: DeleteSaved :execrows
-- Only with the ticket's own id, which only the saving browser has.
DELETE FROM saved_tickets WHERE id = $1 AND ticket_id = $2;

-- name: MarkSavedSettled :execrows
-- settled_at comes from the app's clock, the same one that later expires it.
UPDATE saved_tickets t SET settled_at = sqlc.arg(now)::timestamptz
WHERE settled_at IS NULL
  AND NOT EXISTS (SELECT 1 FROM saved_ticket_legs l JOIN matches m ON m.id = l.match_id
                  WHERE l.saved_ticket_id = t.id AND m.status <> 'finished');

-- name: DeleteExpiredSaved :execrows
-- Settled tickets go after the keep period; tickets that never settle (a
-- game cancelled for good) go after a longer one.
DELETE FROM saved_tickets
WHERE settled_at < sqlc.arg(settled_before)::timestamptz OR created_at < sqlc.arg(created_before)::timestamptz;

-- ---------- calibration (are the estimates right?) ----------

-- name: ListFinishedMatches :many
-- Every finished match, oldest first, for the backtest.
SELECT league_id, kickoff, home_team_id, away_team_id, home_goals, away_goals
FROM matches
WHERE status = 'finished'
  AND (sqlc.narg(league_id)::text IS NULL OR league_id = sqlc.narg(league_id)::text)
ORDER BY kickoff, id;

-- name: ListSettledPickEstimates :many
-- Users' own picks on finished matches, with the estimate shown at the time.
SELECT m.league_id, l.original_market AS market, l.original_estimate AS estimate, l.original_band AS band,
       m.home_goals, m.away_goals
FROM analysed_legs l
JOIN matches m ON m.id = l.match_id
WHERE l.on_ticket AND m.status = 'finished'
  AND (sqlc.narg(league_id)::text IS NULL OR m.league_id = sqlc.narg(league_id)::text);

-- name: ListSettledSuggestionEstimates :many
-- Suggestions shown on finished matches, with their estimates.
SELECT m.league_id, s.market, s.estimate, s.risk_band AS band, m.home_goals, m.away_goals
FROM suggestions s
JOIN analysed_legs l ON l.id = s.analysed_leg_id
JOIN matches m ON m.id = l.match_id
WHERE l.on_ticket AND m.status = 'finished'
  AND (sqlc.narg(league_id)::text IS NULL OR m.league_id = sqlc.narg(league_id)::text);

-- ---------- league profiles (prior_games) ----------

-- name: UpsertLeagueProfile :exec
INSERT INTO league_profiles (league_id, venue, sample_size, win_rate, draw_rate, loss_rate, gf_avg, ga_avg,
                             scored_rate, clean_sheet_rate, btts_rate, over15_rate, over25_rate, under35_rate)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
ON CONFLICT (league_id, venue) DO UPDATE
SET sample_size = EXCLUDED.sample_size, win_rate = EXCLUDED.win_rate, draw_rate = EXCLUDED.draw_rate,
    loss_rate = EXCLUDED.loss_rate, gf_avg = EXCLUDED.gf_avg, ga_avg = EXCLUDED.ga_avg,
    scored_rate = EXCLUDED.scored_rate, clean_sheet_rate = EXCLUDED.clean_sheet_rate,
    btts_rate = EXCLUDED.btts_rate, over15_rate = EXCLUDED.over15_rate, over25_rate = EXCLUDED.over25_rate,
    under35_rate = EXCLUDED.under35_rate, updated_at = now();

-- name: ListLeagueProfiles :many
SELECT * FROM league_profiles;

-- name: ListTeamLeagues :many
SELECT id, league_id FROM teams WHERE id = ANY (sqlc.arg(ids)::bigint[]);
