-- Lightweight catalog of found procurements (hits from search / handoff to parser).
CREATE TABLE tenders (
  id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id          UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  profile_id       UUID REFERENCES search_profiles(id) ON DELETE SET NULL,
  reg_number       TEXT NOT NULL,
  law              TEXT NOT NULL DEFAULT '',
  notice_url       TEXT NOT NULL DEFAULT '',
  notice_guid      TEXT NOT NULL DEFAULT '',
  source_site      TEXT NOT NULL DEFAULT 'https://zakupki.gov.ru',
  object_title     TEXT NOT NULL DEFAULT '',
  status           TEXT NOT NULL DEFAULT '',
  price_raw        TEXT NOT NULL DEFAULT '',
  org_name         TEXT NOT NULL DEFAULT '',
  published_at     TEXT NOT NULL DEFAULT '',
  updated_on_site  TEXT NOT NULL DEFAULT '',
  application_end  TEXT NOT NULL DEFAULT '',
  payload          JSONB NOT NULL DEFAULT '{}'::jsonb,
  found_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
  created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (user_id, source_site, reg_number)
);

CREATE INDEX tenders_user_id_idx ON tenders(user_id);
CREATE INDEX tenders_profile_id_idx ON tenders(profile_id);
CREATE INDEX tenders_reg_number_idx ON tenders(reg_number);
CREATE INDEX tenders_law_idx ON tenders(law);
CREATE INDEX tenders_found_at_idx ON tenders(found_at DESC);
