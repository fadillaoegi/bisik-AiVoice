-- Bisik/Saksi — skema awal.
-- Migrasi bersifat idempotent dan dijalankan backend saat startup.

CREATE TABLE IF NOT EXISTS sessions (
    id          TEXT PRIMARY KEY,
    officer_id  TEXT        NOT NULL,
    product_id  TEXT        NOT NULL DEFAULT '',
    status      TEXT        NOT NULL DEFAULT 'active',
    started_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ended_at    TIMESTAMPTZ,
    score       INT         NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_sessions_officer ON sessions (officer_id, started_at DESC);

CREATE TABLE IF NOT EXISTS utterances (
    id         TEXT PRIMARY KEY,
    session_id TEXT        NOT NULL REFERENCES sessions (id) ON DELETE CASCADE,
    speaker    TEXT        NOT NULL DEFAULT 'unknown',
    text       TEXT        NOT NULL,
    start_ms   INT         NOT NULL DEFAULT 0,
    end_ms     INT         NOT NULL DEFAULT 0,
    revised    BOOLEAN     NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_utterances_session ON utterances (session_id, start_ms);

CREATE TABLE IF NOT EXISTS obligation_states (
    session_id   TEXT             NOT NULL REFERENCES sessions (id) ON DELETE CASCADE,
    code         TEXT             NOT NULL,
    status       TEXT             NOT NULL DEFAULT 'pending',
    confidence   DOUBLE PRECISION NOT NULL DEFAULT 0,
    evidence_id  TEXT,
    satisfied_at TIMESTAMPTZ,
    PRIMARY KEY (session_id, code)
);

CREATE TABLE IF NOT EXISTS violations (
    id          TEXT PRIMARY KEY,
    session_id  TEXT        NOT NULL REFERENCES sessions (id) ON DELETE CASCADE,
    phrase      TEXT        NOT NULL,
    severity    TEXT        NOT NULL DEFAULT 'high',
    evidence_id TEXT,
    detected_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_violations_session ON violations (session_id);
