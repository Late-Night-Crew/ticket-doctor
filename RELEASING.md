# Releasing Ticket Doctor

Production changes only through **version tags**. Merging to `main` runs CI, but it does not release or deploy anything. A tag does.

## Versions

Tags are [semantic versions](https://semver.org): `vMAJOR.MINOR.PATCH`, with one version for the whole repository (frontend and backend share the API contract, so they're released together).

While the MVP is pre-launch we stay below 1.0:

| Change | Bump | Example |
|---|---|---|
| A milestone or a set of features (B1, B2, …) | minor | `v0.1.0` → `v0.2.0` |
| Fixes only | patch | `v0.2.0` → `v0.2.1` |
| Something to try before releasing properly | pre-release suffix | `v0.3.0-rc.1` (marked as pre-release on GitHub) |
| Public launch of the MVP | `v1.0.0` | |

## How to release

1. Make sure everything for the release is **merged into `main`**, and that **CI is green on `main`** (Actions tab, or `gh run list --branch main`).
2. Pick the next version (table above), and check the last one with `git tag --sort=-v:refname | head -3`.
3. Tag `main` and push the tag:

   ```bash
   git switch main && git pull
   git tag -a v0.2.0 -m "v0.2.0: rules engine (B1)"
   git push origin v0.2.0
   ```

4. Watch the **Release** workflow (Actions tab, or `gh run watch`). It:
   1. checks the tag is a valid version and that its commit is on `main`;
   2. runs the frontend and backend CI checks again on exactly that commit;
   3. builds the API image `ghcr.io/late-night-crew/ticket-doctor-api:v0.2.0`, with the version baked in (`/api/health` reports it);
   4. creates the GitHub Release, with notes generated from the merged pull requests;
   5. *(from B5)* deploys that image to Render and the frontend to Vercel.

   If any step fails, nothing after it runs, so nothing half-released goes out. Fix the problem in a pull request, merge it, and tag the **next** version.

## Rolling back

The previous version's image still exists, unchanged. To go back to it:

- **Actions → Release → Run workflow**, enter the previous tag (e.g. `v0.1.0`), and run. It redeploys that exact version. (Until B5 there's nothing deployed, so this only rebuilds the release.)
- In an emergency, Render and Vercel both have a one-click "Rollback" in their dashboards. Afterwards, check that `main` gets the fix and a new tag.

Then fix forward: correct the problem on `main` and release a new patch version.

## Rules

- **Never move, delete or reuse a tag.** If `v0.2.0` is bad, release `v0.2.1`. People and deploys refer to tags, so a tag must always mean the same code.
- **Only tag commits on `main`.** The workflow refuses anything else.
- **Database migrations must work with the previous release** (rolling back the code doesn't roll back the database):
  - Add things freely: new tables, new nullable columns or columns with defaults, new indexes.
  - Remove or rename in two releases: first a release that stops using the column, then a later release that drops it, once you're sure you won't roll back past the first.
  - Never change a migration file that has already been released; add a new one.

## Where things are

| What | Where |
|---|---|
| Release workflow | [.github/workflows/release.yml](.github/workflows/release.yml) |
| Releases and notes | GitHub → Releases |
| API images | GitHub → the organisation's Packages → `ticket-doctor-api` |
| Image build | [backend/Dockerfile](backend/Dockerfile), [backend/deploy/start.sh](backend/deploy/start.sh) (migrate, then serve) |
