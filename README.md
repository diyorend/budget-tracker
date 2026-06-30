# Budget Tracker

A personal finance tracker with real-time budget alerts. Set a monthly spending limit per category, log transactions, and get an instant WebSocket notification the moment you cross 80% or 100% of a budget — pushed the second it happens, not on page refresh.

Built to practice production-style backend patterns in Go: concurrent WebSocket fan-out, Redis pub/sub, interface-driven architecture for testability, and clean separation between transport, business logic, and data access.

## Features

- JWT authentication (register / login)
- Add transactions by category with date and description
- Set monthly budget limits per category
- Real-time WebSocket alerts at 80% and 100% of a budget, fanned out via Redis pub/sub
- Spending-vs-budget chart by category
- Fully containerized: one `docker-compose up` runs the whole stack

## Tech Stack

**Backend:** Go 1.26 · Echo · pgx v5 (raw SQL, no ORM) · Redis pub/sub · gorilla/websocket · JWT (golang-jwt/v5) · bcrypt · `log/slog`
**Frontend:** React 18 · TypeScript · Vite · Bun (package manager + runtime) · Recharts · react-hot-toast
**Infrastructure:** Docker (multi-stage builds) · Docker Compose · Nginx (reverse proxy, WebSocket-aware) · PostgreSQL 16 · Redis 7

## Architecture

```
┌─────────────┐    ┌──────────────────────────────────────┐    ┌──────────┐
│   Browser   │    │              Go Backend              │    │          │
│             │◄──►│  Echo Router                         │◄──►│ Postgres │
│  React/TS   │    │  ├── JWT Middleware                  │    │  (pgx)   │
│  (Vite/Bun) │    │  ├── Handlers (auth/tx/budget/ws)    │    └──────────┘
│             │    │  ├── Services (business logic)       │
│  WebSocket  │◄──►│  ├── Alert Broker (Redis pub/sub)    │◄──►│  Redis   │
│  client     │    │  └── Repositories (pgx queries)      │    │          │
└─────────────┘    └──────────────────────────────────────┘    └──────────┘
         │                                ▲
         └──── Nginx (port 80, reverse proxy) ───┘
```

Request flow for a transaction that crosses a budget threshold:

```
POST /api/transactions
  → handler validates input, extracts userID from JWT claims
  → service.Create() inserts the transaction (pgx)
  → service spawns a goroutine (fresh context.Background(), not the request ctx —
    the request ctx dies the instant the handler returns)
  → goroutine sums spending for that category this month, compares to budget
  → if over 80%/100%, publishes an AlertMessage to Redis channel "alerts:{userID}"
  → Broker.Run() (subscribed to "alerts:*" via PSubscribe) receives it
  → Broker fans it out to every WebSocket connection registered for that user
  → frontend's useAlertWebSocket hook receives it, shows a toast
```

The HTTP response for the original `POST /api/transactions` returns immediately — the budget check happens asynchronously, so creating a transaction never gets slower because of how many budgets there are to check.

## Why these architectural choices

**Redis pub/sub instead of an in-process Go channel.** A channel only works inside one process. The moment there's more than one backend instance behind a load balancer, a user's WebSocket connection might be on instance A while the alert-triggering transaction lands on instance B. Redis pub/sub crosses that boundary — any instance can publish, whichever instance holds the WebSocket picks it up. Overkill for a single-instance side project, but it's the difference between a toy and something that reflects how this would actually need to work at scale.

**pgx over an ORM.** Every query is hand-written SQL. This means query plans, indexes, and `EXPLAIN ANALYZE` output are all directly inspectable — there's no generated query to reverse-engineer when something is slow. Trade-off: more boilerplate per query, no automatic migrations. Acceptable for a project this size.

**JWT passed as a query parameter for WebSocket auth, not just a header.** Browsers' native WebSocket API cannot set custom headers on the upgrade request — there's no way to send `Authorization: Bearer <token>` the way a normal REST call does. So the JWT middleware checks the `Authorization` header first (used by all REST endpoints) and falls back to a `?token=` query parameter (used only by the `/ws` upgrade request). This is a known, accepted pattern for browser-based WebSocket auth — the alternative (a custom handshake message after connecting) adds complexity without a meaningful security improvement for this use case.

**Bun instead of npm/Node for the frontend.** Faster install and dev-server startup, single binary, native TypeScript execution. The Dockerfile's build stage uses `oven/bun:1-alpine` and runs `bun install --frozen-lockfile` against the committed `bun.lock`.

**Interfaces between service and repository layers** (`repository.UserStore`, `TransactionStore`, `BudgetStore`, and `service.AlertPublisher`). Services depend on these interfaces, not on concrete `*repository.X` structs. This is what makes the test suite possible without a live Postgres or Redis connection — `internal/service/budget_service_test.go` and `internal/handler/auth_handler_test.go` both use in-memory mocks that implement these interfaces.

## API Reference

All authenticated routes require `Authorization: Bearer <token>`.

| Method | Path | Auth | Description |
|---|---|---|---|
| `GET` | `/health` | No | Returns `200 {"status":"ok"}` if the DB is reachable, `503` otherwise |
| `POST` | `/api/auth/register` | No | Body: `{email, password}`. Password min 8 chars. Returns the created user |
| `POST` | `/api/auth/login` | No | Body: `{email, password}`. Returns `{token, user}` |
| `GET` | `/api/transactions` | Yes | Query: `?limit=20&offset=0`. Returns the user's transactions, newest first |
| `POST` | `/api/transactions` | Yes | Body: `{amount, category, description?, date?}` (date format `2006-01-02`) |
| `POST` | `/api/budgets` | Yes | Body: `{category, limit_amount, month?}` (month format `2006-01`, e.g. `2026-01`) |
| `GET` | `/api/budgets/status` | Yes | Query: `?month=2026-01`. Returns each budget with current spend/remaining/percentage |
| `GET` | `/ws` | Yes (`?token=` query param) | Upgrades to WebSocket. Pushes `AlertMessage` JSON when a budget threshold is crossed |

## Running locally with Docker

```bash
docker-compose up --build
# Visit http://localhost
```

This starts Postgres, Redis, the Go backend, the React frontend, and an Nginx reverse proxy that routes `/api/*` and `/ws` to the backend and everything else to the frontend.

## Running without Docker

```bash
# Start dependencies
docker run -d -p 5432:5432 \
  -e POSTGRES_USER=postgres -e POSTGRES_PASSWORD=postgres -e POSTGRES_DB=budget_tracker \
  postgres:16-alpine
docker run -d -p 6379:6379 redis:7-alpine

# Apply migrations
for f in backend/migrations/*.sql; do
  docker exec -i <postgres-container-id> psql -U postgres -d budget_tracker < "$f"
done

# Backend
cd backend
cp .env.example .env   # edit values as needed
go run ./cmd/server/main.go

# Frontend
cd frontend
bun install
bun run dev
```

## Environment Variables (backend)

| Variable | Default | Description |
|---|---|---|
| `PORT` | `8080` | HTTP port the Go server listens on |
| `DATABASE_URL` | — (required) | Postgres connection string |
| `REDIS_URL` | `redis://localhost:6379` | Redis connection string |
| `JWT_SECRET` | — (required) | Signing secret for JWTs, 32+ characters in production |
| `JWT_EXPIRY_HOURS` | `24` | Token lifetime |

## Testing

```bash
cd backend
go test ./...
```

Tests use the repository/service interfaces with in-memory mocks — no live Postgres or Redis required. Coverage includes budget percentage calculation, the full `TransactionService.Create` flow (including the asynchronous budget-threshold check and Redis publish), and the auth handler's register/login flows (validation, duplicate-email conflict, wrong-password rejection).

## Known Limitations

This is a portfolio project, not a production system. Things deliberately left out, with the reasoning for each:

- No rate limiting on auth endpoints — would add `golang.org/x/time/rate` per-IP middleware
- No refresh tokens — JWTs simply expire after `JWT_EXPIRY_HOURS`; a refresh flow would be the next addition
- No database migrations tool (golang-migrate, goose) — migrations are plain SQL files applied manually or via `docker-entrypoint-initdb.d` on first container start
- No metrics endpoint — would add `/metrics` with `prometheus/client_golang` next
- `CheckOrigin` on the WebSocket upgrader currently allows all origins — production would restrict this to an explicit allowlist

## License

MIT
