CREATE TABLE meals (
    id BIGSERIAL PRIMARY KEY,
    dish_name TEXT NOT NULL,
    image_url TEXT,
    recorded_at TIMESTAMPTZ NOT NULL,
    analysis_status TEXT NOT NULL,
    estimated_sugar_grams DOUBLE PRECISION NOT NULL,
    estimated_calories INTEGER NOT NULL,
    risk_level TEXT NOT NULL,
    analysis_notes TEXT NOT NULL DEFAULT ''
);

CREATE TABLE daily_reports (
    report_date DATE PRIMARY KEY,
    meal_count INTEGER NOT NULL,
    total_sugar_grams DOUBLE PRECISION NOT NULL,
    average_sugar_grams DOUBLE PRECISION NOT NULL,
    highest_risk_level TEXT NOT NULL,
    summary TEXT NOT NULL
);
