# barberpos-backend

Backend skeleton in Go (clean-ish layering) for BarberPOS. Uses PostgreSQL schema discussed earlier, HTTP with chi, pgx pool, and slog logging.

## Structure
- cmd/server: entrypoint.
- internal/config: env config loader.
- internal/db: pgx pool wiring (implements Health).
- internal/handler: HTTP handlers (health now; extend for domains).
- internal/server: router + server boot/shutdown.
- internal/domain: domain models/enums mirroring PostgreSQL schema (ID-based FKs, soft delete, currency-aware amounts).
- internal/ports: small ports (HealthChecker) to keep handlers infra-agnostic.

## Setup (local)
1) Install Go >= 1.22 and PostgreSQL.
2) Copy `.env.example` to `.env` and adjust `DATABASE_URL` (e.g. `postgres://user:pass@localhost:5432/barberpos?sslmode=disable`).
3) Run `go mod tidy` (once Go is available) to fetch dependencies.
4) Start server:
```bash
go run ./cmd/server
```

Health check: `GET /health` → `{ "status": "ok" }` ("degraded" if DB ping fails).

## Docker
- Build image:
```bash
docker build -t barberpos-backend:latest .
```
- Run with docker compose (includes Postgres on local port 55432):
```bash
cp .env.example .env  # adjust if needed
docker compose up --build
```
  - Backend: http://localhost:8080
  - Postgres: `postgres://barberpos_user:barberpos_password@localhost:55432/barberpos?sslmode=disable`
  - Compose overrides `DATABASE_URL` to point app → `db` container by default.

## Notes
- Money fields use integer minor units + `currency_code` (default IDR). Align with the shared SQL schema.
- Soft delete modeled via `deleted_at`; repositories should filter accordingly.
- `tenant_id` is present in models for future multi-tenant; can stay NULL for single-tenant.
- Add repositories/services per feature (auth, products, transactions, etc.) reusing the domain types.
