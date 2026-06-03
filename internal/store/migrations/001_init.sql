-- 001_init.sql — SocialFlow MVP schema
-- Run with: psql $DATABASE_URL -f internal/store/migrations/001_init.sql

-- Extension for UUID generation
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- ============================================================================
-- Users
-- ============================================================================
CREATE TABLE users (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    email       TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    name        TEXT NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ============================================================================
-- Workspaces
-- ============================================================================
CREATE TABLE workspaces (
    id         UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name       TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ          -- soft delete
);

-- ============================================================================
-- Memberships (links users to workspaces with a role)
-- ============================================================================
CREATE TYPE role_enum AS ENUM ('admin', 'cm', 'viewer');

CREATE TABLE memberships (
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    user_id      UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role         role_enum NOT NULL DEFAULT 'viewer',
    joined_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (workspace_id, user_id)
);

-- ============================================================================
-- Workspace Invites
-- ============================================================================
CREATE TABLE workspace_invites (
    id           UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    created_by   UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token        TEXT NOT NULL UNIQUE,
    max_uses     INT NOT NULL DEFAULT 10,
    use_count    INT NOT NULL DEFAULT 0,
    expires_at   TIMESTAMPTZ NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ============================================================================
-- Clients
-- ============================================================================
CREATE TABLE clients (
    id             UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    workspace_id   UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    name           TEXT NOT NULL,
    social_handles JSONB NOT NULL DEFAULT '{}',
    notes          TEXT NOT NULL DEFAULT '',
    active         BOOLEAN NOT NULL DEFAULT true,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at     TIMESTAMPTZ          -- soft delete
);

-- Unique client name per workspace (case-insensitive), excluding soft-deleted
CREATE UNIQUE INDEX idx_clients_workspace_name
    ON clients (workspace_id, lower(name))
    WHERE deleted_at IS NULL;

CREATE INDEX idx_clients_workspace ON clients (workspace_id);

-- ============================================================================
-- Content Items
-- ============================================================================
CREATE TYPE content_status_enum AS ENUM (
    'draft',
    'review',
    'approved',
    'published',
    'archived'
);

CREATE TYPE content_platform_enum AS ENUM (
    'instagram',
    'facebook',
    'twitter',
    'linkedin',
    'tiktok',
    'youtube',
    'other'
);

CREATE TYPE content_type_enum AS ENUM (
    'post',
    'story',
    'reel',
    'video',
    'carousel',
    'other'
);

CREATE TABLE content_items (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    workspace_id    UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    client_id       UUID REFERENCES clients(id) ON DELETE SET NULL,
    title           TEXT NOT NULL,
    description     TEXT NOT NULL DEFAULT '',
    platform        content_platform_enum NOT NULL DEFAULT 'other',
    content_type    content_type_enum NOT NULL DEFAULT 'post',
    status          content_status_enum NOT NULL DEFAULT 'draft',
    scheduled_date  DATE,
    created_by      UUID NOT NULL REFERENCES users(id),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Lookups by workspace + status
CREATE INDEX idx_content_items_workspace_status
    ON content_items (workspace_id, status);

-- Calendar queries: items with a scheduled_date in a workspace
CREATE INDEX idx_content_items_workspace_scheduled
    ON content_items (workspace_id, scheduled_date)
    WHERE scheduled_date IS NOT NULL;

-- Filter by client within a workspace
CREATE INDEX idx_content_items_workspace_client
    ON content_items (workspace_id, client_id);

-- Recent items for dashboard
CREATE INDEX idx_content_items_workspace_updated
    ON content_items (workspace_id, updated_at DESC);

-- ============================================================================
-- Comments (on content items)
-- ============================================================================
CREATE TABLE comments (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    content_item_id UUID NOT NULL REFERENCES content_items(id) ON DELETE CASCADE,
    author_id       UUID NOT NULL REFERENCES users(id),
    body            TEXT NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_comments_content_item
    ON comments (content_item_id, created_at);

-- ============================================================================
-- Tasks
-- ============================================================================
CREATE TABLE tasks (
    id               UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    workspace_id     UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    title            TEXT NOT NULL,
    description      TEXT NOT NULL DEFAULT '',
    assignee_id      UUID REFERENCES users(id) ON DELETE SET NULL,
    due_date         DATE,
    done             BOOLEAN NOT NULL DEFAULT false,
    content_item_id  UUID REFERENCES content_items(id) ON DELETE SET NULL,
    client_id        UUID REFERENCES clients(id) ON DELETE SET NULL,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_tasks_workspace ON tasks (workspace_id);

-- Overdue tasks: due_date in past, not done
CREATE INDEX idx_tasks_workspace_due
    ON tasks (workspace_id, due_date)
    WHERE done = false;
