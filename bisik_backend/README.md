# Backend Go Bisik

Backend memakai Go 1.26, PostgreSQL 18, REST, dan WebSocket. Struktur kode
mengikuti clean architecture dengan dependency mengarah ke domain/use case.

## Lapisan

```text
cmd/api (composition root)
  └─ adapter/http + adapter/ws
       └─ usecase
            └─ domain

infrastructure/assemblyai ──implements──> usecase ports
adapter/repository/postgres ─implements──> domain repositories
```

| Lapisan | Lokasi | Tanggung jawab |
|---|---|---|
| Domain | `internal/domain` | Entity dan interface repository tanpa HTTP, SQL, atau AssemblyAI |
| Use case | `internal/usecase` | Aturan sesi, scoring, compliance, dan port layanan luar |
| Adapter masuk | `internal/adapter/http`, `internal/adapter/ws` | Menerjemahkan REST/WebSocket menjadi pemanggilan use case |
| Adapter keluar | `internal/adapter/repository/postgres` | Implementasi repository PostgreSQL |
| Infrastructure | `internal/infrastructure` | AssemblyAI, konfigurasi, koneksi DB, migrasi, dan logger |
| Composition root | `cmd/api/main.go` | Membuat implementasi konkret dan memasang seluruh dependency |

Domain tidak mengimpor framework atau package adapter. Use case bergantung pada
interface (`SpeechToText`, `SemanticMatcher`, `Guardrail`, `Nudger`,
`Broadcaster`), bukan pada AssemblyAI atau WebSocket konkret.

Catatan teknis: command kalibrasi speaker saat ini diproses adapter WebSocket
melalui port `SpeakerRoleCalibrator`. Ini masih menjaga arah dependency, tetapi
koordinasinya dapat dipindahkan ke use case khusus jika alur kalibrasi nanti
memerlukan persistence atau pemetaan ulang di tengah sesi.

## State management

Backend tidak memakai library state-management khusus. State dibagi berdasarkan
umur dan sumber kebenarannya:

| Jenis state | Penyimpanan | Pengaman |
|---|---|---|
| Sesi, transkrip, checklist, pelanggaran | PostgreSQL | Repository + transaksi/constraint DB |
| Koneksi AssemblyAI dan mapping speaker aktif | Memory dalam `StreamingSTT` | `sync.RWMutex` per struktur map |
| Klien WebSocket per room/sesi | Memory dalam `Hub` | `sync.RWMutex` + buffered channel |
| Lifecycle request/stream | `context.Context` | Cancellation dan goroutine |
| Aliran event realtime | Go channel | Acknowledgement sebelum finalisasi skor |

PostgreSQL adalah sumber kebenaran untuk laporan. State memory hanya berlaku
selama koneksi realtime hidup dan tidak dipakai sebagai penyimpanan laporan.

## Menjalankan lokal

Prasyarat: Go 1.26 dan Docker Desktop. PostgreSQL berjalan di Docker, sedangkan
backend Go berjalan langsung memakai Go host; tidak membutuhkan container Node.

```bash
cd bisik_backend
make db          # jalankan PostgreSQL 18
make migrate     # opsional; make dev juga menjalankan migrasi otomatis
make dev         # API dan WebSocket di http://localhost:8080
```

File `.env` lokal dimuat otomatis oleh Makefile. Jangan menimpanya karena berisi
API key lokal yang diabaikan Git.

Verifikasi:

```bash
curl -sS http://localhost:8080/health
```

Respons yang diharapkan:

```json
{"service":"bisik_backend","status":"ok"}
```

Endpoint utama:

| Method/path | Fungsi |
|---|---|
| `POST /api/sessions` | Membuat sesi dan lima checklist |
| `POST /api/sessions/{id}/end` | Flush STT lalu menghitung skor |
| `GET /api/sessions/{id}/report` | Mengambil laporan berbukti |
| `GET /api/obligations` | Daftar kewajiban |
| `GET /ws?session_id=...&role=officer` | Audio dan event realtime petugas |
| `GET /ws?session_id=...&role=supervisor` | Event realtime supervisor |

Pemeriksaan kode:

```bash
go test ./...
go vet ./...
```

Menghentikan PostgreSQL tanpa menghapus data:

```bash
make db-down
```

`make db-reset` menghapus volume database dan hanya digunakan saat benar-benar
ingin membuat database lokal dari nol.
