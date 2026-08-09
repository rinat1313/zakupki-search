-- zakupki-search: users + per-user EIS search profiles
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE users (
  id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  login           TEXT NOT NULL UNIQUE,
  password_hash   TEXT NOT NULL,
  display_name    TEXT NOT NULL DEFAULT '',
  created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE sessions (
  id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id      UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  token_hash   TEXT NOT NULL UNIQUE,
  expires_at   TIMESTAMPTZ NOT NULL,
  created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX sessions_user_id_idx ON sessions(user_id);
CREATE INDEX sessions_expires_at_idx ON sessions(expires_at);

-- Named search setting owned by a user. eis_config holds EIS query parameters.
-- id is the stable cross-service key: zakupki-core stores it as search_profile_id.
CREATE TABLE search_profiles (
  id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id      UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  name         TEXT NOT NULL,
  description  TEXT NOT NULL DEFAULT '',
  source       TEXT NOT NULL DEFAULT 'eis',
  eis_config   JSONB NOT NULL DEFAULT '{}'::jsonb,
  enabled      BOOLEAN NOT NULL DEFAULT true,
  created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (user_id, name)
);

CREATE INDEX search_profiles_user_id_idx ON search_profiles(user_id);
CREATE INDEX search_profiles_eis_config_gin ON search_profiles USING GIN (eis_config);

-- Demo user: login=demo / password=demo
-- bcrypt hash for "demo" (cost 10)
INSERT INTO users (login, password_hash, display_name)
VALUES (
  'demo',
  '$2a$10$7DVFvBs46wuWHeLQG9WIie5l5bUGD/H2TiUrU5eoK2cMW2cqFzuqW',
  'Demo User'
);

INSERT INTO search_profiles (user_id, name, description, source, eis_config, enabled)
SELECT
  u.id,
  'ПО и разработка',
  'Демо-настройка: мониторинг по ключевым словам, 44/223, подача заявок',
  'eis',
  '{
    "search_string": "Разработка ПО",
    "morphology": true,
    "strict_equal": false,
    "fz44": true,
    "fz223": true,
    "pp_rf_615": false,
    "stage_af": true,
    "stage_ca": true,
    "stage_pc": false,
    "stage_pa": false,
    "sort_by": "UPDATE_DATE",
    "sort_direction": false,
    "records_per_page": "_10"
  }'::jsonb,
  true
FROM users u
WHERE u.login = 'demo';
