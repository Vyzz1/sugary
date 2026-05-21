# Repository Guidelines

## Project Structure & Module Organization
This project is a Go backend for a sugar checker MVP using `gin`, `sqlc`, and clean architecture. Keep transport, use case, domain, and persistence concerns separate. Prefer a layout such as:

- `cmd/api/` for the application entrypoint
- `internal/domain/` for meals, nutrition, reports, and repository interfaces
- `internal/usecase/` for meal logging, AI analysis orchestration, and daily summaries
- `internal/delivery/http/` for Gin handlers, middleware, and routing
- `internal/repository/` for database adapters and `sqlc` integration
- `sql/schema/`, `sql/queries/` for DDL and `sqlc` query files
- `db/migrations/` for migration scripts

Dependencies must point inward: `delivery` and `repository` depend on `usecase` and `domain`, not the reverse. The MVP centers on recording meals, estimating sugar, and producing end-of-day reports.

## Build, Test, and Development Commands
Use standard Go commands and add wrappers in `Makefile` if needed.

Recommended commands:
- `go run ./cmd/api` runs the API locally
- `go test ./...` runs the full test suite
- `go fmt ./...` formats the codebase
- `go vet ./...` runs baseline static analysis
- `sqlc generate` regenerates typed query code after SQL changes
- `docker compose up -d` starts local Postgres and Redis

Environment variables are loaded from `.env` for local development and can still be overridden by the shell environment. Keep Gemini, Postgres, and Redis settings in `.env.example` as the source of truth for required local config.

## Security & Configuration Tips
Keep secrets out of source control. Commit `.env.example`, not `.env`, and document new settings in both `.env.example` and `README.md`.

## Coding Style & Naming Conventions
Use `gofmt` formatting with tabs as Go expects; do not hand-format around it. Keep package names short, lowercase, and singular where possible. Exported identifiers use `PascalCase`; internal helpers use `camelCase`.

Keep interfaces in the consuming layer when possible. Do not let Gin request models, SQL rows, or database errors leak into domain entities or use cases.

## Testing Guidelines
Write table-driven tests with Go's `testing` package. Place tests next to the code they cover using `_test.go` files, such as `internal/usecase/log_meal_test.go`. Favor use case tests for nutrition/report logic and repository integration tests for `sqlc` queries.

Include at least:
- happy-path tests
- failure-path tests
- regression tests for fixed defects

## Commit & Pull Request Guidelines
Git history is not available yet, so use short imperative commit messages such as `Add daily report compiler` or `Stub nutrition analyzer`. Keep commits focused and avoid mixing refactors with behavior changes.

Pull requests should include:
- a brief summary of the change
- linked issue or task ID when applicable
- test evidence
- migration notes for schema changes

## Database & SQLC Notes
Treat `sql/schema/`, `sql/queries/`, and generated `sqlc` code as a single unit. When changing meal/report storage, update migrations first, regenerate with `sqlc generate`, and review the generated API before merging.

## Agent-Specific Instructions
Preserve clean architecture boundaries in every change. If a feature needs a shortcut that crosses layers, stop and refactor the dependency direction instead of coupling HTTP handlers directly to persistence.
