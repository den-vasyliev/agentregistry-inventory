-- Initial schema for Enterprise MCP Registry
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS pg_trgm;

CREATE TABLE IF NOT EXISTS servers (
  id                UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  server_name       TEXT NOT NULL,
  version           TEXT NOT NULL,

  server_json       JSONB NOT NULL,
  meta_official     JSONB,
  meta_enterprise   JSONB,

  owner             TEXT,
  repo_owner        TEXT,
  repo_name         TEXT,
  stars             INTEGER,
  downloads_total   INTEGER,
  downloads_weekly  INTEGER,
  source_registry   TEXT,
  sources           TEXT[] DEFAULT '{}',

  content_checksum  TEXT NOT NULL,
  first_seen_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
  last_seen_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
  last_changed_at   TIMESTAMPTZ,

  UNIQUE (server_name, version)
);

CREATE INDEX IF NOT EXISTS idx_servers_name ON servers (server_name);
CREATE INDEX IF NOT EXISTS idx_servers_repo ON servers (repo_owner, repo_name);
CREATE INDEX IF NOT EXISTS idx_servers_sources ON servers USING GIN (sources);
CREATE INDEX IF NOT EXISTS idx_servers_serverjson_gin ON servers USING GIN (server_json jsonb_path_ops);
CREATE INDEX IF NOT EXISTS idx_servers_text_trgm ON servers USING GIN ((server_json->>'name') gin_trgm_ops);
CREATE INDEX IF NOT EXISTS idx_servers_desc_trgm ON servers USING GIN ((server_json->>'description') gin_trgm_ops);


