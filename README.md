# sugary

Sugar checker MVP backend using Gin, `sqlc`, and clean architecture.

## MVP scope

- record meals by dish name, with optional image URL
- estimate sugar and calories with an AI analysis port
- compile an end-of-day report summarizing sugar intake and risk

## Quick start

```bash
go mod tidy
go run ./cmd/api
```

## Development

- `go test ./...` runs the full test suite
- `go fmt ./...` formats the codebase
- `go vet ./...` runs baseline static analysis
- `golangci-lint run ./...` runs lint checks
- `sqlc generate` regenerates typed query code after SQL changes
- `go run ./cmd/migrate up` applies pending database migrations
- `go run ./cmd/migrate down` rolls back the latest migration
- `go run ./cmd/migrate create add_new_table` creates a new up/down migration pair
Create a local `.env` from `.env.example` when you want to override defaults:

```bash
APP_ENV=development
LOG_LEVEL=info
APP_PORT=8080
PORT=10000
GEMINI_API_KEY=your-gemini-key
LOGIN_USER=admin
LOGIN_PASSWORD=change-me
JWT_SECRET=change-this-secret
JWT_EXPIRES_IN=24h
POSTGRES_HOST=localhost
POSTGRES_PORT=5432
POSTGRES_USER=postgres
POSTGRES_PASSWORD=postgres
POSTGRES_DB=sugary
REDIS_HOST=localhost
REDIS_PORT=6379
```

## Local infrastructure

Start Postgres and Redis with Docker Compose:

```bash
docker compose up -d
```

Stop them with:

```bash
docker compose down
```

## Example API flow

```bash
curl -X POST http://localhost:8080/login \
  -H "Content-Type: application/json" \
  -d "{\"username\":\"admin\",\"password\":\"change-me\"}"

# then use returned access_token
curl -X POST http://localhost:8080/meals \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <token>" \
  -d "{\"dish_name\":\"milk tea\",\"meal_type\":\"lunch\",\"recorded_at\":\"2026-05-21T12:00:00Z\"}"

curl -X POST "http://localhost:8080/jobs/daily-report?date=2026-05-21" \
  -H "Authorization: Bearer <token>"

curl "http://localhost:8080/reports/daily?date=2026-05-21" \
  -H "Authorization: Bearer <token>"
```

## API Response Shape

Successful responses use:

```json
{
  "success": true,
  "data": {}
}
```

## Auth Implementation

- `POST /login` is public and returns a JWT.
- Credentials are configured via `LOGIN_USER` and `LOGIN_PASSWORD`.
- JWT signing key and TTL are controlled by `JWT_SECRET` and `JWT_EXPIRES_IN`.
- All business routes require `Authorization: Bearer <token>`:
  - `POST /meals`
  - `POST /jobs/daily-report`
  - `GET /reports/daily`
- `GET /health` remains public.

Login success response example:

```json
{
  "success": true,
  "data": {
    "access_token": "<jwt>",
    "token_type": "Bearer",
    "expires_in": "24h"
  }
}
```

## Request Validation

`POST /login`:
- `username` is required and trimmed.
- `password` is required and trimmed.

`POST /meals`:
- `dish_name` is required and trimmed.
- `meal_type` is optional. Allowed values: `breakfast`, `lunch`, `dinner`, `snack`, `unspecified`.
- If `meal_type` is omitted, backend defaults it to `unspecified`.
- `recorded_at` must be RFC3339 if provided.
- `image_url` is optional; if provided it must be a non-empty `http/https` URL.
- backend calls Gemini AI during meal creation; `GEMINI_API_KEY` must be configured, otherwise `POST /meals` fails.

Validation errors return structured codes, for example:

```json
{
  "success": false,
  "error": {
    "code": "invalid_image_url",
    "message": "image_url must be a valid http/https URL"
  }
}
```

Errors use:

```json
{
  "success": false,
  "error": {
    "code": "invalid_token",
    "message": "invalid token"
  }
}
```

## Layout

- `cmd/api/`: application entrypoint
- `internal/domain/`: meals, nutrition analysis, daily report contracts
- `internal/usecase/`: meal logging and daily report compilation
- `internal/delivery/http/`: Gin routing and handlers
- `internal/repository/ai/`: AI analyzer adapters
- `internal/repository/memory/`: runnable in-memory adapters for MVP scaffolding
- `internal/repository/postgres/`: future persistence adapters
- `sql/`: schema and query definitions for `sqlc`
- `db/migrations/`: migration files

## Configuration

- `GEMINI_API_KEY`: API key for the Gemini integration
- `GEMINI_MODEL`: default model name, currently `gemini-2.5-flash`
- `POSTGRES_*`: connection settings for the primary database
- `REDIS_*`: connection settings for cache and job-related state
- `PORT`: runtime port used by platforms like Render
- `LOGIN_USER` / `LOGIN_PASSWORD`: shared login credentials for `/login`
- `JWT_SECRET`: signing secret for issued tokens
- `JWT_EXPIRES_IN`: token lifetime, for example `24h`
- `LOG_LEVEL`: `debug|info|warn|error` for structured JSON logs
