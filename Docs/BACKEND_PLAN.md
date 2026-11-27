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
           ticketdoctor import   (GitHub Actions, daily cron)
           1. upsert matches (results + upcoming fixtures)
           2. settle analysed legs and suggestions (FR-7.2)
           3. recompute team_profiles (FR-2)
                           │
                           ▼
                    PostgreSQL ◄──────────── ticketdoctor server (Render)
                                               reads cached rows only,
                                               runs engine/ per request
                                                       ▲
                                    React app (Vercel) ┘ /api/...
```

- **Ticket requests never call an outside API** (SRS 2.3, NFR-1). A 15-leg analysis is two queries (matches, profiles) plus pure engine code.
- **The import runs from GitHub Actions on a schedule, not inside the server.** Render's free tier sleeps when idle, so an in-process scheduler would miss runs. A daily database write also stops a Supabase free database from pausing.

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
  - **For B5:** analysis makes about 2 queries per leg. On Supabase the cost is network round trips, so keep the API and database in the same region.

**B4 Settlement and track record (SRS M5)**
- ~~The import settles finished matches~~: B3 settles when results are read, so nothing is stored and a corrected score fixes itself.
- ~~Results aggregation and the like-for-like figure~~: done in B3.
- Left: check it against a real matchday, and decide whether to store outcomes later for speed (only if `/api/results` gets slow).
- Done when, after a matchday is imported, `/api/results` counts change correctly and pending legs stay pending.

**Release process (added 2026-11-26).** Tag-based releases are in place: `v0.1.0` exists, and `release.yml` checks the tagged commit, builds `ghcr.io/late-night-crew/ticket-doctor-api:<tag>` and creates the GitHub Release. See [RELEASING.md](../RELEASING.md). Every migration from B2 on must follow its backward-compatibility rule.

**B5 Go live (SRS M4)**
- Database host (D2). `Dockerfile` and `deploy/start.sh` already exist (release tooling).
- Render as an image-backed service pulling from ghcr.io. The package is private while the repo is private, so Render needs registry credentials until then.
- `release.yml` deploy jobs: deploy the tagged image to Render and the frontend to Vercel, with Vercel's Git production deploys switched off.
- `.github/workflows/import.yml` (daily cron, `TD_DATABASE_URL` stored as a repository secret).
- Vercel `VITE_API_URL`, CORS, and the frontend Content-Security-Policy header.
- Make the repository public (needed for Render's free tier).
- Done when production serves real fixtures and the cron has run three days without errors.

**B6 Admin (F7)**
- Manual result entry, threshold view.
- After D5 and open question 6.

## 6. Decisions needed

| # | Decision | Recommendation |
|---|---|---|
| D1 | ~~Fiber v2 or v3?~~ | **Decided 2026-11-26: v3** |
| D2 | ~~Database host~~ | **Decided 2026-11-26: Supabase** (Postgres). The daily import keeps the free tier from pausing; the API connects with the connection string in `TD_DATABASE_URL` |
| D3 | ~~Module name~~ | **`ticketdoctor`** (used in B0) |
| D4 | ~~Fixtures source (FR-1.2)~~ | **Decided 2026-11-27: football-data.org** (free tier, 10 requests a minute). The key goes in `TD_FOOTBALL_DATA_ORG_TOKEN`, or `football_data_org.token` in the untracked `config.yaml`; for B5 it becomes a repository secret for the import cron |
| D5 | **Admin auth (open question 6)** | Reuse the nsms JWT login with admin users created from config; decide before B6 |

Questions 3, 4 and 10 from the frontend plan were decided on 2026-11-27 (SRS addendum C, A5, A6).
