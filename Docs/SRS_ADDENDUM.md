# Ticket Doctor — SRS Addendum

Changes and additions to *Ticket Doctor — Software Requirements Specification (MVP)* (Nov 23, 2026). Where this addendum and the SRS disagree, this addendum wins.

Last updated: 2026-11-27.

---

## A. Amendments to the MVP

These were agreed during frontend work. The reference implementation and tests are in `frontend/src/mocks/scorer.ts` and `scorer.test.ts`. The Go rules should match them.

### A1. Estimates for markets the SRS does not define (amends 4.1)

Profiles store only the over 1.5, over 2.5 and under 3.5 rates, so the other markets use complements:

| Market | Estimate |
|---|---|
| BTTS no | 1 − BTTS yes estimate |
| Under 1.5 | 1 − Over 1.5 estimate |
| Under 2.5 | 1 − Over 2.5 estimate |
| Over 3.5 | 1 − Under 3.5 estimate |

### A2. Swaps never contradict the pick (amends 4.3)

Step 2 of 4.3 skips any market that can never win together with the original pick, for example Double chance 1X against an away-win pick, or Under 2.5 against Over 2.5. Without this rule, a High-risk pick would always be offered its own opposite as the "safer" swap. The worked example in 4.4 already behaves this way.

### A3. Number of games in the wizard (amends FR-8.2 and FR-8.3)

| ID | Requirement | Priority |
|---|---|---|
| FR-8.2 | Guided path asks four questions: risk per game (Safe, Balanced, Bold), focus (wins, goals, mix), **number of games (an exact number, 2–10)**, and when (today, this weekend) | Must |
| FR-8.3 | Beginner path asks risk per game, focus and **number of games (2–4)**, in plain words with a one-line explanation under each option | Must |
| FR-8.11 | The generated ticket shows its combined chance next to its fair odds (1 ÷ combined chance), labelled as the app's estimate | Must |

Risk per game and number of games are separate choices. For example, 2 Bold games can have longer odds than 5 Safe games while being less likely to win.

### A4. Risk level chooses the band of each leg (amends 4.6 rules 3 and 5)

As written, 4.6 takes the highest-estimate pick in each match and fills the ticket in estimate order, so a Bold ticket came out the same as a Safe one whenever enough Low legs existed. The rule is now:

| Risk | Legs |
|---|---|
| Safe | All Low |
| Balanced | Half Medium (rounded down), the rest Low |
| Bold | Half High (rounded down, estimate at least 35%), the rest Medium |

- Within a band, the best-supported pick in each match is used; still one leg per match.
- A band that runs short falls back only to safer bands: Bold goes High, then Medium, then Low; Balanced goes Medium, then Low.
- The SRS limits still hold: Balanced has at most half Medium; Bold keeps at least half Low or Medium, with a 35% floor.
- If too few games qualify, the ticket is shorter and says why. The thresholds are never lowered (4.6.6).

### A5. "Today" and "this weekend" are in Central Africa Time (amends FR-1.2 and FR-8.2)

Agreed 2026-11-27. The server decides which fixtures fall in each window, in Central Africa Time (CAT, UTC+2, no daylight saving), so every user sees the same games whatever their device's clock or time zone.

| Window | Fixtures |
|---|---|
| Today | Kick-off from now until midnight CAT |
| This weekend | Kick-off from now until the end of the coming Sunday (CAT), on Saturday or Sunday. On a Saturday or Sunday, that means the rest of the current weekend |
| Upcoming | Every fixture not yet started that has been imported (about two weeks ahead) |

Only games that haven't kicked off can be picked or analysed. Postponed games are left out.

### A6. How the track record counts picks (amends FR-7.1 and FR-7.3)

Agreed 2026-11-27. With no accounts, every tap of **Analyse** sends the whole ticket again, so the log has to tell repeats and accepted swaps apart from new picks:

| ID | Requirement |
|---|---|
| FR-7.1a | Each ticket has a random id made in the browser (no personal data, NFR-8). The log keeps **one row per ticket per match**, however often the ticket is analysed |
| FR-7.1b | The **user's pick** for a match is the market they chose themselves. Accepting a suggestion changes the *final* pick but not the user's pick, so a suggestion is never counted as the user's own. Choosing a different market themselves replaces the user's pick and drops the suggestions made for the old one |
| FR-7.1c | Each suggestion is logged once per ticket and match, with whether it was accepted |
| FR-7.1d | A match removed from the ticket before the last analysis is not counted |
| FR-7.3a | Next to "our suggestions", the results page shows how **the user's picks that received a suggestion** did. Suggestions are only made for Medium and High legs, so this is the like-for-like comparison; comparing with every user pick would flatter the suggestions |

The row per ticket and match (the user's pick, the suggestions shown, the final pick) is also what V2's saved tickets need (FR-9.3).

---

## B. Version 2: saved tickets and pick comparison

**Not in the MVP.** SRS 7.2 lists "user accounts and saved tickets" as out of scope; this section describes how they come back in V2.

### B1. Purpose

Let a user save a ticket and, once every game has been played, compare side by side what they picked first, what Ticket Doctor suggested, and what they finally chose. Show this whether the ticket won or lost. When the user sees that the swaps they were offered tend to win, they have their own evidence for trusting the app, alongside the overall results page (FR-7.3).

### B2. FR-9 Saved tickets and pick comparison

| ID | Requirement | Priority |
|---|---|---|
| FR-9.1 | After analysis, a user can save a ticket (manual or wizard) without an account | Must |
| FR-9.2 | A saved ticket can be reopened on the same device, and from a private link that can be opened on another device | Must |
| FR-9.3 | For each leg, the saved ticket keeps: the original pick, every suggestion shown (accepted or not), and the final pick | Must |
| FR-9.4 | After each match, every original pick, suggestion and final pick is marked won or lost, using the same settlement as FR-7.2 | Must |
| FR-9.5 | A review screen shows each leg side by side: original pick, our suggestions, final pick, and each one's result. It is shown even when the ticket lost | Must |
| FR-9.6 | The review screen sums up the leg results: our suggestion won and your pick lost; your pick won and our suggestion lost; both won; both lost | Must |
| FR-9.7 | The review includes suggestions the user rejected, and cases where the original pick would have won. Misses are shown as clearly as wins (RG-3) | Must |
| FR-9.8 | A "My tickets" list shows saved tickets, labelled pending or settled | Should |
| FR-9.9 | A personal record across saved tickets: how the user's picks did against our suggestions | Should |
| FR-9.10 | A saved ticket can be deleted by its owner | Should |

### B3. Constraints

- **No personal data (NFR-8 still applies).** A saved ticket is identified only by a random id kept in the browser and in its private link. Accounts can come later and are not required for FR-9.
- **No stakes or money** are recorded. The app still never suggests a stake (RG-6).
- **No betting-company links** (SRS 1.2 and 7.2 unchanged).
- **The landing page does not change.** The decision on 2026-11-26 was not to lead with tickets generated by the app or shared by other users. Ticket Doctor stays a reviewer of the user's own ticket (SRS 2.1), not a tipster.

### B4. Data model additions (sketch)

| Table | Key fields | Notes |
|---|---|---|
| saved_tickets | id, created_at, settled_at, origin | id is random and also used in the private link |
| saved_ticket_legs | saved_ticket_id, match_id, original_market, original_odds, final_market, final_odds, original_outcome, final_outcome | outcome = won, lost or pending |

Suggestions shown for each leg already live in `suggestions` (with `accepted` and `outcome`) through `analysed_legs`. A saved leg links to its last `analysed_legs` row.

### B5. Open questions

1. How long are saved tickets kept: until settled plus N days, or indefinitely?
2. Should a private link be view-only for anyone who has it, or does reopening a ticket need the original browser too?
3. With no accounts, there are no "your ticket has settled" notifications; the user has to come back. Is that acceptable for V2?

---

## C. Deferred

- **Double chance 12 (home or away wins).** Decided 2026-11-27 not to offer it in the MVP. Its estimate (1 − the draw estimate) is usually around 70–75%, so it would crowd Safe wizard tickets and swap suggestions with low-paying legs. It may come back later as a deliberate low-pay option, for example a ticket built only from 12 selections. Adding it then means a new market in the engine and the frontend, a settle rule (a draw loses), and a contradiction rule (it can never win together with a draw).
