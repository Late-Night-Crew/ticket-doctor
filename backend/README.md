# Ticket Doctor: backend

Go API for Ticket Doctor. It has the same layout as the Late Night Crew School Management System backend (and parrot), so it should feel familiar. The plan is in [../Docs/BACKEND_PLAN.md](../Docs/BACKEND_PLAN.md).

> Status: **B0 done.** CLI (`server`, `migrate`), config, logging, problem details, `GET /api/health`, CI. Other packages are placeholders with a `doc.go`.

## Run it

Uses your local Postgres, set up like the School Management System's: same `postgres` login on `localhost:5432`, separate `ticketdoctor` database.

```bash
cd backend
psql -U postgres -h localhost -c "CREATE DATABASE ticketdoctor"   # once
cp config.example.yaml config.yaml                                  # once; fill in the password
task migrate      # apply migrations
task dev          # API on http://127.0.0.1:8080, try /api/health
task check        # everything CI runs
```

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
| `pkg/footballdata/` | football-data.co.uk CSV client and parser | like parrot's `pkg/<provider>` |
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
