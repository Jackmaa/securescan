CREATE TABLE IF NOT EXISTS findings (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    scan_id        UUID NOT NULL REFERENCES scans(id) ON DELETE CASCADE,
    tool_name      TEXT NOT NULL,
    rule_id        TEXT,
    file_path      TEXT,
    line_start     INTEGER,
    line_end       INTEGER,
    col_start      INTEGER,
    col_end        INTEGER,
    message        TEXT NOT NULL,
    severity       TEXT NOT NULL CHECK (severity IN ('critical','high','medium','low','info')),
    owasp_category TEXT,
    owasp_label    TEXT,
    cwe_id         TEXT,
    raw_output     JSONB,
    code_snippet   TEXT,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_findings_scan ON findings(scan_id);
CREATE INDEX IF NOT EXISTS idx_findings_severity ON findings(severity);
CREATE INDEX IF NOT EXISTS idx_findings_owasp ON findings(owasp_category);
