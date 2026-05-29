ALTER TABLE meals
ADD COLUMN analysis_retry_count INTEGER NOT NULL DEFAULT 0,
ADD COLUMN last_analysis_attempt_at TIMESTAMPTZ;
