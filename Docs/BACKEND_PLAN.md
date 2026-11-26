# Ticket Doctor: backend implementation plan

Sources:
- *Ticket Doctor — Software Requirements Specification (MVP)* and [SRS_ADDENDUM.md](SRS_ADDENDUM.md).
- The API contract the frontend already uses: `frontend/src/api/types.ts`.
- The reference rules implementation: `frontend/src/mocks/scorer.ts` and its tests.
- The layout follows the Late Night Crew School Management System backend (`nsms`), which itself follows parrot. See [backend/README.md](../backend/README.md).

## 1. How it fits together

```
football-data.co.uk CSVs ──┐
(fixtures source, see D4) ─┤
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
| 05 | create_analysed_legs | SRS fields + **(+)** `outcome` (won/lost/pending, FR-7.2) |
| 06 | create_suggestions | SRS fields (`accepted`, `outcome`) |
| 07 | create_import_runs **(+)** | `import_runs(id, started_at, finished_at, source, status, rows, error)`: shows whether the daily job worked |

Market values are stored as the same keys as the frontend (`HOME_WIN`, `OVER_2_5`…), with a CHECK constraint.

## 5. Milestones

Each milestone ends with green CI and a pull request.

**B0 Skeleton: done 2026-11-26**
- `go.mod` (module `ticketdoctor`), `main.go`, `cmd/root|server|migrate`, config, logging, problemdetail, `GET /api/health`.
- `Taskfile.yml`, `config.example.yaml`.
- **`.github/workflows/backend.yml`**: `setup-go`, `go vet`, `go test`, `go build`. This is the CI exercise for you.
- Done when `ticketdoctor server` answers `/api/health` and CI is green.
- Also added: `docker-compose.yml` (local Postgres on port 55432), graceful shutdown on SIGTERM, request logging with request IDs, and CORS from config. Config works on environment variables alone, with no file needed on Render. The frontend now reads problem-detail `detail` messages, and the mock API returns the same format.

**B1 Rules engine (SRS M2), no database needed**
- Port `scorer.ts` to `engine/`: profile maths, estimates (4.1 + A1), bands (4.2), swaps (4.3 + A2), combined and value (4.5), wizard (4.6 + A4), and **settle(market, goals) → won/lost**.
- Thresholds come from config (NFR-6).
- Done when the Go tests reproduce every case in `scorer.test.ts`, including the 4.4 worked example (NFR-7). Done first because it is pure, test-led, and the frontend tests already give the expected numbers.

**B2 Data (SRS M1)**
- Migrations 01–07, `query.sql` + sqlc.
- `pkg/footballdata`: `mmz4281/{season}/{E0|SP1|D1}.csv`, covering the current and previous season so early-season windows still have 10 matches.
- `pkg/teamnames` + aliases (FR-1.5).
- Fixtures source (D4).
- `ticketdoctor import` upserts matches, recomputes profiles and records an `import_runs` row. Re-running it changes nothing.
- `ticketdoctor seed` for local development.
- Done when an import on an empty database builds profiles for all three leagues, newly promoted teams show as low confidence, and a second run is a no-op.

**B3 API**
- Leagues and fixtures, analyse, accept, wizard, results (§3), with handler tests using `httptest`.
- Done when the frontend runs with `VITE_USE_MOCKS=false` against a local API and every screen works; and a 15-leg analysis returns in under 2 s on the local database (NFR-1).

**B4 Settlement and track record (SRS M5)**
- The import settles finished matches: legs and suggestions are marked won or lost.
- Results aggregation, including the like-for-like figure if question 10 is agreed.
- Done when, after a matchday is imported, `/api/results` counts change correctly and pending legs stay pending.

**B5 Go live (SRS M4)**
- Database host (D2), `Dockerfile` + `deploy/start.sh` + `render.yaml`.
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
| D4 | **Fixtures source (FR-1.2).** football-data.co.uk also publishes upcoming fixtures as a CSV (no API key); football-data.org needs a free key and allows 10 requests a minute | **Try the CSV first** in B2, with football-data.org as the fallback. Verify coverage for all three leagues before committing |
| D5 | **Admin auth (open question 6)** | Reuse the nsms JWT login with admin users created from config; decide before B6 |

Still open from the frontend plan and relevant here: **3** (double chance 12), **4** (weekend timezone), **10** (like-for-like results). Settle them before B3 and B4.
