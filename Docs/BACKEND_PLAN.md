# Ticket Doctor: backend implementation plan

Sources:
- *Ticket Doctor — Software Requirements Specification (MVP)* and [SRS_ADDENDUM.md](SRS_ADDENDUM.md).
- The API contract the frontend already uses: `frontend/src/api/types.ts`.
- The reference rules implementation: `frontend/src/mocks/scorer.ts` and its tests.
- The layout follows the Late Night Crew School Management System backend (`nsms`), which itself follows parrot. See [backend/README.md](../backend/README.md).

## 1. How it fits together

```
football-data.co.uk CSVs ──┐  results + odds
football-data.org API ─────┤  upcoming fixtures (D4)
                           ▼
           ticketdoctor import   (GitHub Actions, twice a day, latest release image)
           1. upsert matches (results + upcoming fixtures)
           2. recompute team_profiles (FR-2)
              (legs are settled from the score when results are read, B4)
                           │
                           ▼
              PostgreSQL (Render) ◄──────── ticketdoctor server (Render, image)
                                               reads cached rows only,
                                               runs engine/ per request
                                                       ▲ /api/* rewrite
                              React app (Render static site, same address)
```

- **Ticket requests never call an outside API** (SRS 2.3, NFR-1). A 15-leg analysis is two queries (matches, profiles) plus pure engine code.
- **The import runs from GitHub Actions on a schedule, not inside the server.** Render's free tier sleeps when idle, so an in-process scheduler would miss runs, and Render's cron jobs aren't free.

## 2. Stack

| Concern | Choice | Same as nsms? |
|---|---|---|
| HTTP | Fiber **v3** (decided, D1) | Same framework; nsms is on v2 |
| CLI | cobra | Yes |
| Config | viper YAML + `TD_*` environment variables for secrets | Yes |
| Database | PostgreSQL via **pgx v5 only** | Simplified: nsms mixes pgx v4, pgx v5 and lib/pq |
| Queries | sqlc (`query.sql` → `database/dbsqlc`) | Yes |
| Migrations | goose, embedded with `go:embed` | Yes |
| Validation | go-playground/validator v10 | Yes |
| Errors | RFC 7807 `pkg/problemdetail` | Yes (one copy) |
| CSV | `encoding/csv` (SRS 7) | New |
| Jobs | **None in MVP**: CLI + GitHub Actions cron | nsms uses River; not needed here |
| Auth | **None for public endpoints** (no accounts, NFR-8). Admin only, later | nsms JWT pattern reused for admin (D5) |
| Tests | `go test`, table-driven; engine tests match `scorer.test.ts` | Yes |
| Tasks | `Taskfile.yml` (`task dev`, `task test`, `task sqlc`, `task migrate`) | Like parrot |

## 3. API (must match `frontend/src/api/types.ts`)

| Method and path | Handler | Notes |
|---|---|---|
| `GET /api/health` | `api` | For Render health checks |
| `GET /api/leagues` | `api/leagues` | |
| `GET /api/leagues/:id/fixtures?when=today\|weekend\|upcoming` | `api/leagues` | "Weekend" timezone: open question 4 |
| `POST /api/tickets/analyse` | `api/tickets` | 1–15 legs, one per match; logs `analysed_legs` and `suggestions` |
| `POST /api/suggestions/:id/accept` | `api/tickets` | Sets `accepted = true` |
| `POST /api/wizard` | `api/wizard` | `legs` clamped to 2–10, or 2–4 for beginners (addendum A3) |
| `GET /api/results?league=&market=` | `api/results` | Counts distinct (ticket, match, market) so re-analysis isn't double-counted; see open question 10 |
| `/api/admin/...` | `api/admin` | Later (F7), after D5 |

Conventions:
- JSON field names are snake_case, as in nsms.
- IDs are sent as **strings** in JSON, which the frontend already expects.
- Errors use problem details, with `detail` holding the user-facing message. The frontend shows it as is.
- Public `POST` endpoints are rate-limited per IP, like nsms's `rateLimit`.
- CORS allows only the frontend origins listed in config.

## 4. Database (goose migrations)

Based on SRS 5; additions are marked **(+)**.

| # | Migration | Tables / changes |
|---|---|---|
| 01 | create_leagues | `leagues(id text pk 'epl', name, country, source, fd_code 'E0')` |
| 02 | create_teams | `teams(id, league_id, name, aliases text[])` + **(+)** unique (league_id, name) |
| 03 | create_matches | `matches(id, league_id, kickoff timestamptz, home_team_id, away_team_id, home_goals, away_goals, status, odds_home/draw/away)` + **(+)** unique (league_id, kickoff::date, home, away) for idempotent imports |
| 04 | create_team_profiles | SRS fields, pk (team_id, venue, window_n) |
| 05 | create_import_runs **(+)** | Moved up: done in B2 |
| 06 | add_fixtures_source **(+)** | `leagues.fdo_code` (PL, PD, BL1) and match status `postponed` (B2) |
| 07 | create_ticket_log (B3) | `analysed_legs` (one row per ticket and match: user's pick, final pick, `on_ticket`) and `suggestions` (`accepted`). No stored outcome: it's worked out from the score when results are read (addendum A6) |

Market values are stored as the same keys as the frontend (`HOME_WIN`, `OVER_2_5`…), with a CHECK constraint.

## 5. Milestones

Each milestone ends with green CI and a pull request.

**B0 Skeleton: done 2026-11-26**
- `go.mod` (module `ticketdoctor`), `main.go`, `cmd/root|server|migrate`, config, logging, problemdetail, `GET /api/health`.
- `Taskfile.yml`, `config.example.yaml`.
- **`.github/workflows/backend.yml`**: `setup-go`, `go vet`, `go test`, `go build`. This is the CI exercise for you.
- Done when `ticketdoctor server` answers `/api/health` and CI is green.
- Also added: `docker-compose.yml` (local Postgres on port 55432), graceful shutdown on SIGTERM, request logging with request IDs, and CORS from config. Config works on environment variables alone, with no file needed on Render. The frontend now reads problem-detail `detail` messages, and the mock API returns the same format.

**B1 Rules engine (SRS M2): done 2026-11-26**
- Port `scorer.ts` to `engine/`: profile maths, estimates (4.1 + A1), bands (4.2), swaps (4.3 + A2), combined and value (4.5), wizard (4.6 + A4), and **settle(market, goals) → won/lost**.
- Thresholds come from config (NFR-6).
- Done when the Go tests reproduce every case in `scorer.test.ts`, including the 4.4 worked example (NFR-7). Done first because it is pure, test-led, and the frontend tests already give the expected numbers.
- **Result:** `backend/engine`, with 100% test coverage.
  - **Parity:** `engine/testdata/parity.json` is generated from the TypeScript scorer: 36 team pairings, 540 estimates with their reason sentences, 604 swaps and 36 wizard tickets. The Go tests must match it exactly, and the frontend test `src/mocks/parity.test.ts` checks the scorer still produces it, so changing a rule in one language only fails CI. Mutation-checked: a one-word reason change or a 0.11 swap gain fails it.
  - **Settle:** settles all 15 markets (needed for B4). A test checks that `Contradicts` is exactly "no score from 0-0 to 9-9 lets both win", which proves addendum A2 has no gaps.
  - **Also included:** profile building from results with the window and form string (FR-2), and a `rules:` section in config (NFR-6) that is validated at startup.

**B2 Data (SRS M1): done 2026-11-27**
- Migrations 01–07, `query.sql` + sqlc.
- `pkg/footballdata`: `mmz4281/{season}/{E0|SP1|D1}.csv`, covering the current and previous season so early-season windows still have 10 matches.
- `pkg/teamnames` + aliases (FR-1.5).
- Fixtures source (D4).
- `ticketdoctor import` upserts matches, recomputes profiles and records an `import_runs` row. Re-running it changes nothing.
- `ticketdoctor seed` for local development.
- Done when an import on an empty database builds profiles for all three leagues, newly promoted teams show as low confidence, and a second run is a no-op.
- **Result:**
  - **Migrations 01–05:** leagues (with the 3 MVP leagues), teams, matches, team_profiles, import_runs. Analysed legs and suggestions move to B3, where they're first used, so their shape is designed with the endpoints.
  - **Matches** are identified by league + season + home + away, so a rescheduled kickoff updates its row instead of duplicating it.
  - **`pkg/footballdata`** handles the BOM, both date formats, UK time (BST and GMT), market-average odds falling back to Bet365, and bad rows (skipped with line numbers).
  - **`pkg/teamnames`:** keys and an alias table; every 2026/27 club and 41 official long names are tested.
  - **`ticketdoctor import`:** each league is imported in its own transaction, recorded in `import_runs`, and safe to re-run.
  - **First live import into the local database:** 1,395 matches, 67 teams, 134 profiles in about 5 s; a second run changed nothing.
  - **Tests:** integration tests on a real Postgres (CI service container) and a migrations round trip, down to zero and back up.
- **Fixtures (FR-1.2), added 2026-11-27:** `pkg/footballdataorg` reads football-data.org (one request per league), from yesterday to `days_ahead` (14) days ahead.
  - Fixtures are saved in the same transaction as the league's results, after them, and **never overwrite a finished match**.
  - Postponed, suspended and cancelled games get status `postponed` (migration 06) and keep their row, so a new date updates it.
  - **Fixtures never create teams.** A club name that matches nothing is skipped and reported, and the run is marked failed, so a missing alias can't create a duplicate club.
  - A fixtures download failure still saves the results, and also marks the run failed.
  - **Live:** all 58 clubs football-data.org lists matched by name with no new aliases; 73 fixtures (30 EPL, 24 La Liga, 19 Bundesliga) imported, and a second run changed nothing.
  - For B3: fixture lists show `status = 'scheduled'` and kick-off in the future.

**B3 API: done 2026-11-27**
- Leagues and fixtures, analyse, accept, wizard, results (§3), with handler tests using `httptest`.
- Done when the frontend runs with `VITE_USE_MOCKS=false` against a local API and every screen works; and a 15-leg analysis returns in under 2 s on the local database (NFR-1).
- **Result:**
  - **Decisions (2026-11-27):** windows in Central Africa Time (addendum A5); the ticket log counts each match once per ticket and keeps the user's pick apart from accepted swaps, and results show the like-for-like group (A6); double chance 12 deferred (addendum C).
  - **Services:** `services/fixtures.go`, `tickets.go`, `wizard.go`, `results.go`. Handlers in `api/leagues`, `api/tickets`, `api/wizard`, `api/results`. User errors are 404/422 problem details; POSTs limited to 30 a minute per IP.
  - **Ticket log rules:**
    - The same pick again (or back after Undo) changes nothing.
    - Accepting a suggestion changes only the final pick.
    - The user choosing another market replaces their pick and drops the old suggestions.
    - A match taken off the ticket is marked `on_ticket = false` and not counted.
  - **Refused:** games already kicked off, finished or postponed.
  - **Results** are settled from the score when read, so a new matchday counts as soon as it's imported. B4 shrinks to checking that against real matchdays.
  - **Frontend:** results show "Picks we offered a swap for"; `updated_at` can be null; the mock uses CAT days.
  - **Checked live:**
    - The real API on the local database, with the production frontend build in headless Chrome: builder, analysis, swap with Undo, and results worked, with no console errors.
    - A 15-leg analysis took about 30 ms, or 0.6 s on the first request while the database connection opens.
  - **For B5:** analysis makes about 2 queries per leg. On a hosted database the cost is network round trips, so the API and database share a region (Frankfurt).

**B4 Settlement and track record (SRS M5): done 2026-11-27**
- ~~The import settles finished matches~~: B3 settles when results are read, so nothing is stored and a corrected score fixes itself.
- ~~Results aggregation and the like-for-like figure~~: done in B3.
- Done when, after a matchday is imported, `/api/results` counts change correctly and pending legs stay pending.
- **Result:**
  - **End-to-end test** (`services/settlement_test.go`): fixtures import → ticket analysed → results import records the score → track record counts it.
    - The result lands on the fixture's own row, not a duplicate, even though the two sources give different kick-off times.
    - A game not yet played stays pending.
    - A corrected score (2-1 to 1-1) turns the leg from won to lost on the next import.
  - **Real matchdays:** on a copy of the local database, legs were logged in all 15 markets for every game played since 1 November (67 games, 9 match days, 1,005 legs). The track record matched an independent SQL settlement in all 45 league × market cells.
  - **Speed:**
    - **Before:** at 200,000 legs and 267,000 suggestions, sending every row to Go took 2.2–3.6 s.
    - **After:** the database now counts legs per league, market and final score, and the engine settles each group once, so the rules still live only in Go. That takes 0.54 s for everything, 0.25 s for one league and 0.07 s for one league and market. At realistic volumes it's a few milliseconds.
    - **If it ever gets slow:** most of the remaining time is checking whether each leg got a suggestion. Storing that as a column on `analysed_legs`, set during analysis, would remove it.

**Release process (added 2026-11-26).** Tag-based releases are in place: `v0.1.0` exists, and `release.yml` checks the tagged commit, builds `ghcr.io/late-night-crew/ticket-doctor-api:<tag>` and creates the GitHub Release. See [RELEASING.md](../RELEASING.md). Every migration from B2 on must follow its backward-compatibility rule.

**B5 Go live (SRS M4): live 2026-11-27 at https://ticket-doctor.onrender.com (`v0.4.0`); setup in [DEPLOY.md](DEPLOY.md)**
- Done when production serves real fixtures and the import has run three days without errors.
- **Decided 2026-11-27: everything on Render**, in Frankfurt, closest to Malawi (D2 changed). Free Postgres expires after 30 days, which is enough for the trial period; after it, decide on a paid plan with real usage data.
- **`render.yaml` (Blueprint):**
  - the database;
  - the API as an image-backed service (`ghcr.io/late-night-crew/ticket-doctor-api`, health check `/api/health`);
  - the frontend as a static site in `frontend/` with auto-deploy off.
- **Same address for the frontend and the API:** the static site forwards `/api/*` to the API. The frontend needs no `VITE_API_URL`, the API needs no CORS for it, and the Content-Security-Policy is `'self'` throughout. Checked with the production build behind the same headers and `/api` proxy in headless Chrome, with no violations.
- **Rate limit sees visitors, not proxies:** `TD_TRUSTED_PROXIES` trusts Render's proxies (only they can reach the container), so the per-IP limit uses each visitor's address.
- **`release.yml` deploy job:**
  - deploys the tag's image (Render API `imageUrl`) and the tag's commit for the frontend (`commitId`);
  - waits until both are live;
  - checks `SITE_URL/api/health` reports the tag.

  Rollback = running the workflow for an older tag. The job skips with a notice until the Render secret and variables exist.
- **`import.yml`:** twice a day (04:15 and 22:15 UTC), it runs `migrate` then `import` from the **latest release's image**, never unreleased code. It skips with a notice until `TD_DATABASE_URL` exists.
- **Repository made public** 2026-11-27, after checking every commit on every branch for the API key and database password (none found).
- **Removed:** `frontend/vercel.json` and every Supabase reference.
- **Went live 2026-11-27:**
  - **First import:** 1,395 results, 67 teams, 134 profiles, 73 fixtures (all clubs matched).
  - **Deploy path tested end to end:** Release for `v0.4.0` deployed both services through Render's API and confirmed the live site reports `v0.4.0`. The Render key has Deploys permission only.
  - **Setup snags, now in DEPLOY.md:**
    - The organisation had to allow public packages first.
    - Another project's free database took the workspace's only free slot, so it was deleted.
- **Left before calling B5 done:** three days of scheduled imports without errors, and a decision on hosting before the database expires (~27 December).

**B6 Admin (F7)**: done 2026-11-27, backend and pages. It goes live with v0.5.0 once `TD_ADMIN_SESSION_SECRET` is set (DEPLOY.md).
- **Accounts:** a `users` table holding a username, a bcrypt hash and a role (only `admin` for now). No email or other personal data, and no public sign-up.
  - The first admin is created at `/admin/setup`. It needs `TD_ADMIN_SETUP_CODE` and only works while no account exists.
  - After that, admins add, disable or enable each other, and change their own passwords, on the Admins page.
  - `ticketdoctor admin create|password|disable|enable|list` does the same from a terminal.
- **Sessions:** an HS256 JWT signed with `TD_ADMIN_SESSION_SECRET`, as in parrot, carried in an HttpOnly, SameSite=Strict cookie limited to `/api/admin`.
  - A changed password or a disabled account ends sessions at once.
  - After 10 failed sign-ins, an address is blocked for 15 minutes.
  - Changes must be sent as JSON, which stops cross-site forms.
  - In production, admin is off until the secret is set.
- **Endpoints (`/api/admin`):** `login`, `logout`, `me`, `dashboard`, `matches?view=awaiting|recent`, `PUT matches/:id/result`, `thresholds`.
- **Manual results (FR-1.4):** only for games that have kicked off. The score is stored with `source = manual`, and each entry, with the score it replaced, is kept in `result_entries`. The results import still wins for leagues it covers. Team profiles pick the result up at the next import. Since A10, football-data.org scores usually fill a result in first, within about 30 minutes on match days.
- **Thresholds:** read-only, showing each value with its SRS default. They change in config and ship with a release (NFR-6), so every change is reviewed.

**Calibration** (2026-11-28): checks whether the estimates are right, before spending on data or models.
- **Backtest:** every finished match replayed in kick-off order, each rated only from earlier games. Run it with `ticketdoctor backtest` or the admin **Calibration** page.
- **Real tickets:** the same report from the estimates people were actually shown, for their own picks or our suggestions.
- **First backtest** (1,009 games):
  - well calibrated from 20% to 80%;
  - overconfident at the extremes: we said 90%+ and it happened 82%; we said under 10% and it happened 19%;
  - win markets beat guessing the league average (skill +0.03 to +0.06);
  - goals markets (BTTS, overs and unders) don't (skill −0.01 to −0.05).
- **Shrinkage experiment** (`ticketdoctor backtest --prior 0,2,4,6,8,10,15,20,30`): each team's rates are pulled towards the league's average at that venue, as if it had *m* extra average games. Results with *m* = 10:
  - Brier 0.2153 → 0.2111;
  - 90%+ picks happened 93% (was 82%);
  - Low band: said 75.6%, happened 75.3% (was 76.9% vs 73.1%);
  - markets no better than the average: 10 → 3 of 15.

  The gains hold from *m* = 6 to 20. The cost: about 10% fewer legs reach the Low band. **Made live 2026-11-28** as the rule `prior_games` (default 10, SRS addendum A8). The engine, importer (`league_profiles`, migration 10), TS mock scorer and parity golden file all changed together; `ticketdoctor backtest` now uses it by default, and `--prior` overrides it.

## 6. Decisions needed

| # | Decision | Recommendation |
|---|---|---|
| D1 | ~~Fiber v2 or v3?~~ | **Decided 2026-11-26: v3** |
| D2 | ~~Database host~~ | ~~Supabase~~ (2026-11-26). **Changed 2026-11-27: Render Postgres**, next to the API, so everything is in one place. The free database expires after 30 days: upgrade or export the ticket log before then ([DEPLOY.md](DEPLOY.md)) |
| D3 | ~~Module name~~ | **`ticketdoctor`** (used in B0) |
| D4 | ~~Fixtures source (FR-1.2)~~ | **Decided 2026-11-27: football-data.org** (free tier, 10 requests a minute). The key goes in `TD_FOOTBALL_DATA_ORG_TOKEN`, or `football_data_org.token` in the untracked `config.yaml`; for B5 it becomes a repository secret for the import cron |
| D5 | ~~Admin auth (open question 6)~~ | **Decided 2026-11-27:** a `users` table with accounts made from the CLI; the parrot-style JWT goes in an HttpOnly cookie (B6). Public users don't get accounts: V2 saved tickets stay link-based (SRS addendum B3) |

Questions 3, 4 and 10 from the frontend plan were decided on 2026-11-27 (SRS addendum C, A5, A6).
