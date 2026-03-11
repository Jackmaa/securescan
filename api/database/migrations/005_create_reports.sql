CREATE TABLE IF NOT EXISTS reports (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    scan_id    UUID NOT NULL REFERENCES scans(id) ON DELETE CASCADE,
    format     TEXT NOT NULL CHECK (format IN ('html','pdf')),
    file_path  TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
