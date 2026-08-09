-- Fields required by gateway UI (/api/v1/searchers): per-searcher auto-AI and last run.
ALTER TABLE search_profiles
  ADD COLUMN IF NOT EXISTS auto_ai BOOLEAN NOT NULL DEFAULT false,
  ADD COLUMN IF NOT EXISTS last_run_at TIMESTAMPTZ;

COMMENT ON COLUMN search_profiles.auto_ai IS
  'When true, core/auto-AI should analyze tenders from this searcher after document processing';
COMMENT ON COLUMN search_profiles.last_run_at IS
  'Last POST /searchers/{id}/run timestamp';
