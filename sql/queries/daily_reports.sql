-- name: UpsertDailyReport :exec
INSERT INTO daily_reports (
    report_date,
    meal_count,
    total_sugar_grams,
    average_sugar_grams,
    highest_risk_level,
    summary
) VALUES (
    $1,
    $2,
    $3,
    $4,
    $5,
    $6
)
ON CONFLICT (report_date) DO UPDATE SET
    meal_count = EXCLUDED.meal_count,
    total_sugar_grams = EXCLUDED.total_sugar_grams,
    average_sugar_grams = EXCLUDED.average_sugar_grams,
    highest_risk_level = EXCLUDED.highest_risk_level,
    summary = EXCLUDED.summary;

-- name: GetDailyReportByDate :one
SELECT report_date, meal_count, total_sugar_grams, average_sugar_grams, highest_risk_level, summary
FROM daily_reports
WHERE report_date = $1;
