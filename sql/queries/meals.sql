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
RETURNING id, dish_name, meal_type, image_url, recorded_at, analysis_status, estimated_sugar_grams, estimated_carbs_grams, estimated_protein_grams, estimated_calories, risk_level, analysis_notes, is_user_edited, deleted_at;

-- name: ListMealsByDay :many
SELECT id, dish_name, meal_type, image_url, recorded_at, analysis_status, estimated_sugar_grams, estimated_carbs_grams, estimated_protein_grams, estimated_calories, risk_level, analysis_notes, is_user_edited, deleted_at
FROM meals
WHERE recorded_at >= sqlc.arg(day_start)
  AND recorded_at < sqlc.arg(day_end)
  AND deleted_at IS NULL
ORDER BY recorded_at ASC;

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
        deleted_at
    FROM meals
    WHERE deleted_at IS NULL
      AND ($1::text = '' OR dish_name ILIKE '%' || $1 || '%')
    ORDER BY lower(dish_name), COALESCE(image_url, ''), recorded_at DESC, id DESC
)
SELECT *
FROM distinct_meals
ORDER BY
    CASE WHEN sqlc.arg(sort_type)::text = 'created_asc' THEN recorded_at END ASC,
    CASE WHEN sqlc.arg(sort_type)::text = 'created_desc' THEN recorded_at END DESC,
    CASE WHEN sqlc.arg(sort_type)::text = 'name_asc' THEN lower(dish_name) END ASC,
    CASE WHEN sqlc.arg(sort_type)::text = 'name_desc' THEN lower(dish_name) END DESC,
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
      AND ($1::text = '' OR dish_name ILIKE '%' || $1 || '%')
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
RETURNING id, dish_name, meal_type, image_url, recorded_at, analysis_status, estimated_sugar_grams, estimated_carbs_grams, estimated_protein_grams, estimated_calories, risk_level, analysis_notes, is_user_edited, deleted_at;

-- name: GetMealByID :one
SELECT id, dish_name, meal_type, image_url, recorded_at, analysis_status, estimated_sugar_grams, estimated_carbs_grams, estimated_protein_grams, estimated_calories, risk_level, analysis_notes, is_user_edited, deleted_at
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
RETURNING id, dish_name, meal_type, image_url, recorded_at, analysis_status, estimated_sugar_grams, estimated_carbs_grams, estimated_protein_grams, estimated_calories, risk_level, analysis_notes, is_user_edited, deleted_at;

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
    is_user_edited = FALSE
WHERE id = $1
  AND deleted_at IS NULL
RETURNING id, dish_name, meal_type, image_url, recorded_at, analysis_status, estimated_sugar_grams, estimated_carbs_grams, estimated_protein_grams, estimated_calories, risk_level, analysis_notes, is_user_edited, deleted_at;

-- name: SoftDeleteMealByID :execrows
UPDATE meals
SET deleted_at = NOW()
WHERE id = $1
  AND deleted_at IS NULL;
