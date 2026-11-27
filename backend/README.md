# Ticket Doctor: backend

Go API for Ticket Doctor. It has the same layout as the Late Night Crew School Management System backend (and parrot), so it should feel familiar. The plan is in [../Docs/BACKEND_PLAN.md](../Docs/BACKEND_PLAN.md).

> Status: **B0 to B3 done.** CLI (`server`, `migrate`, `import`), rules engine, results and fixtures import, team profiles, and every endpoint the frontend uses. Settlement and go-live (B4, B5) are next (see the plan).

## Run it

Uses your local Postgres, set up like the School Management System's: same `postgres` login on `localhost:5432`, separate `ticketdoctor` database.

```bash
cd backend
psql -U postgres -h localhost -c "CREATE DATABASE ticketdoctor"   # once
cp config.example.yaml config.yaml                                  # once; fill in the password
task migrate      # apply migrations
task import       # results and upcoming fixtures for the 3 leagues, rebuild team profiles (~5 s)
task dev          # API on http://127.0.0.1:8080, try /api/leagues
task check        # everything CI runs
```

The server needs the database. To run the frontend against it instead of the mock API, start it with `VITE_USE_MOCKS=false VITE_API_URL=http://localhost:8080 npm run dev` (in `frontend/`). The API only answers browsers from `server.allowed_origins`, which defaults to `http://localhost:5173`.

| Endpoint | What it does |
|---|---|
| `GET /api/leagues` | The supported leagues |
| `GET /api/leagues/:id/fixtures?when=today\|weekend\|upcoming` | Games not yet started; "today" and "weekend" in Central Africa Time |
| `POST /api/tickets/analyse` | Rates 1–15 legs, suggests swaps, logs the ticket (one row per ticket and match) |
| `POST /api/suggestions/:id/accept` | Records an accepted swap |
| `POST /api/wizard` | Builds a ticket from the rules |
| `GET /api/results?league=&market=` | Track record, including the like-for-like group |

POST requests are limited to 30 a minute per IP. Errors are problem details whose `detail` the frontend shows as is.

Fixtures need a free football-data.org API key in `football_data_org.token` (your `config.yaml`) or `TD_FOOTBALL_DATA_ORG_TOKEN`; without one, a development import loads results only.

`task import -- --league epl --season 2627` imports one league or season; `--from-dir` reads CSVs from disk instead of downloading; `--no-fixtures` skips fixtures. Re-running is safe: unchanged matches are left alone, and every run is recorded in `import_runs`.

### Database tests

Tests that need Postgres use a separate database and are skipped unless `TD_TEST_DATABASE_URL` is set. CI always runs them, against a Postgres service container.

```bash
psql -U postgres -h localhost -c "CREATE DATABASE ticketdoctor_test"   # once
export TD_TEST_DATABASE_URL="postgres://postgres:PASSWORD@localhost:5432/ticketdoctor_test?sslmode=disable"
task test
```

Never point it at your development database: the tests empty the tables.

`config.yaml` is gitignored, so the password stays on your machine. No local Postgres? `task db:up` starts one in Docker on port 55432 (see `config.example.yaml`). Supabase comes in B5, through `TD_DATABASE_URL`.

## Layout

| Path | What | Like in nsms |
|---|---|---|
| `main.go` | Calls `cmd.Execute()` | same |
| `cmd/` | Cobra CLI: `server`, `migrate`, `seed`, `import` | same, plus `import` |
| `config/` | Viper config from `config.yaml`, `TD_*` environment overrides, rule thresholds | same |
| `api/` | Fiber app: `api.go` (wiring), `routes.go`, `middleware.go` | same |
| `api/<area>/` | Handlers per area: `leagues`, `tickets`, `wizard`, `results`, `admin` | same pattern |
| `services/` | Business logic behind interfaces, one file per area | same |
| `schema/` | Request and response structs; must match `frontend/src/api/types.ts` | same |
| `engine/` | **New.** Pure rules engine (profiles, estimates, bands, swaps, wizard), no DB or HTTP | grading and promotion rules, kept separate |
| `pkg/problemdetail/` | RFC 7807 errors | same (one copy only) |
| `pkg/footballdata/` | football-data.co.uk CSV client and parser (results, odds) | like parrot's `pkg/<provider>` |
| `pkg/footballdataorg/` | football-data.org API client (upcoming fixtures) | like parrot's `pkg/<provider>` |
| `pkg/teamnames/` | Club name normalisation across sources | new |
| `common/` | Request parsing and validation, SQL and HTTP helpers | same |
| `logging/` | Logger and module names | same |
| `db/` | `embed.go` (migrations and seeds in the binary), `connect.go` (pgx v5 pool) | same |
| `config.yaml` | Local settings and password (**not committed**) | same |
| `db/migrations/` | Goose SQL migrations `NN_description.sql` | same |
| `db/seeds/` | Demo data for `ticketdoctor seed` | same |
| `database/dbsqlc/` | sqlc output; generated, never edited by hand | same |
| `query.sql`, `sqlc.yaml` | SQL queries and sqlc config | same |
| `deploy/start.sh`, `Dockerfile` | Render container: write config from environment, migrate, serve | same |
| `data/` | Local logs (not committed) | same |
