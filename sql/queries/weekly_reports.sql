-- name: UpsertWeeklyReport :exec
INSERT INTO weekly_reports (
    week_start_date,
    week_end_date,
    created_at,
    meal_count,
    analyzed_meal_count,
    total_sugar_grams,
    average_sugar_grams,
    highest_risk_level,
    summary,
    daily_breakdown,
    ai_insights,
    ai_insight_source,
    ai_insight_status
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
    $12,
    $13
)
ON CONFLICT (week_start_date) DO UPDATE SET
    week_end_date = EXCLUDED.week_end_date,
    meal_count = EXCLUDED.meal_count,
    analyzed_meal_count = EXCLUDED.analyzed_meal_count,
    total_sugar_grams = EXCLUDED.total_sugar_grams,
    average_sugar_grams = EXCLUDED.average_sugar_grams,
    highest_risk_level = EXCLUDED.highest_risk_level,
    summary = EXCLUDED.summary,
    daily_breakdown = EXCLUDED.daily_breakdown,
    ai_insights = EXCLUDED.ai_insights,
    ai_insight_source = EXCLUDED.ai_insight_source,
    ai_insight_status = EXCLUDED.ai_insight_status;

-- name: GetWeeklyReportByWeekStart :one
SELECT week_start_date, week_end_date, created_at, meal_count, analyzed_meal_count, total_sugar_grams, average_sugar_grams, highest_risk_level, summary, daily_breakdown, ai_insights, ai_insight_source, ai_insight_status
FROM weekly_reports
WHERE week_start_date = $1;
