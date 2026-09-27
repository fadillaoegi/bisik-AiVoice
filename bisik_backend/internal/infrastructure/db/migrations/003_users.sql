-- Akun petugas dan supervisor.
--
-- Nasabah sengaja tidak punya baris di sini: dia tidak memakai aplikasi.
-- Yang dia butuhkan adalah persetujuan untuk ditranskrip, bukan akun.
CREATE TABLE IF NOT EXISTS users (
    id            TEXT PRIMARY KEY,
    username      TEXT        NOT NULL UNIQUE,
    name          TEXT        NOT NULL,
    role          TEXT        NOT NULL,
    password_hash TEXT        NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT users_role_valid CHECK (role IN ('officer', 'supervisor')),
    CONSTRAINT users_username_not_blank CHECK (length(trim(username)) > 0)
);

-- sessions.officer_id kini menunjuk ke users.id. Foreign key sengaja tidak
-- dipasang: sesi lama dibuat sebelum ada tabel ini dan officer_id-nya masih
-- berupa teks bebas seperti 'PTG-001'. Menambah FK sekarang akan menolak
-- migrasi pada database yang sudah berisi data.
CREATE INDEX IF NOT EXISTS idx_sessions_officer_started
    ON sessions (officer_id, started_at DESC);
