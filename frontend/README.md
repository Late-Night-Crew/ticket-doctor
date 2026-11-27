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

The admin pages (`/admin`, B6) sign in with a cookie, which needs the app and API on one address. Against a local Go API, leave `VITE_API_URL` empty and set `VITE_API_PROXY=http://localhost:8080` so Vite forwards `/api` to it. With mocks on, sign in as `admin` / `mock password 123`.

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

## Deploy (Render)

The frontend is a Render static site defined in [`../render.yaml`](../render.yaml), which sets:
- the build (`npm ci && npm run build` in `frontend/`, Node 22);
- the client-side routing fallback;
- asset caching;
- the security headers, including the Content-Security-Policy.

It forwards `/api/*` to the Go API, so the app calls its own address and `VITE_API_URL` stays empty. Only releases deploy it (the Release workflow deploys the tagged commit). The one-time setup is in [`../Docs/DEPLOY.md`](../Docs/DEPLOY.md).

The free API sleeps when idle, so the first call after a quiet spell can take up to a minute.

Variables prefixed `VITE_` are compiled into the public bundle. Never put secrets in them; API keys belong only in the Go API (NFR-4).

## Quality checks

- `npm test` runs the unit and integration tests, including a check for banned wording (RG-5).
- Production builds remove unused Bootstrap CSS with PurgeCSS (configured in `vite.config.ts`). If a class added at runtime ever goes missing in production, add it to the safelist there.
