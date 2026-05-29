ALTER TABLE meals
DROP COLUMN IF EXISTS last_analysis_attempt_at,
DROP COLUMN IF EXISTS analysis_retry_count;
