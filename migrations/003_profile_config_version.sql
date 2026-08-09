-- Bumped when eis_config changes so workers/core know the search result set
-- must be re-synced (add new hits, remove ones that no longer match).
ALTER TABLE search_profiles
  ADD COLUMN IF NOT EXISTS config_version BIGINT NOT NULL DEFAULT 1;

COMMENT ON COLUMN search_profiles.config_version IS
  'Increments when eis_config changes; zakupki-core sync uses it to refresh tenders for search_profile_id';
