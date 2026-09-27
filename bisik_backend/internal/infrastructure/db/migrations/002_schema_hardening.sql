-- Audit label mentah speaker dan constraint integritas data.
-- File migrasi yang sudah diterapkan tidak boleh diedit; buat nomor baru.

ALTER TABLE utterances
    ADD COLUMN IF NOT EXISTS source_speaker TEXT NOT NULL DEFAULT 'UNKNOWN';

CREATE INDEX IF NOT EXISTS idx_sessions_status_started
    ON sessions (status, started_at DESC);

CREATE INDEX IF NOT EXISTS idx_obligation_states_status
    ON obligation_states (session_id, status);

CREATE INDEX IF NOT EXISTS idx_violations_session_detected
    ON violations (session_id, detected_at);

DO $migration$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'sessions_status_valid') THEN
        ALTER TABLE sessions ADD CONSTRAINT sessions_status_valid
            CHECK (status IN ('active', 'ended', 'aborted'));
    END IF;

    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'sessions_score_valid') THEN
        ALTER TABLE sessions ADD CONSTRAINT sessions_score_valid
            CHECK (score BETWEEN 0 AND 100);
    END IF;

    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'sessions_time_valid') THEN
        ALTER TABLE sessions ADD CONSTRAINT sessions_time_valid
            CHECK (ended_at IS NULL OR ended_at >= started_at);
    END IF;

    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'utterances_speaker_valid') THEN
        ALTER TABLE utterances ADD CONSTRAINT utterances_speaker_valid
            CHECK (speaker IN ('unknown', 'officer', 'customer'));
    END IF;

    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'utterances_time_valid') THEN
        ALTER TABLE utterances ADD CONSTRAINT utterances_time_valid
            CHECK (start_ms >= 0 AND end_ms >= start_ms);
    END IF;

    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'obligation_status_valid') THEN
        ALTER TABLE obligation_states ADD CONSTRAINT obligation_status_valid
            CHECK (status IN ('pending', 'satisfied', 'violated'));
    END IF;

    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'obligation_confidence_valid') THEN
        ALTER TABLE obligation_states ADD CONSTRAINT obligation_confidence_valid
            CHECK (confidence BETWEEN 0 AND 1);
    END IF;

    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'violations_severity_valid') THEN
        ALTER TABLE violations ADD CONSTRAINT violations_severity_valid
            CHECK (severity IN ('low', 'medium', 'high'));
    END IF;
END
$migration$;
