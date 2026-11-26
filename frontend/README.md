# Ticket Doctor: frontend

React (Vite and TypeScript), Bootstrap 5.3 (SCSS) and a custom CSS design layer. See [../Docs/FRONTEND_PLAN.md](../Docs/FRONTEND_PLAN.md) for the plan and the API contract.

## Run

```bash
npm install
npm run dev        # http://localhost:5173 (uses the mock API, see .env.development)
npm test           # Vitest
npm run typecheck
npm run lint       # oxlint
npm run build
```

When the Go API is up, set `VITE_USE_MOCKS=false` and point `VITE_API_URL` at it (see `.env.example`).

During development, open `/dev/styleguide` to see the shared components.

## Layout

```
src/
  api/          types.ts (API contract), client.ts, hooks.ts (React Query)
  config/       copy.ts (all UI text), markets.ts, glossary.ts, rules.ts (thresholds)
  mocks/        MSW handlers, fake data, scorer.ts (TS copy of SRS section 4 rules)
  state/        ticket reducer, provider and useTicket hook (sessionStorage)
  components/   AppLayout, LegCard, RiskBadge, ProbabilityMeter, GlossaryTerm, ...
  pages/        one file per route, lazy-loaded
  styles/       _variables.scss (Bootstrap overrides), _tokens.scss, components/*
```

`mocks/scorer.test.ts` encodes the SRS worked example (4.4). The Go rule tests should assert the same values.

## Deploy (Vercel)

1. Import the repository in Vercel and set the **root directory** to `frontend`. `vercel.json` sets the build, the client-side routing fallback, asset caching and security headers.
2. Set environment variables in the Vercel project:
   - `VITE_API_URL`: the Go API's base URL, e.g. `https://ticket-doctor-api.onrender.com`. The Go API must allow CORS from the Vercel domain.
   - `VITE_USE_MOCKS`: `false` in Production. Set it to `true` in Preview to demo the app on mock data before the API exists.
3. Render's free tier sleeps when idle, so the first API call after a quiet spell can take 30 s or more.

Variables prefixed `VITE_` are compiled into the public bundle. Never put secrets in them; API keys belong only in the Go API (NFR-4).

## Quality checks

- `npm test` runs the unit and integration tests, including a check for banned wording (RG-5).
- Production builds remove unused Bootstrap CSS with PurgeCSS (configured in `vite.config.ts`). If a class added at runtime ever goes missing in production, add it to the safelist there.
