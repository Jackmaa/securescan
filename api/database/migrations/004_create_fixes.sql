CREATE TABLE IF NOT EXISTS fixes (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    finding_id    UUID NOT NULL REFERENCES findings(id) ON DELETE CASCADE,
    scan_id       UUID NOT NULL REFERENCES scans(id) ON DELETE CASCADE,
    fix_type      TEXT NOT NULL CHECK (fix_type IN ('template','ai')),
    description   TEXT NOT NULL,
    explanation   TEXT,
    original_code TEXT,
    fixed_code    TEXT,
    file_path     TEXT NOT NULL,
    line_start    INTEGER,
    line_end      INTEGER,
    status        TEXT NOT NULL DEFAULT 'pending'
                  CHECK (status IN ('pending','accepted','rejected','applied')),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_fixes_scan ON fixes(scan_id);
