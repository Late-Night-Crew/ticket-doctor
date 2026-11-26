# Ticket Doctor

A web app that reviews a football bet ticket leg by leg, flags risky picks, and suggests better-supported markets in the same match, using recent team statistics. It does not take bets, handle money, or link to bookmakers. Estimates are not guarantees.

| Folder | What |
|---|---|
| [`Docs/`](Docs) | The SRS (PDF), the [SRS addendum](Docs/SRS_ADDENDUM.md) with agreed changes and V2, and the [frontend plan](Docs/FRONTEND_PLAN.md) |
| [`frontend/`](frontend) | React (Vite and TypeScript), Bootstrap and custom CSS. Runs on a mock API until the Go backend exists. See [frontend/README.md](frontend/README.md) |
| [`backend/`](backend) | Go API, scaffolded with the same layout as the Late Night Crew backend. Plan: [Docs/BACKEND_PLAN.md](Docs/BACKEND_PLAN.md) |

## Releases

Production changes only through version tags (`v0.2.0`); see [RELEASING.md](RELEASING.md) for how to release, roll back and write safe migrations.

## Quick start

```bash
cd frontend
npm install
npm run dev     # http://localhost:5173, with mock data
npm test
```

18+ only. Gambling can be addictive. Help: [BeGambleAware](https://www.begambleaware.org) · [GamCare](https://www.gamcare.org.uk)
