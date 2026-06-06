-- Insight queries intentionally do not scope by user yet.
-- TODO: add user_id filter when the meals table supports users.

-- name: GetInsightPeriodStats :one
SELECT
    COALESCE(SUM(estimated_sugar_grams), 0)::double precision AS total_sugar_grams,
    COUNT(*)::bigint AS meal_count,
    COUNT(*) FILTER (WHERE risk_level = 'high')::bigint AS high_risk_meals
FROM meals
WHERE deleted_at IS NULL
  AND analysis_status = 'completed'
  AND estimated_sugar_grams IS NOT NULL
  AND recorded_at >= sqlc.arg(from_inclusive)
  AND recorded_at < sqlc.arg(to_exclusive);

-- name: GetInsightDailySugar :many
SELECT
    (recorded_at AT TIME ZONE sqlc.arg(timezone_name)::text)::date AS date,
    COALESCE(SUM(estimated_sugar_grams), 0)::double precision AS total_sugar_grams,
    COALESCE(SUM(estimated_calories), 0)::bigint AS total_calories,
    COALESCE(SUM(estimated_carbs_grams), 0)::double precision AS total_carbs_grams,
    COALESCE(SUM(estimated_protein_grams), 0)::double precision AS total_protein_grams,
    COUNT(*)::bigint AS meal_count,
    CASE
        WHEN MAX(CASE risk_level WHEN 'high' THEN 3 WHEN 'medium' THEN 2 WHEN 'low' THEN 1 ELSE 0 END) = 3 THEN 'high'
        WHEN MAX(CASE risk_level WHEN 'high' THEN 3 WHEN 'medium' THEN 2 WHEN 'low' THEN 1 ELSE 0 END) = 2 THEN 'medium'
        WHEN MAX(CASE risk_level WHEN 'high' THEN 3 WHEN 'medium' THEN 2 WHEN 'low' THEN 1 ELSE 0 END) = 1 THEN 'low'
        ELSE 'none'
    END AS risk_level
FROM meals
WHERE deleted_at IS NULL
  AND analysis_status = 'completed'
  AND estimated_sugar_grams IS NOT NULL
  AND recorded_at >= sqlc.arg(from_inclusive)
  AND recorded_at < sqlc.arg(to_exclusive)
GROUP BY (recorded_at AT TIME ZONE sqlc.arg(timezone_name)::text)::date
ORDER BY date;

-- name: GetInsightMealTypeBreakdown :many
SELECT
    COALESCE(NULLIF(TRIM(meal_type), ''), 'unknown')::text AS meal_type,
    COALESCE(SUM(estimated_sugar_grams), 0)::double precision AS total_sugar_grams,
    COUNT(*)::bigint AS meal_count,
    COALESCE(AVG(estimated_sugar_grams), 0)::double precision AS average_sugar_grams
FROM meals
WHERE deleted_at IS NULL
  AND analysis_status = 'completed'
  AND estimated_sugar_grams IS NOT NULL
  AND recorded_at >= sqlc.arg(from_inclusive)
  AND recorded_at < sqlc.arg(to_exclusive)
GROUP BY COALESCE(NULLIF(TRIM(meal_type), ''), 'unknown')
ORDER BY total_sugar_grams DESC;

-- name: GetInsightRiskDistribution :many
SELECT
    COALESCE(NULLIF(TRIM(risk_level), ''), 'unknown')::text AS risk_level,
    COUNT(*)::bigint AS count
FROM meals
WHERE deleted_at IS NULL
  AND analysis_status = 'completed'
  AND estimated_sugar_grams IS NOT NULL
  AND recorded_at >= sqlc.arg(from_inclusive)
  AND recorded_at < sqlc.arg(to_exclusive)
GROUP BY COALESCE(NULLIF(TRIM(risk_level), ''), 'unknown')
ORDER BY count DESC, risk_level ASC;

-- name: GetInsightWeeklySugar :many
SELECT
    date_trunc('week', recorded_at AT TIME ZONE sqlc.arg(timezone_name)::text)::date AS week_start,
    COALESCE(SUM(estimated_sugar_grams), 0)::double precision AS total_sugar_grams,
    COALESCE(SUM(estimated_sugar_grams) / NULLIF(COUNT(DISTINCT (recorded_at AT TIME ZONE sqlc.arg(timezone_name)::text)::date), 0), 0)::double precision AS average_per_day,
    COUNT(*)::bigint AS meal_count,
    COUNT(*) FILTER (WHERE risk_level = 'high')::bigint AS high_risk_meals
FROM meals
WHERE deleted_at IS NULL
  AND analysis_status = 'completed'
  AND estimated_sugar_grams IS NOT NULL
  AND recorded_at >= sqlc.arg(from_inclusive)
  AND recorded_at < sqlc.arg(to_exclusive)
GROUP BY date_trunc('week', recorded_at AT TIME ZONE sqlc.arg(timezone_name)::text)::date
ORDER BY week_start ASC;

-- name: GetInsightSugarVsCalories :many
SELECT
    id AS meal_id,
    dish_name,
    estimated_sugar_grams AS sugar_grams,
    estimated_calories AS calories,
    risk_level,
    recorded_at
FROM meals
WHERE deleted_at IS NULL
  AND analysis_status = 'completed'
  AND estimated_sugar_grams IS NOT NULL
  AND estimated_calories IS NOT NULL
  AND recorded_at >= sqlc.arg(from_inclusive)
  AND recorded_at < sqlc.arg(to_exclusive)
ORDER BY recorded_at DESC, id DESC
LIMIT 100;

-- name: GetInsightTopSugarMeals :many
SELECT
    id AS meal_id,
    dish_name,
    estimated_sugar_grams AS sugar_grams,
    estimated_calories AS calories,
    meal_type,
    risk_level,
    recorded_at
FROM meals
WHERE deleted_at IS NULL
  AND analysis_status = 'completed'
  AND estimated_sugar_grams IS NOT NULL
  AND recorded_at >= sqlc.arg(from_inclusive)
  AND recorded_at < sqlc.arg(to_exclusive)
ORDER BY estimated_sugar_grams DESC, recorded_at DESC, id DESC
LIMIT 5;

-- name: GetInsightTopSugarDishes :many
SELECT
    TRIM(dish_name)::text AS dish_name,
    COUNT(*)::bigint AS times_logged,
    COALESCE(SUM(estimated_sugar_grams), 0)::double precision AS total_sugar_grams,
    COALESCE(AVG(estimated_sugar_grams), 0)::double precision AS average_sugar_grams
FROM meals
WHERE deleted_at IS NULL
  AND analysis_status = 'completed'
  AND estimated_sugar_grams IS NOT NULL
  AND NULLIF(TRIM(dish_name), '') IS NOT NULL
  AND recorded_at >= sqlc.arg(from_inclusive)
  AND recorded_at < sqlc.arg(to_exclusive)
GROUP BY TRIM(dish_name)
ORDER BY total_sugar_grams DESC, times_logged DESC
LIMIT 5;
