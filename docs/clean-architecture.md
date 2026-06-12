# Clean Architecture in Sugary

## Overview

Sugary follows **Clean Architecture** — a layered design where business logic is isolated from infrastructure concerns. Dependencies always point inward: outer layers depend on inner layers, never the reverse.

```
┌──────────────────────────────────────────────┐
│               Delivery Layer                 │  HTTP handlers, WebSocket, middleware
├──────────────────────────────────────────────┤
│              Use Case Layer                  │  Business workflows
├──────────────────────────────────────────────┤
│              Domain Layer                    │  Entities + interfaces
├──────────────────────────────────────────────┤
│            Infrastructure Layer              │  DB, AI, file upload, scheduler
└──────────────────────────────────────────────┘
```

---

## Layers

### 1. Domain (`internal/domain/`)

The innermost layer. Contains pure business entities and the interfaces that outer layers must implement. Has **no dependencies** on any other layer or third-party package.

| File | Contents |
|---|---|
| `meal.go` | `Meal`, `Nutrition` structs; `MealRepository`, `NutritionAnalyzer` interfaces |
| `report.go` | `DailyReport`, `DailyReportAIInsights` structs; `DailyReportRepository`, `DailyReportInterpreter` interfaces |

**Example — domain interface:**

```go
// internal/domain/meal.go

type NutritionAnalyzer interface {
    Analyze(ctx context.Context, input AnalyzeMealInput) (*Nutrition, error)
}

type MealRepository interface {
    Create(ctx context.Context, input CreateMealInput) (*Meal, error)
    UpdateAnalysis(ctx context.Context, input UpdateMealAnalysisInput) error
    // ...
}
```

The domain layer defines **what** must exist. It never cares **how** it is implemented.

---

### 2. Use Case (`internal/usecase/`)

Orchestrates domain entities to fulfill application workflows. Each file represents one discrete operation.

| Use Case | File | Description |
|---|---|---|
| Log meal | `log_meal.go` | Save meal, trigger async AI analysis, broadcast result |
| Edit meal | `edit_meal.go` | Update metadata, optionally re-analyze |
| Edit analysis | `edit_meal_analysis.go` | User-corrects nutrition values |
| Delete meal | `delete_meal.go` | Soft delete |
| List meals | `list_meals.go`, `list_meals_by_day.go` | Filtered queries |
| Compile report | `compile_daily_report.go` | Aggregate meals, generate AI insights |
| Retry failed analyses | `retry_failed_meal_analyses.go` | Background retry job |
| Get insight | `get_insight.go` | Retrieve stored AI insights |

Use cases depend only on **domain interfaces**, never on concrete implementations:

```go
// internal/usecase/log_meal.go

type LogMealUseCase struct {
    mealRepo domain.MealRepository       // interface
    analyzer domain.NutritionAnalyzer    // interface
    hub      Publisher                   // interface
}
```

This means the use case is fully testable with stubs — no database or AI service required.

---

### 3. Infrastructure / Repository (`internal/repository/`)

Concrete implementations of domain interfaces. This layer is the only one allowed to import external packages (database drivers, AI SDKs, HTTP clients).

```
internal/repository/
├── postgres/              ← PostgreSQL implementation of MealRepository, DailyReportRepository
│   ├── sqlc/              ← Auto-generated type-safe SQL code (sqlc)
│   ├── meal_repository.go
│   ├── daily_report_repository.go
│   └── insight_repository.go
├── ai/
│   ├── gemini_nutrition_analyzer.go    ← Gemini 2.5 Flash
│   ├── huggingface.go                  ← HuggingFace Qwen2.5-7B
│   ├── fallback.go                     ← HuggingFace → Gemini fallback chain
│   ├── meal_prompt.go
│   └── daily_report_prompt.go
└── uploadproxy/           ← Proxies file uploads to external service
```

**Fallback AI chain:**

```
NutritionAnalyzer request
        │
        ▼
  HuggingFace (primary)
        │ fails
        ▼
  Gemini 2.5 Flash (fallback)
```

---

### 4. Delivery (`internal/delivery/http/`)

The outermost layer. Translates HTTP requests into use case calls and use case results into HTTP responses. Has no business logic of its own.

```
internal/delivery/http/
├── router.go              ← Route registration
├── handler/
│   ├── meal_handler.go
│   ├── report_handler.go
│   ├── insight_handler.go
│   ├── auth_handler.go
│   ├── upload_handler.go
│   ├── ws_handler.go
│   └── job_handler.go
└── middleware/
    ├── jwt.go             ← Bearer token validation
    ├── cors.go
    └── request_logger.go
```

---

### 5. Platform (`internal/platform/`)

Cross-cutting infrastructure that does not belong to any single layer.

| Package | Purpose |
|---|---|
| `platform/scheduler/cron/` | Cron job runner (robfig/cron) |
| `platform/hub/` | WebSocket broadcast hub |
| `platform/logging/` | Structured logging (Uber Zap) |
| `platform/timeutil/` | Day boundary / timezone helpers |

---

## Dependency Flow

```
delivery ──► usecase ──► domain ◄── repository
                                ◄── platform
```

- `delivery` knows about `usecase` structs
- `usecase` knows about `domain` interfaces only
- `repository` implements `domain` interfaces
- `platform` is used by `delivery`, `usecase`, and `repository`

The domain layer has **zero imports** from the rest of the codebase.

---

## Key Patterns

### Interface Defined at Point of Use

Each handler file declares its own small interfaces for the use cases it depends on:

```go
// internal/delivery/http/handler/meal_handler.go

type logMealUseCase interface {
    Execute(ctx context.Context, input domain.LogMealInput) (domain.Meal, error)
}

type MealHandler struct {
    logMeal logMealUseCase  // handler only knows this interface
}
```

`usecase.LogMealUseCase` never declares that it implements `logMealUseCase` — Go's implicit interface satisfaction handles it automatically as long as the method signature matches.

**Why define the interface in the handler, not in the usecase package?**

- **Avoids import cycle** — if `usecase` exported the interface and `delivery` imported it, you'd have `delivery → usecase → domain` which is fine, but it also couples the two packages unnecessarily. With the interface living in `delivery`, the dependency is one-directional.
- **Easier testing** — to test a handler, you only need a tiny mock that satisfies the interface. No real database, no real AI service.
- **Minimal surface** — the interface only exposes what the handler actually uses. If `LogMealUseCase` gains new methods later, the handler is unaffected.

```go
// Testing: a mock that satisfies logMealUseCase
type mockLogMeal struct{}

func (m mockLogMeal) Execute(ctx context.Context, input domain.LogMealInput) (domain.Meal, error) {
    return domain.Meal{DishName: "test"}, nil
}

handler := NewMealHandler(mockLogMeal{}, ...)
```

This follows the Go proverb: **"Accept interfaces, return structs."**

---

### Repository Pattern

Database access is hidden behind a `domain.MealRepository` interface. The use case calls `mealRepo.Create(...)` without knowing whether the backing store is PostgreSQL, an in-memory map, or a test stub.

### Constructor-based Dependency Injection

All dependencies are injected at construction time. There is no global state or service locator.

```go
uc := usecase.NewLogMealUseCase(mealRepo, analyzer).WithPublisher(hub)
```

Wiring happens in `cmd/api/main.go`.

### Async Analysis with Background Goroutine

When a meal is logged, the HTTP handler returns immediately with `status = processing`. A goroutine runs the AI analysis in the background and broadcasts the result via WebSocket when done. This keeps API latency low regardless of AI response time.

```
POST /api/meals
    │
    ├─► Save meal (status=processing) → return 201 immediately
    │
    └─► goroutine: Analyze → Update DB → Broadcast via WebSocket
```

### Retry Job

A cron job runs every 15 minutes, picks up meals with `status = failed`, and re-attempts analysis — up to 5 retries with a 15-minute cooldown between attempts.

### Soft Deletes

Meals are never physically removed. A `deleted_at` timestamp is set instead, and all queries filter on `deleted_at IS NULL`.

---

## Entry Point

`cmd/api/main.go` is responsible for:

1. Loading config from `.env`
2. Connecting to PostgreSQL and Redis
3. Instantiating all repositories, use cases, and handlers
4. Registering cron jobs
5. Starting the HTTP server (Gin) and WebSocket hub

This is the only file that knows about every layer — the composition root.
