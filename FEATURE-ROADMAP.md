# Urutan Pembuatan Fitur Bisik Menuju Submission

**Tanggal keputusan:** 26 September 2026

**Deadline:** 30 September 2026, 22:00 WIB

**Prinsip:** scope fitur dibekukan. Selesaikan bukti alur utama sebelum membuat
kemampuan baru.

## 1. Pelajaran dari pesaing

| Produk | Kekuatan yang mereka tonjolkan | Respons Bisik |
|---|---|---|
| Saakshi | Witness percakapan penjualan teregulasi, tahu siapa bicara, lalu menginterupsi mis-selling | Bedakan secara eksplisit: koreksi **privat** ke earpiece petugas, preventif sebelum kewajiban terlewat, Bahasa Indonesia, dan fail-safe saat identitas suara ragu |
| KiaOra Dispatch | Percakapan telepon, barge-in, tool calling, dispatch otomatis, dan angka dampak bisnis | Jangan meniru tool calling. Tunjukkan satu workflow lapangan yang pendek, stabil, serta manfaat operasional yang konkret |
| MockMate | Produk sangat lengkap: persona, scoring, analytics, report, PDF, history, dan deployment publik | Jangan mengejar breadth. Pastikan URL publik, polish demo, serta laporan evidence Bisik dapat dicoba juri tanpa penjelasan panjang |
| FarmVoice | Alur aman `Understand → Assess → Confirm → Act → Verify → Report` | Gunakan alur Bisik yang sama jelasnya: `Listen → Identify → Coach privately → Verify evidence → Report` |

Sumber publik: halaman submission resmi lablab.ai untuk
[Saakshi](https://lablab.ai/ai-hackathons/assemblyai-voice-agent-hackathon/monster),
[KiaOra Dispatch](https://lablab.ai/ai-hackathons/assemblyai-voice-agent-hackathon/shinydatatech/kiaora-dispatch),
[MockMate](https://lablab.ai/ai-hackathons/assemblyai-voice-agent-hackathon/twin-masters/mockmate-ai-voice-interview-coach),
dan [FarmVoice](https://lablab.ai/ai-hackathons/assemblyai-voice-agent-hackathon/farmvoice-ai/farmvoice-ai-voice-agent).

## 2. Urutan kerja yang wajib diikuti

### Tahap 0 — Bekukan scope

Status: **mulai sekarang**.

- Jangan menambah editor SOP, produk baru, PDF, history analytics, tool calling,
  blockchain certificate, atau fitur percakapan bot.
- PWA adalah deliverable utama. Flutter hanya diperbaiki jika ada blocker demo
  lapangan, bukan untuk menambah fitur baru.
- Setiap perubahan harus membantu salah satu dari empat pesan: private,
  preventive, Indonesia-first, atau safe when uncertain.

Selesai jika backlog baru sudah dipisahkan menjadi `wajib sebelum submit` dan
`setelah hackathon`.

### Tahap 1 — Stabilkan mesin dua suara

Status: **selesai 26 September 2026**.

1. [x] Perbaiki finalisasi `DualStreamSTT`: mapping role tidak boleh hilang sebelum
   transkrip terakhir selesai dipadankan.
2. [x] Teruskan kegagalan stream diarizer menjadi `session_error`; jangan biarkan
   UI terlihat sehat ketika teks masih masuk tetapi identitas pembicara mati.
3. [x] Tambahkan regression test untuk dua kondisi tersebut serta acknowledgement
   event error di adapter WebSocket.
4. [x] Jalankan `go test ./...`, `go vet ./...`, dan race detector pada paket
   AssemblyAI/WebSocket.

Hasil: role hasil kalibrasi disalin ke `dualSession` sampai kedua stream selesai;
error salah satu stream menghentikan penggabung dan dikirim ke petugas; seluruh
verifikasi lulus. Tahap aktif berikutnya adalah Tahap 2.

### Tahap 2 — Buktikan live dua orang lewat UI

Status: **blocker demo**.

Gunakan dialog di `TESTING.md` §6b, dua orang sungguhan, earphone, ruangan tanpa
musik. Uji dari PWA, bukan probe CLI saja.

Urutan bukti:

1. Kalibrasi dipandu: petugas lalu nasabah.
2. Dua label berbeda muncul dan dikunci ke role yang benar.
3. Ucapan nasabah tidak mengubah checklist.
4. Lima kewajiban petugas menjadi hijau dengan evidence.
5. Frasa `pasti disetujui` menghasilkan satu violation dan bisikan privat.
6. Audio ambigu/overlap atau speaker tidak dikenal tidak menghasilkan centang
   hijau.
7. Turn terakhir tetap benar setelah sesi diakhiri.
8. Laporan berisi 5/5, satu violation, skor 90, kutipan, timestamp, dan dua role.

Selesai jika alur ini berhasil dua kali berurutan. Simpan screenshot/log untuk
bahan video. Jika gagal, hanya perbaiki penyebab yang ditemukan; jangan menambah
fitur.

### Tahap 3 — Kalibrasi audio dan latensi seperlunya

Status: **hanya setelah uji Tahap 2**.

- Ubah ambang quality gate hanya jika rekaman nyata salah menuduh audio normal
  atau membiarkan audio buruk lolos.
- Ukur waktu dari akhir ucapan sampai transcript/checklist/nudge muncul.
- Target demo: transcript dan pelanggaran terasa real-time; catat angka nyata,
  jangan membuat klaim latensi yang belum diukur.

Selesai jika ruangan demo terpilih tidak menghasilkan warning palsu dan satu
angka latensi median dapat disebutkan dengan jujur.

### Tahap 4 — Deploy URL publik

Status: **setelah live lokal stabil**.

1. Rotasi API key AssemblyAI karena key lama pernah tampil di chat/screenshot.
2. Deploy Blueprint Render, isi secret dan akun seed di dashboard.
3. Uji dari jaringan/perangkat lain: landing, demo tanpa login, login petugas,
   sesi live, supervisor, dan laporan.
4. Pastikan build produksi tidak mengandung `localhost` dan halaman tetap dapat
   dibuka setelah refresh route.

Selesai jika juri dapat membuka URL anonim dan memainkan demo 11 detik tanpa
backend eksternal tambahan; sesi live juga berfungsi setelah login.

### Tahap 5 — Pertegas empat pembeda di permukaan produk

Status: **polish, bukan fitur baru**.

- **Private:** UI dan copy menyebut koreksi hanya terdengar di earphone petugas.
- **Preventive:** kartu terbesar menunjukkan satu kewajiban berikutnya, bukan
  hanya skor sesudah sesi.
- **Indonesia-first:** seluruh dialog demo Bahasa Indonesia dan kelima butir
  disebut `SOP demo`, bukan klaim regulasi resmi.
- **Safe when uncertain:** satu adegan demo memperlihatkan ucapan nasabah atau
  speaker `unknown` tidak membuat checklist hijau.

Selesai jika keempat pesan dapat dipahami dari layar/video tanpa membaca kode.

### Tahap 6 — Rekam video demo

Status: **setelah URL publik stabil**.

Urutan adegan yang disarankan:

1. Masalah: petugas lapangan bisa lupa disclosure saat berbicara alami.
2. Kalibrasi dua suara singkat.
3. Nasabah berbicara—checklist sengaja tidak berubah.
4. Petugas mengucapkan janji terlarang—bisikan privat muncul.
5. Kartu kewajiban berikutnya membimbing petugas sebelum sesi selesai.
6. Audio/label ambigu—sistem menahan evidence, bukan membuat centang palsu.
7. Laporan 90/100 dan supervisor melihat kutipan serta timestamp.
8. Tutup dengan arsitektur dua stream AssemblyAI dan tagline pembeda.

Selesai jika video memakai satu take aplikasi yang kredibel, tidak menampilkan
secret, terminal error, atau klaim regulasi yang belum diverifikasi.

### Tahap 7 — README, submission, lalu deck

Status: **terakhir**.

1. Sinkronkan README dengan arsitektur dua stream dan status pengujian nyata.
2. Isi halaman submission: masalah, pengguna, pembeda, penggunaan AssemblyAI,
   bukti safe-fail, URL publik, repo, dan video.
3. Buat deck ringkas dari narasi video—jangan menciptakan cerita baru.
4. Verifikasi semua link dan buka submission dari mode incognito.

Selesai jika README, video, deck, dan halaman lablab menyampaikan empat pembeda
yang sama dan semua tautan publik bekerja.

## 3. Fitur yang sudah tersedia—jangan dibuat ulang

- Dual stream: teks Indonesia + label pembicara.
- Kalibrasi suara berurutan di PWA dan Flutter.
- Private nudge ke petugas, termasuk penjaga gema TTS.
- Kartu kewajiban berikutnya dan checklist lima SOP demo.
- Evidence gate, confidence minimum, deterministic guardrail.
- Safe-fail untuk audio buruk, `unknown`, dan pembicara ketiga.
- Re-evaluation setelah revisi diarization.
- Laporan berbukti dan monitoring supervisor.
- Demo lokal 11 detik tanpa API.
- Autentikasi petugas/supervisor dan migrasi PostgreSQL production-safe.

## 4. Ditunda sampai setelah hackathon

- Editor SOP/rule dan dukungan banyak produk.
- Remap role di tengah sesi dengan recompute penuh.
- Riwayat analytics, tren skor, PDF export.
- Reconnect/resume sesi lintas jaringan.
- Multi-channel atau dua perangkat audio terpisah.
- Integrasi CRM, telepon, atau tool calling.
- Sertifikat/hash-chain/blockchain.
- Klaim kepatuhan terhadap regulasi nyata sebelum review legal.

Alasannya sama: fitur-fitur ini memperlebar produk tetapi tidak memperkuat bukti
empat pembeda utama dalam waktu tersisa.
