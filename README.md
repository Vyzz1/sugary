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
CORS_ALLOW_ORIGINS=*
GEMINI_API_KEY=your-gemini-key
UPLOAD_API_URL=https://your-upload-api.example.com/upload
UPLOAD_INTERNAL_TOKEN=your-internal-upload-token
UPLOAD_FOLDER=sugary
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
CRON_ENABLED=false
CRON_DAILY_REPORT_EXPRESSION="5 0 * * *"
CRON_TIMEZONE=Asia/Ho_Chi_Minh
CRON_MEAL_ANALYSIS_RETRY_EXPRESSION="*/15 * * * *"
CRON_MEAL_ANALYSIS_RETRY_MAX_ATTEMPTS=5
CRON_MEAL_ANALYSIS_RETRY_COOLDOWN_MINUTES=15
CRON_MEAL_ANALYSIS_RETRY_BATCH_SIZE=25
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

curl -X POST http://localhost:8080/meals \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <token>" \
  -d "{\"source_meal_id\":1,\"meal_type\":\"dinner\",\"recorded_at\":\"2026-05-22T19:00:00Z\"}"

curl -X POST http://localhost:8080/api/upload \
  -H "Authorization: Bearer <token>" \
  -F "file=@./meal.jpg" \
  -F "file_name=meal-cover"

curl "http://localhost:8080/api/meals?date=2026-05-22" \
  -H "Authorization: Bearer <token>"

curl "http://localhost:8080/api/meals?q=tea&start_date=2026-05-20&end_date=2026-05-25&meal_type=snack&page=1&page_size=20&sort_by=estimated_sugar_grams&sort_type=desc" \
  -H "Authorization: Bearer <token>"

curl -X POST "http://localhost:8080/jobs/daily-report?date=2026-05-21" \
  -H "Authorization: Bearer <token>"

curl "http://localhost:8080/reports/daily?date=2026-05-21" \
  -H "Authorization: Bearer <token>"

curl "http://localhost:8080/api/meals/recent?q=tea&sort=created_desc&page=1&page_size=20" \
  -H "Authorization: Bearer <token>"

curl -X PATCH http://localhost:8080/api/meals/1 \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <token>" \
  -d "{\"meal_type\":\"lunch\",\"recorded_at\":\"2026-05-21T12:00:00Z\"}"

curl -X PATCH http://localhost:8080/api/meals/1/analysis \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <token>" \
  -d "{\"estimated_sugar_grams\":24,\"estimated_carbs_grams\":48,\"estimated_protein_grams\":12,\"estimated_calories\":460}"

curl -X DELETE http://localhost:8080/api/meals/1 \
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
- created meals include `is_user_edited=false` by default.
- alternatively send `source_meal_id` to clone an existing meal into a new independent meal without calling Gemini.
- when `source_meal_id` is provided, `dish_name` and `image_url` must be omitted; `meal_type` and `recorded_at` may override the cloned values.

`GET /api/meals`:
- returns filtered meals.
- optional `date` query in `YYYY-MM-DD` keeps the old exact-day behavior.
- optional `start_date` and `end_date` queries in `YYYY-MM-DD` filter by local recorded date range.
- optional `q` or `query` searches by `dish_name`.
- optional `meal_type` filters by `breakfast`, `lunch`, `dinner`, `snack`, `drink`, `unspecified`.
- pagination uses `page` and `page_size`; defaults to `1` and `20`, and `page_size` is capped at `100`.
- sorting uses `sort_by` (`recorded_at`, `dish_name`, `meal_type`, `estimated_sugar_grams`, `estimated_calories`) and `sort_type` (`asc`, `desc`).
- `sortBy` and `sortType` are also accepted for frontend compatibility.
- optional `X-Timezone` header controls how `date`, `start_date`, and `end_date` are interpreted; defaults to `Asia/Ho_Chi_Minh`.

`POST /api/upload`:
- accepts `multipart/form-data`
- required field: `file`
- optional field: `file_name`
- forwards to the configured upload API with `folder` from env and `x-internal-upload-token` from env

`PATCH /api/meals/:id`:
- if only `meal_type` or `recorded_at` changes: update directly, no AI re-analysis.
- if `dish_name` or `image_url` changes: backend updates the meal immediately with `analysis_status=processing`, then re-runs AI analysis asynchronously and pushes the result over WebSocket.

`GET /api/meals/recent`:
- returns recent distinct meals for the "choose again" UI.
- uses `SELECT DISTINCT ON (lower(dish_name), COALESCE(image_url, ''))` and keeps the newest meal per dish/image pair.
- optional `q` filters by `dish_name`.
- optional `sort`: `created_desc` (default), `created_asc`, `name_asc`, `name_desc`.
- pagination uses `page` and `page_size`; defaults to `1` and `20`, and `page_size` is capped at `100`.

`PATCH /api/meals/:id/analysis`:
- only edits estimated fields: `estimated_sugar_grams`, `estimated_carbs_grams`, `estimated_protein_grams`, `estimated_calories`.
- does not allow editing `risk_level` and `notes`.
- marks `is_user_edited=true`.

`DELETE /api/meals/:id`:
- soft delete via `deleted_at` (record is hidden from app reads and daily report compilation).

`GET /api/reports/daily` and `POST /api/jobs/daily-report`:
- accept optional `X-Timezone` header to interpret `date` and `today` using a client timezone.
- if the header is missing, backend falls back to `Asia/Ho_Chi_Minh`.
- manual trigger remains available even if the built-in cron scheduler is enabled.

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
- `UPLOAD_API_URL`: upstream upload endpoint used by the upload proxy
- `UPLOAD_INTERNAL_TOKEN`: value sent as `x-internal-upload-token` to the upstream upload API
- `UPLOAD_FOLDER`: forwarded form field for the upstream upload API, defaults to `sugary`
- `CORS_ALLOW_ORIGINS`: comma-separated allowed origins for browser requests, defaults to `*`
- `POSTGRES_*`: connection settings for the primary database
- `REDIS_*`: connection settings for cache and job-related state
- `CRON_ENABLED`: enables the built-in in-process scheduler for daily report generation
- `CRON_DAILY_REPORT_EXPRESSION`: cron expression for the daily report job, default `5 0 * * *`
- `CRON_TIMEZONE`: timezone used by the built-in scheduler, default `Asia/Ho_Chi_Minh`
- `CRON_MEAL_ANALYSIS_RETRY_EXPRESSION`: cron expression for retrying failed meal analyses, default every 15 minutes
- `CRON_MEAL_ANALYSIS_RETRY_MAX_ATTEMPTS`: max failed analysis waves before the retry cron stops picking a meal
- `CRON_MEAL_ANALYSIS_RETRY_COOLDOWN_MINUTES`: minimum wait time after a failed analysis before cron retries it
- `CRON_MEAL_ANALYSIS_RETRY_BATCH_SIZE`: max failed meals retried in one cron run
- `PORT`: runtime port used by platforms like Render
- `LOGIN_USER` / `LOGIN_PASSWORD`: shared login credentials for `/login`
- `JWT_SECRET`: signing secret for issued tokens
- `JWT_EXPIRES_IN`: token lifetime, for example `24h`
- `LOG_LEVEL`: `debug|info|warn|error` for structured JSON logs

## Built-In Cron

When `CRON_ENABLED=true`, the API process starts an internal scheduler backed by `robfig/cron/v3`.

- the scheduled job reuses the same `CompileDailyReport` use case as `POST /api/jobs/daily-report`
- the default schedule is `00:05` every day in `Asia/Ho_Chi_Minh`
- the job always compiles the report for the previous local day
- manual triggering via `POST /api/jobs/daily-report` is still available for re-runs and backfills
- a second scheduled job retries failed meal analyses using the same async AI runner path
- failed analyses remain eligible while `analysis_retry_count < CRON_MEAL_ANALYSIS_RETRY_MAX_ATTEMPTS`
