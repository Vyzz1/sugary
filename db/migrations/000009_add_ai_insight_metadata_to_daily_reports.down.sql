ALTER TABLE daily_reports
DROP COLUMN IF EXISTS ai_insight_status,
DROP COLUMN IF EXISTS ai_insight_source;
