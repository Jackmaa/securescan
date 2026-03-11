CREATE TABLE IF NOT EXISTS scans (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id   UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    status       TEXT NOT NULL DEFAULT 'pending'
                 CHECK (status IN ('pending','cloning','detecting','scanning','mapping','scoring','completed','failed')),
    score        INTEGER,
    grade        CHAR(1),
    tool_count   INTEGER NOT NULL DEFAULT 0,
    tools_done   INTEGER NOT NULL DEFAULT 0,
    error_msg    TEXT,
    started_at   TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
