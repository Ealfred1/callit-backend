CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'PENDING_VERIFICATION',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);

CREATE TABLE IF NOT EXISTS verification_codes (
    email VARCHAR(255) PRIMARY KEY,
    code VARCHAR(6) NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL
);

-- One row per community a creator sets up (fan page, campus group, etc).
-- attribution_id is sent to Panta as the buy-flow userId so volume rolls up
-- per community (PRD "Attribution").
CREATE TABLE IF NOT EXISTS communities (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(64) NOT NULL UNIQUE,
    attribution_id VARCHAR(64) NOT NULL UNIQUE,
    creator_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_communities_creator ON communities(creator_user_id);

-- Bridges an AI-compiled draft through Panta's quote -> build -> register
-- pipeline. Looked up by create_id when /markets/register/ comes back, so we
-- know which community/creator to attach the finished market to.
CREATE TABLE IF NOT EXISTS create_sessions (
    create_id VARCHAR(128) PRIMARY KEY,
    creator_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    community_id UUID REFERENCES communities(id) ON DELETE SET NULL,
    draft JSONB NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'quoted' CHECK (status IN ('quoted', 'built', 'registered')),
    market_id VARCHAR(128),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_create_sessions_creator ON create_sessions(creator_user_id);
CREATE INDEX IF NOT EXISTS idx_create_sessions_community ON create_sessions(community_id);

-- One row per live market that came through Callit, so community pages /
-- leaderboards / share cards don't have to re-derive attribution from Panta.
-- Panta stays the source of truth for prices/volume (PRD "Key decisions").
CREATE TABLE IF NOT EXISTS market_meta (
    market_id VARCHAR(128) PRIMARY KEY,
    create_id VARCHAR(128) REFERENCES create_sessions(create_id) ON DELETE SET NULL,
    creator_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    community_id UUID REFERENCES communities(id) ON DELETE SET NULL,
    category VARCHAR(32),
    title TEXT,
    share_image_path TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_market_meta_community ON market_meta(community_id);
CREATE INDEX IF NOT EXISTS idx_market_meta_creator ON market_meta(creator_user_id);
