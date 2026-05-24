-- name: UpsertDailyReport :exec
INSERT INTO daily_reports (
    report_date,
    meal_count,
    total_sugar_grams,
    average_sugar_grams,
    highest_risk_level,
    summary,
    ai_insights
) VALUES (
    $1,
    $2,
    $3,
    $4,
    $5,
    $6,
    $7
)
ON CONFLICT (report_date) DO UPDATE SET
    meal_count = EXCLUDED.meal_count,
    total_sugar_grams = EXCLUDED.total_sugar_grams,
    average_sugar_grams = EXCLUDED.average_sugar_grams,
    highest_risk_level = EXCLUDED.highest_risk_level,
    summary = EXCLUDED.summary,
    ai_insights = EXCLUDED.ai_insights;

-- name: GetDailyReportByDate :one
SELECT report_date, meal_count, total_sugar_grams, average_sugar_grams, highest_risk_level, summary, ai_insights
FROM daily_reports
WHERE report_date = $1;
