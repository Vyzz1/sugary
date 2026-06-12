CREATE TABLE weekly_reports (
    week_start_date DATE PRIMARY KEY,
    week_end_date DATE NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    meal_count INTEGER NOT NULL,
    analyzed_meal_count INTEGER NOT NULL,
    total_sugar_grams DOUBLE PRECISION NOT NULL,
    average_sugar_grams DOUBLE PRECISION NOT NULL,
    highest_risk_level TEXT NOT NULL,
    summary TEXT NOT NULL,
    daily_breakdown JSONB NOT NULL DEFAULT '[]'::jsonb,
    ai_insights JSONB NOT NULL DEFAULT '{}'::jsonb,
    ai_insight_source TEXT NOT NULL DEFAULT 'fallback',
    ai_insight_status TEXT NOT NULL DEFAULT 'fallback'
);
