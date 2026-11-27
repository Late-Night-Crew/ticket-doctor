# Ticket Doctor: frontend implementation plan

Source: *Ticket Doctor — Software Requirements Specification (MVP)*, Nov 23, 2026, plus [SRS_ADDENDUM.md](SRS_ADDENDUM.md), which holds the agreed changes and the V2 saved-tickets feature (FR-9).
Stack: React (Vite, TypeScript), Bootstrap 5.3 (SCSS source) and a custom CSS design layer.

The SRS builds the UI in milestone M3, after the data and analyser core. We are building the frontend first, so:

1. The API contract is fixed up front in `frontend/src/api/types.ts`.
2. Until the Go API exists, the UI runs against an MSW mock API (`frontend/src/mocks/`). The mock includes a TypeScript version of the section 4 rules.
3. The mock scorer's tests (including the worked example in 4.4) are the expected values for the Go unit tests in M2.

## Status

| Phase | Scope | SRS | Status |
|---|---|---|---|
| F0 | Project setup: Vite, TS, Bootstrap SCSS, router, React Query, MSW, Vitest | NFR-2, NFR-6, NFR-7 | Done |
| F1 | Design system and app shell: tokens, layout, bottom nav, footer, shared components, copy guard | RG-1, RG-2, RG-4, RG-5, NFR-5 | Done |
| F2 | Ticket builder | FR-3, FR-2.4 | Done |
| F3 | Analysis screen | FR-4, FR-5, FR-6 | Done |
| F4 | Ticket wizard (beginner and guided paths) | FR-8, RG-6 | Done |
| F5 | Results page | FR-7.3, RG-3 | Done |
| F6 | Performance and accessibility pass, Vercel deploy setup | NFR-2, NFR-3 | Done |
| F7 | Admin: manual result entry, threshold view | FR-1.4 (Should) | Waiting on open question 6 (admin access) |

## API contract

| Method and path | Purpose |
|---|---|
| `GET /api/leagues` | Supported leagues |
| `GET /api/leagues/:id/fixtures?when=today\|weekend\|upcoming` | Upcoming fixtures |
| `POST /api/tickets/analyse` | Score legs, suggest swaps, combined probability |
| `POST /api/suggestions/:id/accept` | Log an accepted swap (FR-7.1) |
| `POST /api/wizard` | Generate a ticket (FR-8); `seed` changes on Regenerate |
| `GET /api/results?league=&market=` | Hit rates of original picks vs swaps |

Request and response types are in `frontend/src/api/types.ts`. Wire fields are snake_case to match Go JSON tags.

Market keys: `HOME_WIN, AWAY_WIN, DRAW, DC_1X, DC_X2, BTTS_YES, BTTS_NO, OVER_1_5, UNDER_1_5, OVER_2_5, UNDER_2_5, OVER_3_5, UNDER_3_5, HOME_TO_SCORE, AWAY_TO_SCORE`.

## Screens

| Route | Screen | Phase |
|---|---|---|
| `/` | Home: two paths (check my ticket, build one for me) | F1 |
| `/ticket` | Builder: league, then fixture, then market (bottom sheet), then optional odds | F2 |
| `/ticket/analysis` | Leg cards, swaps, combined probability before and after | F3 |
| `/wizard/*` | Entry, beginner path, guided path, Bold confirmation, result | F4 |
| `/results` | Public track record, losses shown as clearly as wins | F5 |
| `/glossary` | All terms; also opened in a modal by tapping any market name | F1 |
| `/dev/styleguide` | Component showcase (dev builds only) | F1 |

## Design decisions

- **Mobile-first at 360 px**: a bottom nav on phones and a top nav from 768 px; tap targets of at least 44 px.
- **Risk is never shown by colour alone**: every badge has an icon and a text label.
- **The leg card looks like a ticket stub**: a coloured risk edge, and a perforated divider above the swap suggestions.
- **Performance**: Bootstrap is imported partial by partial, pages are lazy-loaded, and the system font stack means no web-font download (slow 3G, NFR-2). The current build is about 94 KB gzipped JS and 29 KB gzipped CSS.
- **Copy**: all user-facing text lives in `config/copy.ts`. A test fails if "sure win", "banker" or "guaranteed" appears anywhere in `src/` (RG-5).
- **No accounts**: the ticket is kept in `sessionStorage` with a random `ticket_id` (FR-3.5, NFR-8).

## F2 notes

- **Flow:** pick a league (pills), then a window (Today, This weekend, All upcoming), then tap a fixture. A bottom sheet offers every market in plain words, with optional odds.
- **Ticket slip:** sits beside the fixtures on desktop. On phones, a bar above the bottom nav shows the leg count and an Analyse button, and the full slip opens in a sheet.
- **Rules enforced:** one leg per match (picking again updates that leg), and a 15-leg limit with a message when full.
- **Odds** accept a comma or a dot (e.g. 1,9 or 1.9), from 1.01 to 1000. Changing a leg's market clears its old odds.
- **Form strings** appear on fixture rows at 400 px and wider, and always in the sheet header (FR-2.4).
- **URLs:** the league and window are kept in the URL (`/ticket?league=laliga&when=weekend`), so Back and shared links keep the view.
- **Tests:** `pages/Builder.test.tsx` drives the page against the same MSW handlers the dev app uses.

## F3 notes

- **When analysis runs:** it is a cached query keyed on the ticket's contents (`useAnalysis`). The same ticket is never sent twice; this matters because the server logs each analysed leg (FR-7.1). Undoing a swap reuses the earlier result, and the previous result stays on screen, dimmed, while a new one loads.
- **Layout:** the summary comes first on phones and sits in a sticky right-hand column on desktop. It shows:
  - The combined chance as a percentage and "about 1 in N" (RG-2).
  - "was X%" once a swap has been accepted (FR-6.2).
  - A count of High, Medium and Low legs.
  - The weakest leg, with a Show button that scrolls to it (FR-6.3).
  - Combined odds, the bookmaker's implied chance and our fair odds, when every leg has odds (FR-6.4).
- **Leg cards** show:
  - Risk band, the reason and the sample size (NFR-5).
  - Implied probability and a Value tag when odds were entered.
  - Up to 2 swaps, each accepted with one tap (FR-5.3).
  - An explanation when limited data blocks swaps (FR-5.4).
  - A Remove button.
- **Swaps:**
  - The ticket state records each swap: the original pick and odds (for Undo), and the combined chance before the first swap.
  - A swap clears the leg's odds, because they were for the old pick, and the notice says so.
  - Any manual edit (adding, removing or changing a leg) resets the "before" comparison.
  - The server is told about an accepted swap, but if that call fails the user is not blocked.
- **Tests:** `pages/Analysis.test.tsx` covers the empty state, leg details, swap with before and after plus Undo, combined odds, removing a leg, swaps blocked by limited data, and API errors.

## F4 notes

- **Routes:** `/wizard` (pick a path), `/wizard/beginner` and `/wizard/guided`. The current step is in the URL (`?step=`), so the browser Back button works. A step is never shown before the questions ahead of it are answered (for example after a refresh).
- **Beginner path:**
  - A stake warning comes first and must be acknowledged (RG-6).
  - Then risk, with Safe selected by default, and focus, each with a one-line explanation.
  - Choosing Bold opens an "About Bold" screen that needs an extra tap: Keep Bold, or Switch to Safe (FR-8.4).
  - Number of games: 2, 3 or 4 (amended FR-8.3, see agreed addition 11). The beginner path doesn't ask when, so it defaults to this weekend (see open question 9).
- **Guided path:** four questions (risk per game, focus, number of games 2–10, when); tapping an answer moves to the next question (FR-8.2, amended).
- **Result:**
  - Answer chips, the disclaimer and the combined chance.
  - Each leg in plain words with its reason, sample size and fair odds labelled as our estimate (FR-8.6, 4.6.7).
  - Beginners also get the "How this ticket works" card (FR-8.7).
  - When too few matches qualify, a shortfall message (4.6.6).
  - Open in analyser, which asks before replacing an existing ticket (FR-8.8); Regenerate, which sends a new seed (FR-8.9); and Change answers.
- **Risk and reward:** the result shows the combined chance next to the ticket's fair odds (1 ÷ chance), so the trade-off between fewer, riskier games and more, safer games is visible.
- **Contract change:** `WizardRequest.size` (ranges) was replaced by `legs` (an exact number). The API clamps it to 2–10, or 2–4 on the beginner path.
- **Contract change:** `WizardLeg` now carries `sample_size` and `window_n`, so wizard estimates show their sample like everywhere else (NFR-5).
- **Tests:** `pages/Wizard.test.tsx` covers both paths, the Bold confirmation, the exact request sent, Regenerate, an empty result, replacing a ticket, and the answer chips.

## F5 notes

- **Route:** `/results`, filterable by league (buttons) and market (a dropdown). The filters are kept in the URL (`?league=epl&market=OVER_2_5`).
- **Headline:** two cards, original picks and suggested swaps, each showing hit rate, won and lost. Won and lost always use the same size and weight, with colour plus words, and a bar where both segments are visible (RG-3).
- **Breakdown:** one row per market and league, as a table on desktop and stacked cards on phones (one HTML table, restyled with CSS).
- **Honesty:**
  - Hit rates on fewer than `RULES.results.smallSample` (20) settled legs are tagged "small sample".
  - Cells with nothing settled say "None settled yet" instead of showing 0%.
  - Notes under the table explain that unplayed matches are not counted, that past results do not predict future ones, and when the data was last updated.
  - The page says the two groups are not a like-for-like comparison (see open question 10).
- **Tests:** `pages/Results.test.tsx` checks totals against the mock data, won and lost given equal weight, small-sample and unsettled cells, filters (by click and from the URL), the empty state and errors.

## F6 notes

**Accessibility.** axe-core was run in headless Chrome on 17 screens and states: every page, plus the open market sheet, the mobile ticket slip, analysis before and after a swap, the wizard's stake, Bold and result screens, a glossary modal, and the error and warning alerts. Each ran in light and dark mode on mobile and in light mode on desktop. A deliberately broken page confirmed the checker was working. The run ended at **0 violations**. Fixes made along the way:
- **Label in name (WCAG 2.5.3):** fixture rows and market buttons now take their accessible name from their visible text, plus hidden separators, instead of a different `aria-label`.
- **Dialog names:** the glossary modal and the mobile ticket slip are now named dialogs.
- **Contrast:**
  - Primary button in dark mode: #0f766e, 5.5:1.
  - Bootstrap's `danger` colour: darker in light mode (#a3302a, 5.1:1 on alert backgrounds) and remapped in dark mode (#f08a7e, 6.2:1).
  - Dark-mode outline-secondary button.
- **Navigation:**
  - Each route sets the document title.
  - Focus moves to the main content and the page scrolls to the top on navigation; wizard steps do the same per step, focusing the step's heading.
  - The error boundary resets on navigation.

**Performance.** Production build of the home page in headless Chrome with a 4× slower CPU, median of 3 runs:

| | Before F6 | After F6 |
|---|---|---|
| CSS (gzip) | 31.2 KB | 15.7 KB (PurgeCSS, production only) |
| JS, first load (gzip) | 98 KB | 98 KB (85% is React DOM and React Router; app code is 3%) |
| Fast 3G: first paint / load | 2.44 s / 1.94 s | 2.18 s / 1.82 s |
| Slow 3G: first paint / load | 7.78 s / 6.82 s | 6.62 s / 6.35 s |
| Transferred, first visit | 128 KB | 113 KB |

On Slow 3G about 4 s is network round-trip latency (2 s per round trip in the DevTools profile), which no frontend change can remove. Repeat visits load the hashed assets from cache (`immutable`).

**Deploy:**
- `frontend/vercel.json`: client-side routing fallback, one-year immutable cache on `/assets`, security headers.
- `.env.production` defaults to the real API; mocks can be turned on for Preview deploys.
- The MSW worker file and mock code are left out of builds unless mocks are on (checked).

**Tests:** Testing Library's wait was raised to 3 s, because tests that go through the mock API timed out at the 1 s default on a busy machine.

**Not done:** a Content-Security-Policy header, because it needs the final API domain for `connect-src`. Add it once the Render URL is known.

## Agreed SRS additions

Items 1, 2, 11 and 12 were confirmed on 2026-11-26; items 3, 4 and 10 were decided on 2026-11-27. Items 6 to 9 are still open.


1. **Derived markets.** The SRS has no rule for BTTS no, Under 1.5, Under 2.5 or Over 3.5, and profiles only store the over 1.5, over 2.5 and under 3.5 rates. The mock uses complements: BTTS no = 1 − BTTS yes, Under 1.5 = 1 − Over 1.5, Under 2.5 = 1 − Over 2.5, Over 3.5 = 1 − Under 3.5.
2. **Swaps must not contradict the pick.** For a High-risk away win at 25%, the rule in 4.3 would also suggest Double chance 1X at 75%, which is a bet *against* the user's pick. The worked example in 4.4 does not suggest it, so the mock excludes markets that can never win together with the original pick, such as 1X vs away win or over 2.5 vs under 2.5.
3. ~~**Double chance 12**~~ **Decided 2026-11-27: not in the MVP**, possibly later as a low-pay option. See SRS addendum C.
4. ~~**"This weekend"**~~ **Decided 2026-11-27: Central Africa Time**, decided by the server. See SRS addendum A5.
5. ~~Wizard ticket sizes~~: replaced by agreed addition 11 (exact number of games).
6. **Admin access** is not specified. A shared secret on the API is suggested for the MVP.
7. **Analysis runs on an explicit "Analyse" tap**, not on every edit, to stay within the free-tier limits.
8. **Help links in the footer** are UK-based (BeGambleAware, GamCare, Gamblers Anonymous). Add Malawi-specific resources if there are any.
9. **When for beginners.** FR-8.3 says the beginner path asks only risk and focus, so the wizard uses this weekend. Should it use today when today has enough matches?
10. ~~**Fair swap comparison (backend).**~~ **Decided 2026-11-27: yes**, together with counting each match once per ticket. See SRS addendum A6. FR-7.3 compares all original picks with all suggestions, but swaps exist only for Medium and High legs, which flatters the swaps. Suggest `/api/results` also returns, per row, the hit rate of the *original picks that received a suggestion*, so the page can compare like with like.
11. **Number of games in the wizard (amends FR-8.2 and FR-8.3).** Agreed 2026-11-26: the user picks an exact number of games, 2–10 on the guided path and 2–4 on the beginner path. Risk level and number of games are separate choices. If fewer games qualify, the ticket is shorter and says why (4.6.6).
12. **Risk level picks the band, not just the limit (amends 4.6 rules 3 and 5).** As written, 4.6 takes the highest-estimate pick per match and fills by estimate, so Bold tickets came out the same as Safe whenever enough Low legs existed. Agreed 2026-11-26 (from the request for "high risk, high reward with only 2 games"):
    - **Safe:** all Low legs.
    - **Balanced:** half Medium (rounded down), the rest Low.
    - **Bold:** half High (rounded down, at least 35%), the rest Medium.
    - Within a band, the best-supported pick per match wins. A short band falls back only to safer bands (Bold: High, then Medium, then Low; Balanced: Medium, then Low).
    - The original limits still hold: Balanced has at most half Medium; Bold keeps at least half Low or Medium, with a 35% floor.
    - Reference implementation and tests: `buildWizardTicket` in `frontend/src/mocks/scorer.ts`. The Go version should match it.

