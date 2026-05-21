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
curl -X POST http://localhost:8080/meals \
  -H "Content-Type: application/json" \
  -d "{\"dish_name\":\"milk tea\",\"recorded_at\":\"2026-05-21T12:00:00Z\"}"

curl -X POST "http://localhost:8080/jobs/daily-report?date=2026-05-21"

curl "http://localhost:8080/reports/daily?date=2026-05-21"

curl -X POST "http://localhost:8080/meals/1/image" \
  -H "Authorization: Bearer <token>" \
  -F "image=@/path/to/photo.jpg"
```

## API Response Shape

Successful responses use:

```json
{
  "success": true,
  "data": {}
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
