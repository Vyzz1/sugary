ALTER TABLE daily_reports
ADD COLUMN ai_insight_source TEXT NOT NULL DEFAULT 'fallback',
ADD COLUMN ai_insight_status TEXT NOT NULL DEFAULT 'fallback';
