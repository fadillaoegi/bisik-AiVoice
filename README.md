# Bisik

> Kopilot kepatuhan real-time untuk petugas keuangan lapangan Indonesia. Bisik mendengarkan percakapan tatap muka **petugas ↔ nasabah**, mengenali siapa yang berbicara, lalu membisikkan kewajiban yang belum disampaikan — selagi percakapan masih berlangsung.

Dibangun untuk **AssemblyAI Voice Agent Hackathon 2026** (lablab.ai).

**Bisik** adalah nama produk sekaligus nama ketiga component/module. Nama folder
repository dan remote GitHub masih memakai `saksi-voiceAI` agar riwayat dan URL
publik tidak terputus.

## Kenapa ini berbeda

Hampir semua voice agent adalah **manusia ↔ bot**. Bisik adalah **AI yang mendengarkan dua manusia** dan hanya ikut campur saat ada kewajiban yang terlewat.

Bedanya bukan sekadar teknis. Audit kepatuhan yang ada hari ini berjalan *setelah* kerugian terjadi — rekaman diputar ulang berminggu-minggu kemudian, saat nasabah sudah terlanjur menandatangani sesuatu yang tidak dia pahami. Bisik memindahkan koreksinya ke detik ketika masih ada gunanya: petugas mendengar pengingat di earpiece-nya sendiri, nasabah tidak mendengar apa pun, dan percakapan berjalan terus tanpa canggung.

Pelanggaran tetap tercatat untuk supervisor. Mencegah bukan berarti menghapus jejak.

## Coba sendiri

**Demo terarah · 11 detik.** Buka `/officer`, klik **Putar demo terarah**. Berjalan sepenuhnya di browser — tanpa mikrofon, tanpa backend, tanpa API eksternal.

Alurnya memperlihatkan seluruh mekanisme: petugas mengucapkan janji terlarang → bisikan koreksi masuk privat → kewajiban terpenuhi satu per satu → **diarization salah melabeli kalimat petugas sebagai nasabah, checklist sengaja tetap diam** → AssemblyAI mengoreksi labelnya, ucapan itu dinilai ulang, dua kewajiban terakhir baru berubah hijau → laporan berbukti.

Babak koreksi label itu bukan hiasan. Tanpa penilaian ulang setelah revisi, koreksi diarization justru membuat laporan kepatuhan salah. Semua layar diberi label tegas sebagai simulasi.

**Sesi sungguhan.** Masuk sebagai petugas, lalu klik **Mulai sesi** untuk menjalankan mikrofon, AssemblyAI, rule engine, dan laporan yang sebenarnya. Butuh backend berjalan, `ASSEMBLYAI_API_KEY` terisi, dan akun petugas. Demo terarah sengaja tetap terbuka tanpa login.

Gunakan earphone saat sesi sungguhan, supaya bisikan tidak masuk kembali ke mikrofon.

## Alur

```
Mic (PWA AudioWorklet / Flutter record)
  │  PCM16 16 kHz mono, frame 50–1000 ms, WebSocket
  ▼
Go Gateway ── goroutine per sesi, channel fan-out
  ├→ AssemblyAI Streaming STT + streaming speaker diarization
  │    └─ kalibrasi A/B → konfirmasi PETUGAS vs NASABAH sebelum scoring
  ├→ Rule engine: 5 butir kewajiban, semantic match via LLM Gateway
  ├→ Guardrail deterministik: janji terlarang ("dijamin untung", "pasti cair")
  └→ Bisikan → HANYA ke klien role officer → TTS di perangkat
             (nasabah tidak mendengar, latensi lebih rendah)
  ▼
Laporan: skor + transkrip ber-timestamp + bukti per butir
```

## 5 butir kewajiban

| Kode | Butir |
|---|---|
| `IDENTITY` | Identitas petugas & lembaga |
| `RATE` | Suku bunga / total biaya |
| `TENOR` | Jangka waktu & besaran cicilan |
| `PENALTY` | Denda keterlambatan |
| `RIGHT` | Hak nasabah menolak/membatalkan |

Sengaja hardcoded. Editor rule bukan scope hackathon.

## Cara sistem ini gagal dengan aman

Produk kepatuhan punya satu arah kegagalan yang jauh lebih berbahaya daripada yang lain. Checklist yang tertunda hijau hanya merepotkan petugas. Checklist yang hijau **padahal kewajibannya tidak pernah disampaikan** membuat laporannya berbohong — dan laporan itulah seluruh nilai produknya.

Karena itu setiap keputusan di bawah ini condong ke arah yang sama:

| Situasi | Yang dilakukan sistem |
|---|---|
| Ucapan tidak memuat bukti spesifik butirnya | Ditolak sebelum LLM dipanggil — hemat biaya sekaligus menahan false-green |
| LLM yakin di bawah 0,80 | Tidak dihitung memenuhi |
| Janji terlarang terucap | Dicocokkan frasa secara deterministik, bukan lewat LLM, supaya tidak bergantung sampling model |
| Suara belum dikalibrasi | Scoring ditahan sampai manusia mengonfirmasi mana petugas dan mana nasabah |
| Muncul pembicara ketiga | Diberi label sendiri (`max_speakers=3`), tidak mewarisi role siapa pun, tidak ikut dinilai |
| Diarization mengoreksi label belakangan | Ucapan yang ternyata milik petugas dinilai ulang — tanpa ini koreksi justru merusak laporan |
| Audio terlalu pelan, pecah, atau bising | Checklist ditahan dan peringatan ditampilkan; guardrail tetap jalan karena meloloskan janji terlarang lebih berbahaya |
| Stream upstream putus di tengah sesi | Petugas diberi tahu dan pengingat berkala berhenti, supaya UI tidak tampak menyimak padahal sudah mati |

## Privasi

- **Audio mentah tidak pernah disimpan.** Database hanya memuat transkrip dan bukti kepatuhan. Lihat [`bisik_backend/SCHEMA.md`](./bisik_backend/SCHEMA.md).
- **Nasabah tidak memakai aplikasi ini.** Dia tidak memegang perangkat dan tidak mendengar bisikannya; suaranya ditranskrip sebagai bagian dari percakapan.
- **Bisikan keluar di perangkat petugas**, bukan di-streaming balik dari server.

Persetujuan nasabah untuk ditranskrip adalah kewajiban lembaga yang memakai Bisik, dan alurnya belum ada di prototipe ini.

## Model kepercayaan

| Aktor | Login | Boleh apa |
|---|---|---|
| **Petugas** | Ya — PWA atau Flutter | Menjalankan **sesinya sendiri**, mengirim audio, menerima bisikan privat |
| **Supervisor** | Ya — `/supervisor` | Memantau sesi mana pun dan membaca laporannya. **Tidak pernah** mengirim audio |
| **Nasabah** | Tidak | Hanya berbicara. Tidak login, tidak menekan apa pun, tidak mendengar bisikan |

Yang ditegakkan backend:

- **Pemilik sesi diambil dari token, tidak pernah dari body permintaan.** Tanpa ini, klien bisa menyebutkan sendiri siapa dirinya dan atribusi laporan tidak membuktikan apa pun — padahal klaim utama produk ini justru bukti.
- **Peran WebSocket ditentukan token, bukan query.** Sebelumnya klien menulis `role=officer` sendiri, artinya siapa pun yang tahu ID sesi bisa mendorong audio ke percakapan orang lain.
- Laporan hanya bisa dibaca pemilik sesinya atau supervisor. Sesi hanya bisa diakhiri petugas yang memulainya.
- Kata sandi disimpan sebagai PBKDF2-HMAC-SHA256 600.000 iterasi. Login yang gagal memberi pesan yang sama persis untuk akun tidak ada dan kata sandi salah, supaya endpoint ini tidak bisa dipakai memetakan akun yang valid.

Batasan yang disadari: token bertanda tangan HMAC dan bersifat stateless, jadi **tidak bisa dicabut sebelum kedaluwarsa** — untuk produksi perlu daftar cabut atau umur token yang lebih pendek plus refresh. Token WebSocket dikirim lewat query karena browser tidak mengizinkan header kustom pada handshake, jadi bisa muncul di access log proxy. Aplikasi Flutter menyimpan token di memori saja, sehingga petugas masuk ulang setiap aplikasi dibuka.

## Batasan yang disadari

- **Jalur live belum terbukti ujung ke ujung.** Handshake `whisper-rt` + diarization sudah berhasil dan ukuran frame audio sudah diperbaiki, tetapi belum ada satu pun transkrip sungguhan yang diterima dari AssemblyAI. Mode demo terarah disediakan justru karena ini.
- **Percakapan tumpang tindih dari satu mikrofon tidak dapat dipisahkan sempurna.** UX meminta bicara bergantian; solusi produksi yang benar adalah mikrofon terpisah per pembicara.
- **Ambang quality gate audio belum dikalibrasi dengan rekaman nyata**, angkanya dipilih dari teori sinyal.
- **Flutter tertinggal dari PWA**: belum ada UI kalibrasi suara dan belum ada layar laporan. PWA adalah deliverable utama.
- Pemetaan ulang role di tengah sesi, riwayat sesi supervisor, dan reconnect otomatis belum ada.

## Tiga project

| Folder | Isi | Stack |
|---|---|---|
| [`bisik_frontend/`](./bisik_frontend) | PWA petugas (`/officer`) + dashboard supervisor (`/supervisor`) | React 19 · Vite · TypeScript · Redux Toolkit |
| [`bisik_backend/`](./bisik_backend) | Gateway WebSocket, rule engine, laporan | Go 1.26 · pgx · PostgreSQL 18 |
| [`bisik_mobile/`](./bisik_mobile) | App petugas di lapangan | Flutter · Riverpod |

Ketiganya memakai **clean architecture** dengan aturan dependensi yang sama:

```
domain  ←  application/usecase  ←  adapter/infrastructure  ←  presentation
   ↑ tidak pernah mengimpor ke arah kanan
```

Lapisan backend, pembagian state, dan daftar endpoint dijelaskan di
[`bisik_backend/README.md`](./bisik_backend/README.md). Skema database ada di
[`bisik_backend/SCHEMA.md`](./bisik_backend/SCHEMA.md).

## Menjalankan

Runbook lengkap — termasuk alamat untuk emulator, simulator, dan perangkat
fisik, serta skenario uji dua pembicara — ada di [`TESTING.md`](./TESTING.md).

### 1. Backend + PostgreSQL

```bash
cd bisik_backend
test -f .env || cp .env.example .env   # jangan timpa .env yang sudah terisi
make db                                # PostgreSQL 18 di Docker, migrasi otomatis
make dev                               # API di :8080
```

### 2. Frontend

```bash
cd bisik_frontend
cp .env.example .env
pnpm install && pnpm dev               # http://localhost:5173/officer
```

### 3. Mobile

```bash
cd bisik_mobile
flutter pub get
flutter run --dart-define=API_URL=http://10.0.2.2:8080 --dart-define=WS_URL=ws://10.0.2.2:8080
```

> `10.0.2.2` adalah alamat host dari emulator Android. Untuk perangkat fisik atau
> simulator iOS, ganti dengan IP LAN mesinmu — kalau tidak, aplikasi tidak akan
> menemukan gateway.

### Build deployment

Frontend dibuild memakai Node host (minimum Node 24), bukan image Node di
Docker. Hasil `bisik_frontend/dist` disimpan di repository supaya builder
deployment hanya memerlukan Go:

```bash
make deploy-build   # pnpm install + build frontend dengan Node host
make deploy-image   # langkah di atas + build image aplikasi
```

Setiap perubahan frontend wajib diikuti `make deploy-build` sebelum commit.

## Status

Ketiga project terkompilasi bersih: `go test ./...`, `go vet ./...`, `pnpm lint`,
`pnpm build`, dan `flutter analyze` semuanya hijau. Rule engine, evidence gate,
kalibrasi pembicara, safe-fail, dan kontrak protokol AssemblyAI memiliki unit test.

Yang belum terbukti disebutkan apa adanya di [Batasan yang disadari](#batasan-yang-disadari).
Catatan pengerjaan harian, keputusan teknis beserta alasannya, dan daftar
pekerjaan tersisa ada di [`PROGRESS.md`](./PROGRESS.md).
