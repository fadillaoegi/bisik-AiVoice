# Menjalankan dan Menguji Bisik

Runbook lokal untuk tiga project: backend Go, frontend PWA, dan mobile Flutter.
Jalankan backend lebih dulu karena frontend mode live dan mobile memakai REST +
WebSocket dari backend yang sama.

## 1. Prasyarat

- Docker Desktop aktif.
- Go 1.26 tersedia.
- Node 24 dan pnpm tersedia.
- Flutter tersedia untuk pengujian mobile.
- `ASSEMBLYAI_API_KEY` sudah terisi di `bisik_backend/.env`.
- `AUTH_SECRET` sudah terisi di `bisik_backend/.env` — backend menolak start
  kalau kosong. Buat dengan `cd bisik_backend && make auth-secret`.
- `SEED_OFFICER_PASSWORD` dan `SEED_SUPERVISOR_PASSWORD` sudah terisi. Akun
  hanya dibuat saat tabel `users` masih kosong; kalau sudah pernah dibuat,
  mengubah variabel ini tidak mengubah kata sandinya.
- Gunakan earphone saat tes live agar suara bisikan tidak masuk lagi ke mikrofon.

Jangan menyalin API key ke terminal, screenshot, log, atau commit Git.

## 2. Jalankan backend dan PostgreSQL

Buka terminal pertama dari root repository:

```bash
cd bisik_backend
make db
docker compose ps
make dev
```

Tunggu sampai `bisik_postgres` berstatus `healthy` dan backend berjalan pada
port `8080`. Migrasi database dijalankan otomatis oleh backend saat startup.

Periksa health dari terminal lain:

```bash
curl -sS http://localhost:8080/health
```

Hasil yang diharapkan:

```json
{"service":"bisik_backend","status":"ok"}
```

Migrasi otomatis berjalan saat backend mulai. Untuk menjalankannya secara
terpisah dan melihat riwayat versinya:

```bash
cd bisik_backend
make migrate
docker compose exec postgres psql -U saksi -d saksi -c \
  "SELECT version, applied_at FROM schema_migrations ORDER BY version;"
```

File `.env` lokal sudah tersedia. Jangan menjalankan `cp .env.example .env`
karena dapat menimpa API key yang telah dipasang.

## 2b. Akun petugas dan supervisor

Akun awal dibuat otomatis saat backend pertama kali start dengan tabel `users`
kosong. Log backend akan mencetak `akun awal dibuat`.

| Peran | Nama pengguna bawaan | Kata sandi |
|---|---|---|
| Petugas | `petugas` | isi `SEED_OFFICER_PASSWORD` |
| Supervisor | `supervisor` | isi `SEED_SUPERVISOR_PASSWORD` |

Periksa akunnya sudah ada:

```bash
cd bisik_backend
docker compose exec postgres psql -U saksi -d saksi -c \
  "SELECT username, role FROM users ORDER BY role;"
```

Kalau tabelnya kosong padahal backend sudah jalan, berarti kedua variabel kata
sandi masih kosong — akun tanpa kata sandi sengaja tidak dibuat.

Untuk menguji dari nol setelah mengubah kata sandi seed:

```bash
docker compose exec postgres psql -U saksi -d saksi -c "DELETE FROM users;"
```

lalu restart backend.

## 2c. Menguji pemisahan peran

Ini membuktikan otorisasinya benar-benar ditegakkan, bukan sekadar disembunyikan UI.

1. Masuk sebagai **petugas** di <http://localhost:5173/officer>, mulai satu sesi,
   lalu salin ID sesinya dari log backend atau dari halaman supervisor.
2. Di jendela penyamaran, buka <http://localhost:5173/supervisor> dan masuk
   sebagai **supervisor**. Sesi tadi harus muncul di daftar **Sesi terbaru** —
   tidak perlu lagi menyalin ID dari psql.
3. Coba masuk sebagai supervisor di halaman `/officer`. Harus ditolak dengan
   pesan bahwa halaman itu untuk petugas.
4. Buktikan token benar-benar diperiksa, bukan hanya UI:

   ```bash
   # Tanpa token → 401
   curl -s -o /dev/null -w '%{http_code}\n' http://localhost:8080/api/obligations

   # Dengan token petugas → 200
   TOKEN=$(curl -s -X POST http://localhost:8080/api/auth/login \
     -H 'Content-Type: application/json' \
     -d '{"username":"petugas","password":"<sandi-petugas>"}' \
     | python3 -c 'import sys,json;print(json.load(sys.stdin)["token"])')
   curl -s -o /dev/null -w '%{http_code}\n' \
     -H "Authorization: Bearer $TOKEN" http://localhost:8080/api/obligations

   # Petugas mencoba daftar sesi milik supervisor → 403
   curl -s -o /dev/null -w '%{http_code}\n' \
     -H "Authorization: Bearer $TOKEN" http://localhost:8080/api/sessions
   ```

   Hasil yang diharapkan berurutan: `401`, `200`, `403`.

## 3. Jalankan frontend PWA

Buka terminal kedua dari root repository:

```bash
cd bisik_frontend
pnpm install
pnpm dev
```

Buka:

- Petugas: <http://localhost:5173/officer>
- Supervisor: <http://localhost:5173/supervisor>

Masuk dengan akun petugas dari §2b. **Demo terarah sengaja tetap bisa diputar
tanpa login** — juri harus bisa mencobanya tanpa kredensial.

Di halaman petugas tersedia dua jalur:

1. **Putar demo terarah · 11 detik** menjalankan simulasi tanpa mikrofon atau API
   eksternal. Semua hasil diberi label simulasi.
2. **Mulai sesi** menjalankan mikrofon, WebSocket, AssemblyAI, rule engine, dan
   laporan sebenarnya.

Pilih **Allow/Izinkan** saat browser meminta akses mikrofon.

## 4. Jalankan mobile Flutter

Pastikan backend masih berjalan, lalu buka terminal ketiga dari root repository:

```bash
cd bisik_mobile
flutter pub get
flutter devices
```

Pilih command sesuai target.

### Android Emulator

`10.0.2.2` adalah alamat Mac dari Android Emulator:

```bash
flutter run \
  --dart-define=API_URL=http://10.0.2.2:8080 \
  --dart-define=WS_URL=ws://10.0.2.2:8080
```

### iOS Simulator

```bash
flutter run -d ios \
  --dart-define=API_URL=http://127.0.0.1:8080 \
  --dart-define=WS_URL=ws://127.0.0.1:8080
```

Jika nama device bukan `ios`, salin ID dari `flutter devices` lalu gunakan
`flutter run -d <DEVICE_ID> ...`.

### iPhone atau Android fisik

Laptop dan HP harus berada pada Wi-Fi yang sama. Cari IP LAN Mac:

```bash
ifconfig en0 | awk '/inet / {print $2}'
```

Contoh jika hasilnya `192.168.1.8`:

```bash
flutter run -d <DEVICE_ID> \
  --dart-define=API_URL=http://192.168.1.8:8080 \
  --dart-define=WS_URL=ws://192.168.1.8:8080
```

Izinkan akses **Microphone** dan **Local Network** jika diminta. IP LAN dapat
berubah setelah berpindah jaringan — periksa ulang dengan perintah di atas
setiap kali berganti Wi-Fi.

> **Kenapa HTTP polos bisa dipakai di perangkat.** iOS memblokir cleartext
> lewat App Transport Security dan Android 9+ memblokirnya lewat kebijakan
> jaringan. Keduanya sudah diberi pengecualian: `NSAppTransportSecurity` plus
> `NSLocalNetworkUsageDescription` di `ios/Runner/Info.plist`, dan
> `android:usesCleartextTraffic="true"` di AndroidManifest. Tanpa itu,
> koneksi gagal **tanpa pesan yang berguna** dan dialog izin jaringan lokal
> tidak pernah muncul. Pengecualian ini hanya melayani pengujian lokal;
> deployment Render memakai HTTPS. Untuk demo final di HP fisik, gunakan URL
HTTPS/WSS hasil deployment Render.

### macOS desktop

```bash
flutter run -d macos \
  --dart-define=API_URL=http://127.0.0.1:8080 \
  --dart-define=WS_URL=ws://127.0.0.1:8080
```

Jalankan satu aplikasi petugas saja saat tes audio: PWA **atau** Flutter.

## 5. Pemeriksaan otomatis

Backend:

```bash
cd bisik_backend
go test ./...
go vet ./...
```

Frontend:

```bash
cd bisik_frontend
pnpm lint
pnpm build
```

Mobile:

```bash
cd bisik_mobile
flutter analyze
flutter build apk --debug              # memeriksa resource Android
flutter build ios --debug --no-codesign
```

`flutter analyze` **tidak** memeriksa berkas XML Android maupun kompatibilitas
SDK plugin. Dua kegagalan build pernah lolos dari analyze dan baru ketahuan
saat build sungguhan, jadi jalankan build minimal sekali sebelum menyerahkan
hasil.

## 6. Pengujian live AssemblyAI

Gunakan dua orang dengan suara berbeda. PWA tidak lagi menganggap pembicara
pertama sebagai petugas: role dikunci lewat kalibrasi sebelum scoring. Beri jeda
sekitar satu detik antar giliran dan ucapkan setiap kalimat dengan jelas.

1. Buka `/officer`, pastikan backend berjalan, lalu klik **Mulai sesi**.
2. Pastikan indikator berubah menjadi `Terhubung · merekam` dan panel
   **Kenali dua suara** tampil.
3. Petugas mengatakan:

   > Saya petugas yang menjalankan sesi ini.

4. Nasabah mengatakan:

   > Saya nasabah dan siap memulai.

5. Tunggu sampai dua kartu `Suara A/B` muncul. Baca contoh kalimatnya, pilih
   kartu yang benar sebagai **Petugas**, lalu klik **Konfirmasi dan mulai
   penilaian**. Kedua kalimat kalibrasi tidak boleh mengubah checklist atau
   masuk ke laporan.
6. Setelah panel kalibrasi hilang, petugas mengatakan:

   > Perkenalkan, nama saya Rina dari Bank Nusantara.

7. Nasabah mengatakan:

   > Kalau saya ajukan hari ini, apakah pasti diterima?

8. Petugas sengaja mengatakan janji terlarang:

   > Tenang, pengajuan Bapak pasti disetujui.

   Hasil yang diharapkan: pelanggaran `pasti disetujui` tersimpan dan petugas
   mendengar bisikan koreksi melalui earphone.

9. Petugas mengoreksi dan melanjutkan:

   > Maaf, persetujuan tetap mengikuti penilaian. Suku bunganya 1,2 persen per
   > bulan dan biaya administrasinya seratus ribu rupiah.

   > Tenornya 12 bulan dengan cicilan satu juta rupiah per bulan.

   > Jika terlambat, ada denda 0,1 persen per hari.

   > Bapak berhak menolak atau membatalkan pengajuan ini.

10. Pastikan kelima checklist berubah hijau dengan confidence minimal 80%.
11. Klik **Akhiri sesi** dan tunggu laporan. Finalisasi juga menunggu koreksi
    label pembicara terakhir dari AssemblyAI.
12. Periksa laporan:

    - kewajiban `5/5`;
    - satu pelanggaran `pasti disetujui`;
    - skor yang diharapkan `90/100`;
    - setiap kewajiban memiliki kutipan dan timestamp;
    - transkrip membedakan Petugas dan Nasabah;
    - kalimat kalibrasi tidak ada di transkrip laporan.

Jika hanya satu kartu suara yang muncul, ulangi kalimat orang kedua dengan lebih
panjang dan pastikan kedua orang tidak berbicara bersamaan. Jika checklist
kosong, periksa log backend untuk error model, autentikasi, atau LLM Gateway.
Status WebSocket terhubung saja belum membuktikan jalur live sukses.

## 6a. Trace jalur sesi (untuk diagnosis)

Urutan yang harus terlihat ketika satu sesi live berjalan adalah:

```text
Login petugas
  → POST /api/sessions (pemilik diambil dari token)
  → WebSocket /ws (token menentukan peran officer)
  → dua STT Begin ke AssemblyAI atas PCM16 yang sama
      • whisper-rt: sumber teks Bahasa Indonesia
      • universal-streaming-multilingual: sumber label A/B/C pembicara
  → begin_speaker_calibration
  → mikrofon → AudioWorklet/Flutter → PCM16 binary
  → worklet mengumpulkan 2.048 sample (128 ms)
  → kedua backend stream mengamankan frame 50–1.000 ms
  → turn teks menunggu label maksimal 1,2 detik
  → label dipilih dari durasi tumpang tindih terbesar; revisi terlambat
    mengoreksi utterance yang sudah masuk
  → kalibrasi dari stream label mengirim kartu Suara A/B, lalu
    confirm_speaker_roles
  → turn petugas disimpan sebagai utterance
  → quality gate → speaker mapping → evidence gate/LLM matcher
  → obligation_satisfied atau violation + nudge hanya ke petugas
  → Stop flush frame terakhir → Terminate kedua stream → revision terakhir
  → score/report (evidence, timestamp, violation, transcript)
```

Supervisor hanya mengambil snapshot laporan sesi aktif lalu berlangganan event
WebSocket; supervisor tidak pernah mengirim audio. Audio mentah tidak disimpan.
Yang masuk database adalah `utterances`, status kewajiban, pelanggaran, dan
label sumber speaker untuk audit. Jika upstream putus, jalur aman adalah event
`session_error` ke petugas dan nudge berhenti; checklist tidak boleh berubah
hijau dari audio yang ditandai buruk, speaker `unknown`, atau gema bisikan
aplikasi.

Catatan diagnosis: frame 128 sample/8 ms yang dulu menyebabkan `resource tidak
ditemukan` sudah diperbaiki di worklet dan backend serta memiliki unit test.
`DualStreamSTT` juga sudah terbukti terhadap AssemblyAI sungguhan memakai PCM
dua suara: teks Indonesia datang dari `whisper-rt` dan label A/B dari model
multilingual. Yang belum terbukti adalah pengujian penuh **dua orang melalui
UI** (mikrofon, kalibrasi, scoring, dan laporan), jadi gunakan log backend dan
laporan sebagai bukti, bukan hanya indikator “Terhubung”.

Sebelum tes, periksa banner backend menyatakan mode dua stream dan dua model
di atas. Jika `ASSEMBLYAI_DIARIZER_MODEL` dikosongkan, mode kembali satu stream:
lebih hemat biaya, tetapi atribusi pembicara mengikuti model teks dan tidak
layak dipakai untuk demo klaim diarization.

## 6b. Dialog uji siap baca (dua orang, PWA live)

Gunakan **P** (petugas/operator) dan **N** (nasabah). Operator menjalankan
langkah dalam tanda kurung; kalimat setelah nama pembicara dibaca persis dan
diberi jeda sekitar satu detik. Jangan memakai speaker laptop—gunakan earphone.

### Babak A — kalibrasi dua suara

1. **P (klik Mulai sesi):** tunggu status `Terhubung · merekam` dan panel
   **Kenali dua suara**.
2. **P:** “Saya petugas yang menjalankan sesi ini. Nama saya Rina.”
3. **N:** “Saya nasabah yang sedang mengajukan kredit.”
4. **P (operator):** tunggu kartu `Suara A` dan `Suara B`, pilih label yang
   contoh suaranya milik P sebagai **Petugas**, label lainnya sebagai
   **Nasabah**, lalu klik **Konfirmasi dan mulai penilaian**.

Ekspektasi: dua ucapan kalibrasi tidak masuk checklist/laporan. Jika label
terbalik, jangan menebak—ulangi dua kalimat dengan suara yang lebih panjang dan
terpisah, lalu konfirmasi ulang.

### Babak B — jalur berhasil + satu pelanggaran

**P:** “Perkenalkan, nama saya Rina dari Bank Nusantara.”  
Ekspektasi: kewajiban identitas menjadi hijau.

**N:** “Kalau saya ajukan hari ini, apakah pasti diterima?”  
Ekspektasi: tidak mengubah checklist petugas.

**P:** “Tenang, pengajuan Bapak pasti disetujui.”  
Ekspektasi: satu violation `pasti disetujui` dan bisikan koreksi ke earphone P.

**P:** “Maaf, persetujuan tetap mengikuti penilaian. Suku bunganya 1,2 persen
per bulan dan biaya administrasinya seratus ribu rupiah.”  
Ekspektasi: kewajiban suku bunga/biaya hijau.

**P:** “Tenornya 12 bulan dengan cicilan satu juta rupiah per bulan.”  
Ekspektasi: kewajiban tenor/cicilan hijau.

**P:** “Jika terlambat, ada denda 0,1 persen per hari.”  
Ekspektasi: kewajiban denda hijau.

**P:** “Bapak berhak menolak atau membatalkan pengajuan ini.”  
Ekspektasi: kewajiban hak membatalkan hijau.

**P (tunggu 2 detik):** pastikan kalimat hak membatalkan sudah tampak di
transkrip sebagai Petugas sebelum mengakhiri. Ini khusus menguji bahwa turn
terakhir dari dua stream sudah selesai dipadankan.

**P (klik Akhiri sesi):** tunggu laporan. Hasil target: `5/5`, satu violation,
skor `90/100`, setiap butir memiliki kutipan/timestamp, dan kalimat kalibrasi
tidak muncul di laporan.

### Babak C — safe-fail yang sengaja diuji

- **N** mengucapkan kewajiban (misalnya “bunganya satu persen”). Checklist
  petugas harus tetap pending; hanya ucapan **Petugas** yang boleh memenuhi.
- Salah satu orang hanya berkata “Halo” atau berbicara sambil bertumpuk. Sistem
  boleh menandai `PENDING`, `speaker_unknown`, atau `audio_quality`; ucapan itu
  tidak boleh menjadi evidence hijau.
- Tutup koneksi upstream atau cabut izin mikrofon hanya jika siap mengulang
  sesi. Ekspektasi: banner `session_error`/peringatan audio tampil dan nudge
  berhenti, bukan aplikasi terus mengaku merekam.
- Saat menjalankan dua stream, putusnya stream teks **atau** stream label harus
  menampilkan `session_error` dan menghentikan nudge. Jika teks terus masuk tanpa
  atribusi setelah error diarizer, anggap regression dan jangan memakai laporan.

### Babak D — supervisor dan mobile

Setelah Babak B menghasilkan setidaknya satu checklist hijau, buka `/supervisor`
di jendela lain dan pilih sesi aktif. Snapshot harus langsung menampilkan progres
lama, lalu event baru terus masuk. Untuk Flutter/iPhone, baca dialog yang sama;
yang berubah hanya cara menjalankan (login → Mulai sesi → kalibrasi → Akhiri).

## 7. Menguji pengingat berkala

Nilai normal `NUDGE_INTERVAL_SECONDS` adalah 45 detik. Untuk tes cepat, ubah
sementara nilainya di `bisik_backend/.env` menjadi `10`, lalu restart backend.
Mulai sesi dan jangan ucapkan salah satu kewajiban. Setelah sekitar 10 detik,
petugas seharusnya mendengar satu kewajiban pending. Kembalikan ke `45` setelah
pengujian.

## 8. Memantau dari halaman supervisor

Ambil ID sesi aktif dari PostgreSQL:

```bash
cd bisik_backend
docker compose exec postgres psql -U saksi -d saksi -Atc \
  "SELECT id FROM sessions WHERE status='active' ORDER BY started_at DESC LIMIT 1;"
```

Sejak autentikasi ditambahkan, ID sesi tidak perlu lagi disalin dari psql.
Buka <http://localhost:5173/supervisor>, masuk sebagai supervisor, lalu pilih
sesi dari daftar **Sesi terbaru**. Transkrip dan pelanggaran baru harus muncul
tanpa supervisor mengirim audio.

Supervisor yang bergabung **di tengah sesi** kini langsung melihat progres yang
sudah terjadi: checklist yang sudah hijau, pelanggaran yang sudah tercatat, dan
transkrip sejauh ini. Snapshot diambil sebelum WebSocket tersambung, lalu event
live melanjutkannya.

Cara membuktikannya dalam satu percobaan:

1. Di halaman petugas, mulai sesi dan ucapkan beberapa kewajiban sampai ada
   yang berubah hijau.
2. **Baru setelah itu** buka halaman supervisor dan pilih sesi tersebut.
3. Checklist hijau dan pelanggaran dari langkah 1 harus sudah tampil begitu
   halaman terbuka — bukan menunggu ucapan berikutnya.

Tombol **← Kembali ke daftar sesi** memungkinkan berpindah antar sesi tanpa
memuat ulang halaman.

## 9. Menghentikan semua service

- Tekan `Ctrl+C` pada terminal backend, frontend, dan Flutter.
- Hentikan PostgreSQL jika sudah selesai:

```bash
cd bisik_backend
make db-down
```

`make db-down` mempertahankan volume database. `make db-reset` menghapus volume
dan seluruh data lokal, jadi gunakan hanya saat ingin memulai dari nol.

## 10. Gejala umum

| Gejala | Yang diperiksa |
|---|---|
| `/health` tidak bisa dibuka | Docker Desktop, `docker compose ps`, dan terminal `make dev` |
| Frontend gagal memuat kewajiban | Backend belum berjalan atau URL di `bisik_frontend/.env` salah |
| `401`/`403` dari AssemblyAI | API key salah, kedaluwarsa, atau sudah dirotasi |
| Backend menolak start | `AUTH_SECRET` kosong di `.env`; buat dengan `make auth-secret` |
| Tidak bisa masuk, akun tidak dikenal | Tabel `users` kosong karena kata sandi seed belum diisi saat backend pertama start |
| Semua permintaan `401` setelah restart | `AUTH_SECRET` berubah, jadi token lama tidak berlaku. Masuk ulang |
| Mikrofon tidak merekam | Izin mikrofon browser/OS dan device input aktif |
| Mobile tidak terhubung | Gunakan alamat target sesuai §4, bukan `localhost` untuk Android Emulator/HP fisik |
| WebSocket terhubung tetapi tidak ada transkrip | Log backend dan kompatibilitas `whisper-rt` + diarization |
| Checklist tidak hijau | Kalimat harus memenuhi evidence gate dan confidence LLM minimal 80% |
