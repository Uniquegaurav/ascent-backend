# Summit Backend

Go REST API for the **Summit** app — replaces the app's offline mock engine with a
real server (auth, profiles, range, experiences; friends/challenges/discovery next).

## Stack
- **Go** + **chi** router (stdlib `net/http`)
- **Postgres** via **pgx/v5** (hand-written SQL, no ORM/codegen)
- **JWT** auth (`golang-jwt/v5`) with **OTP** login (dev: code is logged + a fixed
  `OTP_DEV_CODE` always works; prod: implement `SMSSender` with Twilio/MSG91)
- Embedded SQL migrations (run automatically on startup — no `migrate` CLI needed)
- Structured logging (`slog`), graceful shutdown, CORS

## Layout
```
cmd/api/            entrypoint (config, db, migrate, serve)
internal/
  config/           env config
  db/               pgx pool + embedded migration runner + migrations/*.sql
  httpx/            JSON helpers + middleware (recover, logger, CORS, auth)
  domain/           models (JSON tags mirror the app's DTOs)
  auth/             OTP + JWT + handlers
  user/             onboarding + profile
  experience/       log moments, list, feed
  rangex/           derives peaks + milestones from interests + experiences
  server/           router wiring
```

## Run

**Option A — Docker (everything):**
```bash
docker compose up --build      # API on :8080, Postgres on :5432
```

**Option B — local Go + Dockerized Postgres:**
```bash
cp .env.example .env
docker compose up -d db        # just Postgres
go mod tidy
go run ./cmd/api               # migrations apply on startup
```

> Prereqs: Go 1.23+, and Docker (or a local Postgres reachable via `DATABASE_URL`).

## API

| Method | Path | Auth | Body |
|---|---|---|---|
| GET  | `/health` | – | – |
| POST | `/auth/request-otp` | – | `{ "phone": "+91..." }` |
| POST | `/auth/verify-otp` | – | `{ "phone": "...", "code": "000000" }` → `{ token, user }` |
| GET  | `/users/me` | ✓ | – |
| POST | `/users/onboarding` | ✓ | `{ "name": "...", "interestIds": ["trekking", ...] }` |
| GET  | `/range` | ✓ | – |
| GET  | `/experiences` | ✓ | – |
| POST | `/experiences` | ✓ | `{ title, description, interestId, moodScore, imageUrls, location }` |
| GET  | `/experiences/feed` | ✓ | – |

Authenticated requests send `Authorization: Bearer <token>`.

### Quick smoke test
```bash
curl -s localhost:8080/health
curl -s -XPOST localhost:8080/auth/request-otp -d '{"phone":"+919999999999"}'
TOKEN=$(curl -s -XPOST localhost:8080/auth/verify-otp \
  -d '{"phone":"+919999999999","code":"000000"}' | jq -r .token)
curl -s -XPOST localhost:8080/users/onboarding -H "Authorization: Bearer $TOKEN" \
  -d '{"name":"Gaurav","interestIds":["trekking","music","gaming"]}'
curl -s localhost:8080/range -H "Authorization: Bearer $TOKEN" | jq
```

## Wiring the app to this server
In the Kotlin app, `data/remote/HttpClientFactory.kt` currently uses a `MockEngine`.
Swap it for a real engine (OkHttp/Darwin), set the base URL to this server, and add
the OTP calls in `AuthRepositoryImpl`. JSON shapes already match the app's DTOs.

## Roadmap
- [ ] Friends + contact sync (`/friends/*`)
- [ ] Challenges (`/challenges/*`)
- [ ] Discovery proxy (`/places`, `/events`) over Google Places
- [ ] Refresh tokens, rate limiting on OTP, request validation
- [ ] sqlc migration of the data layer; tests
