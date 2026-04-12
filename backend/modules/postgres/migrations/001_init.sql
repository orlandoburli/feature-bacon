-- +goose Up

CREATE TABLE flags (
    tenant_id      TEXT NOT NULL,
    key            TEXT NOT NULL,
    type           TEXT NOT NULL,
    semantics      TEXT NOT NULL,
    enabled        BOOLEAN NOT NULL DEFAULT true,
    description    TEXT,
    rules          JSONB NOT NULL DEFAULT '[]',
    default_result JSONB NOT NULL DEFAULT '{}',
    created_by     TEXT,
    updated_by     TEXT,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (tenant_id, key)
);

CREATE TABLE assignments (
    tenant_id   TEXT NOT NULL,
    subject_id  TEXT NOT NULL,
    flag_key    TEXT NOT NULL,
    enabled     BOOLEAN NOT NULL,
    variant     TEXT NOT NULL DEFAULT '',
    assigned_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at  TIMESTAMPTZ,
    PRIMARY KEY (tenant_id, subject_id, flag_key)
);

CREATE TABLE experiments (
    tenant_id         TEXT NOT NULL,
    key               TEXT NOT NULL,
    name              TEXT NOT NULL,
    status            TEXT NOT NULL DEFAULT 'draft',
    sticky_assignment BOOLEAN NOT NULL DEFAULT false,
    variants          JSONB NOT NULL DEFAULT '[]',
    allocation        JSONB NOT NULL DEFAULT '[]',
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (tenant_id, key)
);

CREATE TABLE api_keys (
    id         TEXT PRIMARY KEY,
    tenant_id  TEXT NOT NULL,
    key_hash   TEXT NOT NULL UNIQUE,
    key_prefix TEXT NOT NULL,
    scope      TEXT NOT NULL,
    name       TEXT NOT NULL DEFAULT '',
    created_by TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    revoked_at TIMESTAMPTZ
);

CREATE INDEX idx_assignments_expires ON assignments(expires_at) WHERE expires_at IS NOT NULL;
CREATE INDEX idx_api_keys_tenant ON api_keys(tenant_id);

-- +goose Down

DROP INDEX IF EXISTS idx_api_keys_tenant;
DROP INDEX IF EXISTS idx_assignments_expires;
DROP TABLE IF EXISTS api_keys;
DROP TABLE IF EXISTS experiments;
DROP TABLE IF EXISTS assignments;
DROP TABLE IF EXISTS flags;
