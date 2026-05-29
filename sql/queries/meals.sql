-- name: CreateMeal :one
INSERT INTO meals (
    dish_name,
    meal_type,
    image_url,
    recorded_at,
    analysis_status,
    estimated_sugar_grams,
    estimated_carbs_grams,
    estimated_protein_grams,
    estimated_calories,
    risk_level,
    analysis_notes,
    is_user_edited
) VALUES (
    $1,
    $2,
    $3,
    $4,
    $5,
    $6,
    $7,
    $8,
    $9,
    $10,
    $11,
    $12
)
RETURNING id, dish_name, meal_type, image_url, recorded_at, analysis_status, estimated_sugar_grams, estimated_carbs_grams, estimated_protein_grams, estimated_calories, risk_level, analysis_notes, is_user_edited, analysis_retry_count, last_analysis_attempt_at, deleted_at;

-- name: ListMealsByDay :many
SELECT id, dish_name, meal_type, image_url, recorded_at, analysis_status, estimated_sugar_grams, estimated_carbs_grams, estimated_protein_grams, estimated_calories, risk_level, analysis_notes, is_user_edited, analysis_retry_count, last_analysis_attempt_at, deleted_at
FROM meals
WHERE recorded_at >= sqlc.arg(day_start)
  AND recorded_at < sqlc.arg(day_end)
  AND deleted_at IS NULL
ORDER BY
    CASE WHEN sqlc.arg(sort_type)::text = 'created_asc' THEN recorded_at END ASC,
    CASE WHEN sqlc.arg(sort_type)::text = 'created_desc' THEN recorded_at END DESC,
    CASE WHEN sqlc.arg(sort_type)::text = 'name_asc' THEN unaccent(lower(dish_name)) END ASC,
    CASE WHEN sqlc.arg(sort_type)::text = 'name_desc' THEN unaccent(lower(dish_name)) END DESC,
    recorded_at DESC, 
    id DESC;

-- name: ListRecentDistinctMeals :many
WITH distinct_meals AS (
    SELECT DISTINCT ON (lower(dish_name), COALESCE(image_url, ''))
        id,
        dish_name,
        meal_type,
        image_url,
        recorded_at,
        analysis_status,
        estimated_sugar_grams,
        estimated_carbs_grams,
        estimated_protein_grams,
        estimated_calories,
        risk_level,
        analysis_notes,
        is_user_edited,
        analysis_retry_count,
        last_analysis_attempt_at,
        deleted_at
    FROM meals
    WHERE deleted_at IS NULL
      AND (sqlc.arg(query_text)::text = '' OR unaccent(lower(dish_name)) LIKE '%' || unaccent(lower(sqlc.arg(query_text)::text)) || '%')
    ORDER BY lower(dish_name), COALESCE(image_url, ''), recorded_at DESC, id DESC
)
SELECT *
FROM distinct_meals
ORDER BY
    CASE WHEN sqlc.arg(sort_type)::text = 'created_asc' THEN recorded_at END ASC,
    CASE WHEN sqlc.arg(sort_type)::text = 'created_desc' THEN recorded_at END DESC,
    CASE WHEN sqlc.arg(sort_type)::text = 'name_asc' THEN unaccent(lower(dish_name)) END ASC,
    CASE WHEN sqlc.arg(sort_type)::text = 'name_desc' THEN unaccent(lower(dish_name)) END DESC,
    recorded_at DESC,
    id DESC
LIMIT sqlc.arg(limit_count)
OFFSET sqlc.arg(offset_count);

-- name: CountRecentDistinctMeals :one
WITH distinct_meals AS (
    SELECT DISTINCT ON (lower(dish_name), COALESCE(image_url, ''))
        id
    FROM meals
    WHERE deleted_at IS NULL
      AND (sqlc.arg(query_text)::text = '' OR unaccent(lower(dish_name)) LIKE '%' || unaccent(lower(sqlc.arg(query_text)::text)) || '%')
    ORDER BY lower(dish_name), COALESCE(image_url, ''), recorded_at DESC, id DESC
)
SELECT COUNT(*) FROM distinct_meals;

-- name: UpdateMealAnalysisByID :one
UPDATE meals
SET
    estimated_sugar_grams = $2,
    estimated_carbs_grams = $3,
    estimated_protein_grams = $4,
    estimated_calories = $5,
    is_user_edited = TRUE
WHERE id = $1
  AND deleted_at IS NULL
RETURNING id, dish_name, meal_type, image_url, recorded_at, analysis_status, estimated_sugar_grams, estimated_carbs_grams, estimated_protein_grams, estimated_calories, risk_level, analysis_notes, is_user_edited, analysis_retry_count, last_analysis_attempt_at, deleted_at;

-- name: GetMealByID :one
SELECT id, dish_name, meal_type, image_url, recorded_at, analysis_status, estimated_sugar_grams, estimated_carbs_grams, estimated_protein_grams, estimated_calories, risk_level, analysis_notes, is_user_edited, analysis_retry_count, last_analysis_attempt_at, deleted_at
FROM meals
WHERE id = $1
  AND deleted_at IS NULL;

-- name: UpdateMealMetaByID :one
UPDATE meals
SET
    meal_type = $2,
    recorded_at = $3
WHERE id = $1
  AND deleted_at IS NULL
RETURNING id, dish_name, meal_type, image_url, recorded_at, analysis_status, estimated_sugar_grams, estimated_carbs_grams, estimated_protein_grams, estimated_calories, risk_level, analysis_notes, is_user_edited, analysis_retry_count, last_analysis_attempt_at, deleted_at;

-- name: UpdateMealForReanalysisByID :one
UPDATE meals
SET
    dish_name = $2,
    meal_type = $3,
    image_url = $4,
    recorded_at = $5,
    analysis_status = 'processing',
    is_user_edited = FALSE,
    analysis_retry_count = 0,
    last_analysis_attempt_at = NULL
WHERE id = $1
  AND deleted_at IS NULL
RETURNING id, dish_name, meal_type, image_url, recorded_at, analysis_status, estimated_sugar_grams, estimated_carbs_grams, estimated_protein_grams, estimated_calories, risk_level, analysis_notes, is_user_edited, analysis_retry_count, last_analysis_attempt_at, deleted_at;

-- name: UpdateMealWithAnalysisByID :one
UPDATE meals
SET
    dish_name = $2,
    meal_type = $3,
    image_url = $4,
    recorded_at = $5,
    estimated_sugar_grams = $6,
    estimated_carbs_grams = $7,
    estimated_protein_grams = $8,
    estimated_calories = $9,
    risk_level = $10,
    analysis_notes = $11,
    analysis_status = 'completed',
    is_user_edited = FALSE
WHERE id = $1
  AND deleted_at IS NULL
RETURNING id, dish_name, meal_type, image_url, recorded_at, analysis_status, estimated_sugar_grams, estimated_carbs_grams, estimated_protein_grams, estimated_calories, risk_level, analysis_notes, is_user_edited, analysis_retry_count, last_analysis_attempt_at, deleted_at;

-- name: SoftDeleteMealByID :execrows
UPDATE meals
SET deleted_at = NOW()
WHERE id = $1
  AND deleted_at IS NULL;

-- name: UpdateMealAnalysisResultByID :one
-- Called by the async goroutine after AI analysis succeeds.
-- Sets all nutrition fields and marks analysis_status = 'completed'.
UPDATE meals
SET
    estimated_sugar_grams  = $2,
    estimated_carbs_grams  = $3,
    estimated_protein_grams = $4,
    estimated_calories     = $5,
    risk_level             = $6,
    analysis_notes         = $7,
    analysis_status        = 'completed',
    is_user_edited         = FALSE
WHERE id = $1
  AND deleted_at IS NULL
RETURNING id, dish_name, meal_type, image_url, recorded_at, analysis_status, estimated_sugar_grams, estimated_carbs_grams, estimated_protein_grams, estimated_calories, risk_level, analysis_notes, is_user_edited, analysis_retry_count, last_analysis_attempt_at, deleted_at;

-- name: ListRetryableFailedMeals :many
SELECT id, dish_name, meal_type, image_url, recorded_at, analysis_status, estimated_sugar_grams, estimated_carbs_grams, estimated_protein_grams, estimated_calories, risk_level, analysis_notes, is_user_edited, analysis_retry_count, last_analysis_attempt_at, deleted_at
FROM meals
WHERE analysis_status = 'failed'
  AND deleted_at IS NULL
  AND analysis_retry_count < sqlc.arg(max_retry_count)
  AND (
    last_analysis_attempt_at IS NULL
    OR last_analysis_attempt_at <= sqlc.arg(before_time)
  )
ORDER BY last_analysis_attempt_at ASC NULLS FIRST, id ASC
LIMIT sqlc.arg(limit_count);

-- name: RetryFailedMealAnalysisByID :one
UPDATE meals
SET
    analysis_status = 'processing'
WHERE id = $1
  AND analysis_status = 'failed'
  AND deleted_at IS NULL
RETURNING id, dish_name, meal_type, image_url, recorded_at, analysis_status, estimated_sugar_grams, estimated_carbs_grams, estimated_protein_grams, estimated_calories, risk_level, analysis_notes, is_user_edited, analysis_retry_count, last_analysis_attempt_at, deleted_at;

-- name: UpdateMealAnalysisStatusByID :execrows
-- Called by the async goroutine when all retries are exhausted.
UPDATE meals
SET
    analysis_status = $2,
    analysis_retry_count = CASE
        WHEN $2 = 'failed' THEN analysis_retry_count + 1
        ELSE analysis_retry_count
    END,
    last_analysis_attempt_at = CASE
        WHEN $2 = 'failed' THEN NOW()
        ELSE last_analysis_attempt_at
    END
WHERE id = $1
  AND deleted_at IS NULL;
