# Budget Tracker

A personal finance tracker with real-time budget alerts.

## Features
- JWT authentication (register / login)
- Add transactions by category
- Set monthly budget limits per category
- Real-time WebSocket alerts when spending exceeds 80% or 100% of a budget
- Spending chart by category

## Tech Stack

**Backend:** Go 1.22 · Echo · pgx (raw SQL, no ORM) · Redis pub/sub · gorilla/websocket · JWT · slog  
**Frontend:** React 18 · TypeScript · Vite · Recharts · react-hot-toast  
**Infrastructure:** Docker multi-stage · docker-compose · Nginx (reverse proxy + WS)

## Architecture

```
┌─────────────┐    ┌──────────────────────────────────────┐    ┌──────────┐
│   Browser   │    │              Go Backend              │    │          │
│             │◄──►│  Echo Router                         │◄──►│ Postgres │
│  React/TS   │    │  ├── JWT Middleware                  │    │          │
│  Vite       │    │  ├── Handlers (auth/tx/budget/ws)    │    └──────────┘
│             │    │  ├── Services (business logic)       │
│  WebSocket  │◄──►│  ├── Alert Broker (pub/sub fan-out)  │◄──►│  Redis   │
│  client     │    │  └── Repositories (pgx queries)      │    │          │
└─────────────┘    └──────────────────────────────────────┘    └──────────┘
         │                                ▲
         └──── Nginx (port 80) ───────────┘
```

## Why these choices?
- **pgx over GORM** — query optimization is visible; no magic
- **Redis pub/sub over in-process channels** — decouples the alert producer from WS delivery; scales across instances
- **Echo** — clean middleware API, well-documented
- **SSE → WebSocket** — chosen WebSocket for bidirectional potential (ping/pong keepalive)

## Running locally

```bash
docker-compose up --build
# Visit http://localhost
```

## Running without Docker

```bash
# Start dependencies
docker run -d -p 5432:5432 -e POSTGRES_USER=postgres -e POSTGRES_PASSWORD=postgres -e POSTGRES_DB=budget_tracker postgres:16-alpine
docker run -d -p 6379:6379 redis:7-alpine

# Apply migrations
for f in backend/migrations/*.sql; do
  docker exec -i <postgres-container-id> psql -U postgres -d budget_tracker < "$f"
done

# Backend
cd backend && go run ./cmd/server/main.go

# Frontend
cd frontend && bun install && bun run dev
```
