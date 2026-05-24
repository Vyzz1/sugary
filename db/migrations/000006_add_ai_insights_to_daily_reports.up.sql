ALTER TABLE daily_reports
ADD COLUMN ai_insights JSONB NOT NULL DEFAULT '{}'::jsonb;

UPDATE daily_reports
SET ai_insights = jsonb_build_object(
    'summary', summary,
    'top_contributors', '[]'::jsonb,
    'recommendations', '[]'::jsonb,
    'pattern_signals', '[]'::jsonb
)
WHERE ai_insights = '{}'::jsonb;
