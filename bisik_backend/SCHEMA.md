# Skema Database Bisik

Backend memakai PostgreSQL 18. Database menyimpan hasil percakapan dan bukti
kepatuhan, tetapi **tidak menyimpan audio mentah**.

## Relasi

```text
sessions (1)
  ├──< utterances (N)
  ├──< obligation_states (5 per sesi)
  └──< violations (N)

schema_migrations
  └── riwayat migrasi internal; tidak berelasi dengan data aplikasi
```

`obligation_states.evidence_id` dan `violations.evidence_id` menunjuk secara
logis ke `utterances.id`. Foreign key sengaja belum dipasang agar revisi/finalisasi
stream tidak gagal ketika event datang tidak berurutan. Konsistensinya dijaga
use case dan ditampilkan kembali pada laporan.

## Tabel aplikasi

### `sessions`

Satu baris untuk satu percakapan petugas dengan nasabah.

| Kolom | Tipe | Keterangan |
|---|---|---|
| `id` | `TEXT` PK | UUID sesi dari backend |
| `officer_id` | `TEXT` | ID petugas dari sistem pemilik |
| `product_id` | `TEXT` | Produk yang sedang dijelaskan |
| `status` | `TEXT` | `active`, `ended`, atau `aborted` |
| `started_at` | `TIMESTAMPTZ` | Waktu mulai UTC |
| `ended_at` | `TIMESTAMPTZ` nullable | Waktu selesai UTC |
| `score` | `INT` | Skor final 0–100 |

### `utterances`

Transkrip final per giliran bicara. Turn kalibrasi tidak dimasukkan ke tabel ini.

| Kolom | Tipe | Keterangan |
|---|---|---|
| `id` | `TEXT` PK | `<session-id>-t<turn-order>` |
| `session_id` | `TEXT` FK | Sesi pemilik; cascade saat sesi dihapus |
| `speaker` | `TEXT` | Role hasil mapping: `officer`, `customer`, `unknown` |
| `source_speaker` | `TEXT` | Label mentah diarization, misalnya `A`/`B`; untuk audit |
| `text` | `TEXT` | Hasil transkripsi |
| `start_ms`, `end_ms` | `INT` | Rentang waktu relatif terhadap awal audio |
| `revised` | `BOOLEAN` | Pernah dikoreksi oleh speaker revision |
| `created_at` | `TIMESTAMPTZ` | Waktu penyimpanan |

### `obligation_states`

Lima checklist wajib untuk setiap sesi. Primary key gabungan
`(session_id, code)` memastikan satu state per butir.

| Kolom | Tipe | Keterangan |
|---|---|---|
| `session_id` | `TEXT` FK | Sesi pemilik |
| `code` | `TEXT` | `IDENTITY`, `RATE`, `TENOR`, `PENALTY`, `RIGHT` |
| `status` | `TEXT` | `pending`, `satisfied`, atau `violated` |
| `confidence` | `DOUBLE PRECISION` | Nilai semantic matcher 0–1 |
| `evidence_id` | `TEXT` nullable | ID ucapan petugas yang menjadi bukti |
| `satisfied_at` | `TIMESTAMPTZ` nullable | Waktu butir terpenuhi |

### `violations`

Janji/frasa terlarang yang ditemukan guardrail deterministik.

| Kolom | Tipe | Keterangan |
|---|---|---|
| `id` | `TEXT` PK | UUID pelanggaran |
| `session_id` | `TEXT` FK | Sesi pemilik |
| `phrase` | `TEXT` | Frasa yang cocok |
| `severity` | `TEXT` | `low`, `medium`, atau `high` |
| `evidence_id` | `TEXT` nullable | ID ucapan sumber pelanggaran |
| `detected_at` | `TIMESTAMPTZ` | Waktu deteksi |

## Riwayat migrasi

`schema_migrations` dibuat oleh migration runner dan berisi:

| Kolom | Fungsi |
|---|---|
| `version` | Nama file migrasi, menjadi primary key |
| `checksum` | SHA-256 isi file ketika diterapkan |
| `applied_at` | Waktu penerapan |

Migration runner:

- membaca berkas `internal/infrastructure/db/migrations/*.sql` sesuai nama;
- memakai PostgreSQL advisory lock agar dua instance tidak migrasi bersamaan;
- menjalankan setiap migrasi dalam transaksi terpisah;
- melewati versi yang sudah diterapkan;
- menolak startup jika isi migrasi lama berubah setelah diterapkan.

Migrasi saat ini:

1. `001_init.sql` — empat tabel aplikasi dan indeks dasar.
2. `002_schema_hardening.sql` — `source_speaker`, indeks query laporan, dan
   constraint status, skor, timestamp, speaker, confidence, serta severity.

## Menjalankan migrasi

Migrasi otomatis dijalankan saat API mulai. Untuk menjalankannya tanpa API:

```bash
cd bisik_backend
make db
make migrate
```

Melihat versi yang sudah diterapkan:

```sql
SELECT version, checksum, applied_at
FROM schema_migrations
ORDER BY version;
```

Untuk perubahan berikutnya, tambahkan file baru seperti
`003_nama_perubahan.sql`. Jangan mengubah file yang sudah diterapkan karena
checksum akan menolak perubahan tersebut. Strategi proyek ini adalah migrasi
maju (*forward-only*); koreksi dibuat sebagai nomor migrasi baru.
