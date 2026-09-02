-- Sub2Video initial schema (PostgreSQL)
-- Inspired by Sub2API users/auth/payment patterns; creator-domain tables are new.

CREATE TABLE IF NOT EXISTS schema_migrations (
  version TEXT PRIMARY KEY,
  applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS users (
  id              BIGSERIAL PRIMARY KEY,
  public_id       VARCHAR(32) NOT NULL UNIQUE,
  email           VARCHAR(255) NOT NULL,
  password_hash   VARCHAR(255) NOT NULL,
  display_name    VARCHAR(64) NOT NULL DEFAULT '',
  role            VARCHAR(20) NOT NULL DEFAULT 'user',
  status          VARCHAR(20) NOT NULL DEFAULT 'active',
  token_version   INT NOT NULL DEFAULT 0,
  email_verified_at TIMESTAMPTZ NULL,
  created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  deleted_at      TIMESTAMPTZ NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS uk_users_email_active
  ON users (email) WHERE deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS workspaces (
  id              BIGSERIAL PRIMARY KEY,
  user_id         BIGINT NOT NULL UNIQUE REFERENCES users(id),
  name            VARCHAR(128) NOT NULL,
  created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS wallets (
  user_id         BIGINT PRIMARY KEY REFERENCES users(id),
  balance         NUMERIC(12,2) NOT NULL DEFAULT 0,
  version         INT NOT NULL DEFAULT 0,
  updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS wallet_ledger (
  id              BIGSERIAL PRIMARY KEY,
  public_id       VARCHAR(32) NOT NULL UNIQUE,
  user_id         BIGINT NOT NULL REFERENCES users(id),
  type            VARCHAR(32) NOT NULL,
  amount          NUMERIC(12,2) NOT NULL,
  balance_after   NUMERIC(12,2) NOT NULL,
  ref_type        VARCHAR(32) NULL,
  ref_id          VARCHAR(64) NULL,
  description     VARCHAR(512) NOT NULL DEFAULT '',
  created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_ledger_user_time ON wallet_ledger (user_id, created_at DESC);

CREATE TABLE IF NOT EXISTS refresh_tokens (
  id              BIGSERIAL PRIMARY KEY,
  user_id         BIGINT NOT NULL REFERENCES users(id),
  token_hash      CHAR(64) NOT NULL UNIQUE,
  expires_at      TIMESTAMPTZ NOT NULL,
  revoked_at      TIMESTAMPTZ NULL,
  created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_refresh_user ON refresh_tokens (user_id);

CREATE TABLE IF NOT EXISTS redeem_codes (
  id              BIGSERIAL PRIMARY KEY,
  code            VARCHAR(64) NOT NULL UNIQUE,
  amount          NUMERIC(12,2) NOT NULL,
  label           VARCHAR(128) NOT NULL DEFAULT '',
  max_uses        INT NULL,
  used_count      INT NOT NULL DEFAULT 0,
  expires_at      TIMESTAMPTZ NULL,
  created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS redeem_code_uses (
  id              BIGSERIAL PRIMARY KEY,
  code_id         BIGINT NOT NULL REFERENCES redeem_codes(id),
  user_id         BIGINT NOT NULL REFERENCES users(id),
  created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (code_id, user_id)
);

CREATE TABLE IF NOT EXISTS recharge_products (
  id              BIGSERIAL PRIMARY KEY,
  public_id       VARCHAR(16) NOT NULL UNIQUE,
  amount          NUMERIC(12,2) NOT NULL,
  bonus           NUMERIC(12,2) NOT NULL DEFAULT 0,
  price           NUMERIC(12,2) NOT NULL,
  enabled         BOOLEAN NOT NULL DEFAULT TRUE,
  sort_order      INT NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS generation_tasks (
  id              BIGSERIAL PRIMARY KEY,
  public_id       VARCHAR(32) NOT NULL UNIQUE,
  user_id         BIGINT NOT NULL REFERENCES users(id),
  title           VARCHAR(256) NOT NULL,
  workflow        VARCHAR(64) NOT NULL,
  status          VARCHAR(20) NOT NULL,
  progress        SMALLINT NOT NULL DEFAULT 0,
  duration_sec    SMALLINT NOT NULL,
  resolution      VARCHAR(16) NOT NULL,
  cost            NUMERIC(12,2) NOT NULL,
  thumb_url       TEXT NOT NULL DEFAULT '',
  project_public_id VARCHAR(32) NULL,
  saved_to_project BOOLEAN NOT NULL DEFAULT FALSE,
  completed_at    TIMESTAMPTZ NULL,
  created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_tasks_user_status ON generation_tasks (user_id, status, updated_at DESC);

CREATE TABLE IF NOT EXISTS projects (
  id              BIGSERIAL PRIMARY KEY,
  public_id       VARCHAR(32) NOT NULL UNIQUE,
  user_id         BIGINT NOT NULL REFERENCES users(id),
  name            VARCHAR(128) NOT NULL,
  description     VARCHAR(512) NOT NULL DEFAULT '',
  is_default      BOOLEAN NOT NULL DEFAULT FALSE,
  trashed_at      TIMESTAMPTZ NULL,
  folders_json    JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_projects_user ON projects (user_id, trashed_at);

CREATE TABLE IF NOT EXISTS scene_templates (
  id              BIGSERIAL PRIMARY KEY,
  public_id       VARCHAR(32) NOT NULL UNIQUE,
  title           VARCHAR(128) NOT NULL,
  category        VARCHAR(32) NOT NULL,
  description     TEXT NOT NULL,
  cover_color     VARCHAR(16) NOT NULL DEFAULT '',
  accent_color    VARCHAR(16) NOT NULL DEFAULT '',
  aspect_ratio    VARCHAR(8) NOT NULL,
  duration        VARCHAR(16) NOT NULL,
  resolution      VARCHAR(16) NOT NULL,
  poster          TEXT NOT NULL DEFAULT '',
  preview         TEXT NOT NULL DEFAULT '',
  enabled         BOOLEAN NOT NULL DEFAULT TRUE,
  sort_order      INT NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS tickets (
  id              BIGSERIAL PRIMARY KEY,
  public_id       VARCHAR(32) NOT NULL UNIQUE,
  user_id         BIGINT NOT NULL REFERENCES users(id),
  title           VARCHAR(256) NOT NULL,
  category        VARCHAR(32) NOT NULL,
  status          VARCHAR(20) NOT NULL DEFAULT 'open',
  task_ref        VARCHAR(64) NOT NULL DEFAULT '-',
  detail          TEXT NOT NULL DEFAULT '',
  created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_tickets_user ON tickets (user_id, updated_at DESC);

CREATE TABLE IF NOT EXISTS user_drafts (
  user_id         BIGINT PRIMARY KEY REFERENCES users(id),
  draft_prompt    TEXT NOT NULL DEFAULT '',
  current_project_public_id VARCHAR(32) NULL,
  updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO schema_migrations (version) VALUES ('001_init')
ON CONFLICT (version) DO NOTHING;
