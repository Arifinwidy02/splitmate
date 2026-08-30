-- group_invite_links
-- Shared, multi-use invite links: one link can be used by many people to
-- join a group (unlike group_invitations, which are email-bound and
-- single-use). The raw token is stored so admins can re-display the link;
-- lookups use the hash.
CREATE TABLE group_invite_links (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    group_id   UUID NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    token      TEXT NOT NULL,
    token_hash TEXT NOT NULL UNIQUE,
    created_by UUID NOT NULL REFERENCES users(id),
    expires_at TIMESTAMPTZ NOT NULL,
    revoked_at TIMESTAMPTZ,
    max_uses   INT CHECK (max_uses IS NULL OR max_uses > 0),
    used_count INT NOT NULL DEFAULT 0 CHECK (used_count >= 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- At most one non-revoked link per group; creating a new link revokes the
-- previous one (done in the same transaction by the repository).
CREATE UNIQUE INDEX idx_group_invite_links_active
    ON group_invite_links(group_id)
    WHERE revoked_at IS NULL;

CREATE INDEX idx_group_invite_links_token_hash ON group_invite_links(token_hash);
