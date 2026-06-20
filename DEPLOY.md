# Deploying the Ascent backend to Railway

The repo is deploy-ready: a `Dockerfile`, `railway.toml` (Dockerfile builder + `/health`
check), env-driven config, and migrations that run automatically on first boot.

> OTP is intentionally left in **dev mode** (`000000` always verifies) — treat the URL as
> a **private demo**. To go truly public, wire a real SMS provider first.

## 1. Push to GitHub
A local git repo with an initial commit already exists. Create an empty GitHub repo, then:
```bash
cd ~/AndroidStudioProjects/summit-backend
git remote add origin https://github.com/<you>/summit-backend.git
git push -u origin main
```

## 2. Create the Railway project
1. Go to **railway.app → New Project → Deploy from GitHub repo** → pick `summit-backend`.
   Railway reads the `Dockerfile` automatically.
2. In the project, **New → Database → Add PostgreSQL**. This creates a `DATABASE_URL`.

## 3. Set variables (on the API service → Variables)
| Variable | Value |
|---|---|
| `DATABASE_URL` | `${{Postgres.DATABASE_URL}}` (reference the Postgres service) |
| `JWT_SECRET` | a long random string (e.g. `openssl rand -hex 32`) |
| `OTP_DEV_CODE` | `000000` (optional; this is the default) |

- **Do not set `PORT`** — Railway injects it and the app already binds `:$PORT`.
- Leave `ENV` unset (defaults to `dev`) so the demo code `000000` keeps working.
- `GOOGLE_PLACES_KEY` — optional; without it, curated sample places are served.

## 4. Deploy + get a URL
- Railway builds and starts the service; on boot it runs migrations (creates the schema +
  seeds the explore catalog and demo climbers).
- **Service → Settings → Networking → Generate Domain** → you get
  `https://<something>.up.railway.app`.

## 5. Verify
```bash
./scripts/smoke.sh https://<something>.up.railway.app
```
Expect the explore sections, an ascent created, a log added, etc.

## 6. Point the app at it
In the Kotlin app, edit one line —
`shared/.../data/remote/PlatformConfig.kt`:
```kotlin
const val DEPLOYED_BASE_URL = "https://<something>.up.railway.app"
```
Rebuild the app. It now talks to the cloud from the emulator **and real phones** (HTTPS,
so no cleartext exception needed).

---

## Alternative: Railway CLI (no GitHub)
```bash
brew install railway        # or: npm i -g @railway/cli
railway login
cd ~/AndroidStudioProjects/summit-backend
railway init
railway add --database postgres
railway up                  # builds the Dockerfile and deploys
railway domain              # generate a public URL
# set JWT_SECRET in the dashboard or: railway variables set JWT_SECRET=...
```

## Notes
- **Migrations** run on every boot but only apply unseen versions (tracked in
  `schema_migrations`), so redeploys are safe. Keep to a single instance while the
  embedded runner is in use (no migration locking yet).
- **Connection SSL**: the `${{Postgres.DATABASE_URL}}` reference uses Railway's private
  network (no SSL needed). If you ever use the *public* DB URL, append `?sslmode=require`.
- **Cost**: Railway's trial then ~$5/mo; the service + Postgres are small.
