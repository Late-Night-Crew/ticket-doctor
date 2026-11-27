# Going live on Render

Everything runs on Render's free plan, in Frankfurt (the closest region to Malawi). One file, [`render.yaml`](../render.yaml), describes it all:

| Render service | What it is | Free plan |
|---|---|---|
| `ticket-doctor` | The React app (static site on Render's CDN). Forwards `/api/*` to the API | Free, always on |
| `ticket-doctor-api` | The Go API, running the release image from `ghcr.io` | Sleeps after 15 minutes idle; the first request then takes about a minute |
| `ticket-doctor-db` | PostgreSQL 17, 1 GB | **Expires 30 days after creation.** You then have 14 days to upgrade before Render deletes it |

GitHub does the rest:
- **Release workflow:** deploys each version tag ([RELEASING.md](../RELEASING.md)).
- **Import workflow:** loads results and fixtures twice a day ([`import.yml`](../.github/workflows/import.yml)).

Secrets never go in the repository, in chat or in commit messages. Store them in GitHub's encrypted secrets, as below.

## One-time setup

**Done on 2026-11-27:** live at https://ticket-doctor.onrender.com (`v0.4.0`). The database expires around **27 December 2026**.

Do these in order when setting it up again, e.g. in a new workspace.

### 1. Release v0.4.0 (Claude)
Merge the B4 and B5 pull requests and tag `v0.4.0`. The Release workflow builds the image `ghcr.io/late-night-crew/ticket-doctor-api:v0.4.0`, which Render's first deploy uses. Its deploy step will say "Render isn't set up yet" and skip; that's expected.

### 2. Make the API image public
Render pulls the image without logging in to GitHub, so it must be public. Being public is fine: it contains only the compiled API, and no secrets.

1. Open the organisation's **Packages** tab ([github.com/orgs/Late-Night-Crew/packages](https://github.com/orgs/Late-Night-Crew/packages)) → `ticket-doctor-api`. It isn't linked to the repository, so it doesn't show in the repo's sidebar.
2. **Package settings** (right side) → **Danger Zone** → **Change visibility** → **Public**, and confirm.

   If "Public" is greyed out ("disabled by organization administrators"), an organisation owner must first allow it: organisation **Settings → Packages → Package creation → Public**.

Until the image is public, Render reports it as "not found": a private image looks the same as a missing one.

### 3. Create everything on Render from the Blueprint
1. Sign up at [render.com](https://render.com) with **GitHub**, and allow Render to see the `Late-Night-Crew/ticket-doctor` repository. As an organisation repository, it may need an organisation owner's approval.
2. **New → Blueprint**, pick `ticket-doctor`, branch `main`. Render reads `render.yaml` and lists the three resources above.
3. Give the Blueprint a name, e.g. `ticket-doctor`, and click **Deploy Blueprint**.

   Render creates the database, starts the API (which creates the tables on start), and builds the site. The first run takes a few minutes.

   **"Cannot have more than one active free tier database"** means the workspace already has a free database, perhaps from another project. Render allows one per workspace. Delete it (back it up first with `pg_dump` if needed), or use a separate workspace. Then **Blueprints → ticket-doctor → Manual Sync**. The sync creates only what's missing.
4. **Check the two addresses.** They should be `https://ticket-doctor.onrender.com` and `https://ticket-doctor-api.onrender.com`. If a name was taken, Render adds a suffix (e.g. `ticket-doctor-x1y2.onrender.com`). In that case, tell Claude the real addresses so `render.yaml` can be updated: they appear in the API rewrite and in `TD_ALLOWED_ORIGINS`.
5. **Turn Blueprint auto-sync off:** Blueprints → `ticket-doctor` → Settings → **Auto Sync: No**. Releases decide what's deployed; without this, editing `render.yaml` would redeploy the image tag written in it.

At this point the site loads but has no matches: the database is empty until the first import.

### 4. Load the data (first import)
1. Render → `ticket-doctor-db` → **Connect** → copy the **External Database URL**. It contains the database password.
2. Save it, and your football-data.org key, as repository secrets. In a terminal in the project:

   ```bash
   gh secret set TD_DATABASE_URL --repo Late-Night-Crew/ticket-doctor            # paste the URL when asked
   gh secret set TD_FOOTBALL_DATA_ORG_TOKEN --repo Late-Night-Crew/ticket-doctor # paste the key when asked
   ```

   `gh secret set` asks for the value and never shows it, so it stays out of your shell history. (Or: GitHub → repository → Settings → Secrets and variables → Actions → **New repository secret**.)
3. GitHub → **Actions → Import → Run workflow**. It takes about a minute. Then the site shows fixtures.

From now on the import runs by itself at 06:15 and 00:15 Malawi time.

### 5. Let releases deploy
1. Render → **Account Settings → API Keys → Create API key**. Give it only **Deploys** (Deployment information: Read, Run deployments: Run) and everything else off. The key works across the whole workspace, so with only Deploys a leaked key can't read databases or change settings. Save it as a secret:

   ```bash
   gh secret set RENDER_API_KEY --repo Late-Night-Crew/ticket-doctor
   ```

2. Find each service's id: open the service on Render. The id starts with `srv-` and appears in the address bar and in the service's Settings. Save them as **variables**, which aren't secret:

   ```bash
   gh variable set RENDER_API_SERVICE_ID  --repo Late-Night-Crew/ticket-doctor --body srv-...   # ticket-doctor-api
   gh variable set RENDER_SITE_SERVICE_ID --repo Late-Night-Crew/ticket-doctor --body srv-...   # ticket-doctor
   gh variable set SITE_URL               --repo Late-Night-Crew/ticket-doctor --body https://ticket-doctor.onrender.com
   ```

3. Test it: **Actions → Release → Run workflow**, tag `v0.4.0`. The run:
   1. deploys the API image and the frontend commit;
   2. waits for both to go live;
   3. checks that `https://ticket-doctor.onrender.com/api/health` reports `v0.4.0`.

   The run's page shows a **production** environment.

No environment variables need setting by hand on Render: `render.yaml` sets them all, including the database address. The only secrets are the three GitHub ones above.

Setup is done. From here on, a version tag is the only way anything reaches production, and running the Release workflow for an older tag rolls back to it.

## The 30-day database limit

The free database expires **30 days after it was created**. Render emails before it does. The date is on the database's page. Before then, decide what to do:

- **Keep going on Render:** upgrade the database to a paid plan, from about $7 a month. Nothing else changes.
- **Move:** create the new database, point `TD_DATABASE_URL` (in Render's API service and in the GitHub secret) at it, deploy, and run the import. Matches, teams and profiles come back from the import. **The ticket log does not**, so export it first.

Exporting the ticket log (the data worth keeping: every analysed leg and suggestion). Use PostgreSQL 17's `pg_dump` and the External Database URL:

```bash
pg_dump "$EXTERNAL_DATABASE_URL" --data-only --table=analysed_legs --table=suggestions \
  --table=matches --table=teams > ticket-log-$(date +%F).sql
```

`matches` and `teams` are included because the legs point to them. Keep the file private. It holds no personal data (NFR-8), but it's the app's track record.

## Day to day

| Task | How |
|---|---|
| Release | Tag on `main` ([RELEASING.md](../RELEASING.md)); the Release workflow deploys it |
| Roll back | Actions → Release → Run workflow → the previous tag |
| Import now | Actions → Import → Run workflow |
| Logs | Render → `ticket-doctor-api` → Logs (JSON, one line per request) |
| Is it up? | `https://ticket-doctor.onrender.com/api/health` shows the running version |
| Free hours | All free web services in the workspace share 750 hours a month (sleeping ones use none). If they run out, Render suspends them until the 1st |
