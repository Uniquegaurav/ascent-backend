# Deploying the Ascent backend (Render)

The repo is deploy-ready: a `Dockerfile`, a `render.yaml` Blueprint (web service +
Postgres, auto-wired), env-driven config, and migrations that run on first boot.

> OTP is intentionally in **dev mode** (`000000` always verifies) — treat the URL as a
> **private demo**. To go truly public, wire a real SMS provider first.

## 1. Push to GitHub
A local git repo with commits already exists. Create an empty GitHub repo, then:
```bash
cd ~/AndroidStudioProjects/summit-backend
git remote add origin https://github.com/<you>/summit-backend.git
git push -u origin main
```

## 2. Deploy on Render (Blueprint — recommended)
1. **render.com → New → Blueprint**, connect your GitHub, pick `summit-backend`.
2. Render reads `render.yaml` and proposes: **ascent-api** (Docker web service) +
   **ascent-db** (free Postgres). Click **Apply**.
3. It builds the Dockerfile, creates the DB, injects `DATABASE_URL`, generates
   `JWT_SECRET`, and on boot runs migrations (schema + explore/demo seed).
4. When live, the service shows a URL like `https://ascent-api.onrender.com`.

> No Blueprint? Do it manually: **New → Web Service** → connect repo → Runtime **Docker**
> → Health check path `/health`; then **New → Postgres** (free) and add an env var
> `DATABASE_URL` = the DB's *Internal Connection String*; add `JWT_SECRET` = a random
> string. (Don't set `PORT` — Render injects it; the app already binds `:$PORT`.)

## 3. Verify
```bash
./scripts/smoke.sh https://ascent-api.onrender.com
```
(The free service sleeps when idle, so the first request after a while takes ~30s to wake.)

## 4. Point the app at it
Edit one line — `shared/.../data/remote/PlatformConfig.kt`:
```kotlin
const val DEPLOYED_BASE_URL = "https://ascent-api.onrender.com"
```
Rebuild the app. It now talks to the cloud from the emulator **and real phones** (HTTPS).

---

## Notes
- **Free tier caveats:** the free web service **sleeps after ~15 min idle** (cold start on
  next hit); Render's **free Postgres expires** after its limited window — fine for a demo,
  upgrade the DB when you need permanence.
- **Migrations** run every boot but only apply unseen versions (`schema_migrations`), so
  redeploys/cold-starts are safe. Keep **1 instance** (no migration locking yet).
- **SSL:** Render's internal `DATABASE_URL` connects fine; if you ever use the *external*
  DB URL, append `?sslmode=require`.

## Alternative: Railway CLI
```bash
brew install railway      # or npm i -g @railway/cli
railway login && railway init
railway add --database postgres
railway up && railway domain
# set JWT_SECRET in the dashboard
```
(Railway's `railway.toml` is also in the repo.)
