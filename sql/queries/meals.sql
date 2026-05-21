-- name: CreateMeal :one
INSERT INTO meals (
    dish_name,
    image_url,
    recorded_at,
    analysis_status,
    estimated_sugar_grams,
    estimated_calories,
    risk_level,
    analysis_notes
) VALUES (
    $1,
    $2,
    $3,
    $4,
    $5,
    $6,
    $7,
    $8
)
RETURNING id, dish_name, image_url, recorded_at, analysis_status, estimated_sugar_grams, estimated_calories, risk_level, analysis_notes;

-- name: ListMealsByDay :many
SELECT id, dish_name, image_url, recorded_at, analysis_status, estimated_sugar_grams, estimated_calories, risk_level, analysis_notes
FROM meals
WHERE recorded_at >= sqlc.arg(day_start)
  AND recorded_at < sqlc.arg(day_end)
ORDER BY recorded_at ASC;
