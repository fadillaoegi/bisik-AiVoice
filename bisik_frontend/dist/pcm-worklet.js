// Bisik — AudioWorklet: Float32 -> PCM16 mono 16 kHz.
// Sengaja vanilla dan berdiri sendiri: ini bagian paling rapuh di pipeline,
// jangan dibungkus abstraksi framework.
//
// Dua aturan AssemblyAI Streaming yang wajib dipatuhi di sini:
//  1. Satu frame WebSocket harus berisi 50–1000 ms audio. Render quantum
//     Web Audio hanya 128 sample (8 ms), jadi frame HARUS diakumulasi dulu.
//     Mengirim langsung per quantum membuat upstream menutup koneksi
//     (close 3007) dan backend melaporkan "push audio gagal".
//  2. Sample rate harus sama dengan yang dideklarasikan gateway (16 kHz).
//     Browser boleh menolak permintaan AudioContext 16 kHz, karena itu
//     worklet menurunkan sample rate sendiri kalau ternyata berbeda.

const TARGET_SAMPLE_RATE = 16000
// 2048 sample = 4096 byte = 128 ms: ukuran frame yang direkomendasikan AssemblyAI.
const FRAME_SAMPLES = 2048
// 800 sample = 50 ms: batas minimum yang masih boleh dikirim saat flush terakhir.
const MIN_FLUSH_SAMPLES = 800
// Jendela pengukuran kualitas audio: 1 detik pada laju target.
const STATS_WINDOW_SAMPLES = TARGET_SAMPLE_RATE
// Sample dianggap pecah kalau menyentuh batas skala penuh.
const CLIP_THRESHOLD = 0.98

/*
 * Noise gate adaptif.
 *
 * Yang BISA dilakukan: meredam bunyi latar di sela-sela ucapan. Ini penting
 * untuk diarization — musik atau obrolan jauh yang terus mengalir saat tidak
 * ada yang bicara bisa dikira pembicara tambahan, dan kalibrasi jadi kacau.
 *
 * Yang TIDAK bisa: menghapus musik dari suara orang yang sedang bicara.
 * Musik itu broadband dan berstruktur harmonik seperti suara manusia; tidak
 * ada gate sederhana yang mampu memisahkannya. Solusinya lingkungan, bukan kode.
 *
 * Ambangnya mengikuti lantai kebisingan ruangan, bukan angka tetap, supaya
 * tetap bekerja di warung ramai maupun ruang tamu sunyi.
 */
// Bicara harus sekian kali di atas lantai bising sebelum gate dibuka.
const GATE_OPEN_RATIO = 2.2
// Di bawah ini tidak mungkin ucapan, seberapa pun sunyinya ruangan.
// Sengaja jauh di bawah ambang "suara terlalu pelan" (0.008) di
// worklet_audio_repository.ts: mikrofon bervolume rendah masih ucapan sah,
// dan gate tidak boleh ikut mencacahnya.
const GATE_ABSOLUTE_FLOOR = 0.002
// Gate hanya masuk akal di ruangan yang memang berisik. Di ruangan sunyi
// tidak ada yang perlu diredam, dan menyalakannya justru berisiko memotong
// ucapan pelan — kerugian tanpa imbalan.
const GATE_MIN_NOISE_FLOOR = 0.004
// Gate tetap terbuka sesaat setelah suara berhenti, supaya akhir kata tidak terpotong.
const GATE_HANGOVER_MS = 320
// Saat tertutup audio diredam, bukan dinolkan: nol mendadak terdengar seperti
// potongan dan bisa membingungkan deteksi ucapan di sisi server.
// Nilainya sengaja tidak agresif — peredaman berlebihan lebih merugikan
// diarization daripada kebisingan yang lolos.
const GATE_CLOSED_GAIN = 0.18
// Lantai bising turun cepat mengikuti jeda, naik lambat supaya ucapan panjang
// tidak ikut dianggap kebisingan.
const FLOOR_FALL = 0.05
const FLOOR_RISE = 0.0008

class PCMProcessor extends AudioWorkletProcessor {
  constructor() {
    super()
    this.buffer = new Int16Array(FRAME_SAMPLES)
    this.filled = 0
    // Posisi baca pecahan untuk decimation; disimpan antar quantum supaya
    // tidak ada sample yang hilang atau terhitung dua kali di batas blok.
    this.readCursor = 0
    this.ratio = sampleRate / TARGET_SAMPLE_RATE
    // Statistik mentah untuk quality gate. Worklet sengaja hanya mengukur
    // dan tidak memutuskan: ambang batasnya ada di sisi TypeScript supaya
    // mudah dibaca dan diubah tanpa menyentuh file paling rapuh ini.
    this.statSamples = 0
    this.statSumSquares = 0
    this.statPeak = 0
    this.statClipped = 0

    // Keadaan noise gate.
    this.noiseFloor = GATE_ABSOLUTE_FLOOR
    this.gain = 1
    this.hangoverBlocks = 0
    this.gateEnabled = true
    this.gatedBlocks = 0
    this.totalBlocks = 0
    this.targetGain = 1

    this.port.onmessage = (e) => {
      if (e.data && e.data.type === 'gate') {
        this.gateEnabled = e.data.enabled !== false
        return
      }
      if (e.data === 'flush') this.flush(MIN_FLUSH_SAMPLES)
    }
  }

  /** Kirim isi buffer kalau sudah memenuhi jumlah sample minimum. */
  flush(minSamples) {
    if (this.filled < minSamples) return
    const frame = this.buffer.slice(0, this.filled)
    this.filled = 0
    this.port.postMessage(frame.buffer, [frame.buffer])
  }

  push(sample) {
    const raw = Math.max(-1, Math.min(1, sample))

    // Statistik dihitung dari sinyal MENTAH, sebelum gate.
    // Kalau diukur setelah gate, peredaman kebisingan akan membuat vonis
    // "kebisingan latar tinggi" tidak pernah menyala — sistem jadi menilai
    // hasil pekerjaannya sendiri dan petugas kehilangan peringatan.
    const magnitude = Math.abs(raw)
    this.statSamples++
    this.statSumSquares += raw * raw
    if (magnitude > this.statPeak) this.statPeak = magnitude
    if (magnitude >= CLIP_THRESHOLD) this.statClipped++
    if (this.statSamples >= STATS_WINDOW_SAMPLES) this.reportStats()

    // Gain digeser bertahap menuju target supaya tidak ada bunyi 'klik'
    // di tiap pembukaan dan penutupan gate.
    this.gain += (this.targetGain - this.gain) * 0.05
    const s = raw * this.gain

    this.buffer[this.filled++] = s < 0 ? s * 0x8000 : s * 0x7fff
    if (this.filled === FRAME_SAMPLES) this.flush(FRAME_SAMPLES)
  }

  /** Perbarui lantai bising dan keadaan gate dari satu blok render. */
  updateGate(channel) {
    if (!this.gateEnabled) {
      this.targetGain = 1
      return
    }

    let sumSquares = 0
    for (let i = 0; i < channel.length; i++) sumSquares += channel[i] * channel[i]
    const blockRms = Math.sqrt(sumSquares / channel.length)

    const adapt = blockRms < this.noiseFloor ? FLOOR_FALL : FLOOR_RISE
    this.noiseFloor += (blockRms - this.noiseFloor) * adapt

    // Ruangan sudah sunyi: biarkan sinyal apa adanya.
    if (this.noiseFloor < GATE_MIN_NOISE_FLOOR) {
      this.targetGain = 1
      this.totalBlocks++
      return
    }

    const threshold = Math.max(this.noiseFloor * GATE_OPEN_RATIO, GATE_ABSOLUTE_FLOOR)
    const blockMs = (channel.length / sampleRate) * 1000

    if (blockRms > threshold) {
      this.hangoverBlocks = Math.ceil(GATE_HANGOVER_MS / blockMs)
    } else if (this.hangoverBlocks > 0) {
      this.hangoverBlocks--
    }

    const open = this.hangoverBlocks > 0
    this.targetGain = open ? 1 : GATE_CLOSED_GAIN

    this.totalBlocks++
    if (!open) this.gatedBlocks++
  }

  reportStats() {
    this.port.postMessage({
      rms: Math.sqrt(this.statSumSquares / this.statSamples),
      peak: this.statPeak,
      clippedRatio: this.statClipped / this.statSamples,
      noiseFloor: this.noiseFloor,
      // Berapa bagian waktu gate menutup — petunjuk seberapa ramai ruangannya.
      gatedRatio: this.totalBlocks > 0 ? this.gatedBlocks / this.totalBlocks : 0,
    })
    this.gatedBlocks = 0
    this.totalBlocks = 0
    this.statSamples = 0
    this.statSumSquares = 0
    this.statPeak = 0
    this.statClipped = 0
  }

  process(inputs) {
    const input = inputs[0]
    if (!input || input.length === 0) return true

    const channel = input[0]
    if (!channel) return true

    this.updateGate(channel)

    if (this.ratio === 1) {
      for (let i = 0; i < channel.length; i++) this.push(channel[i])
      return true
    }

    // Decimation sederhana: ambil sample terdekat pada laju target.
    // readCursor dibawa lintas blok agar jarak antar sample tetap konsisten.
    for (let i = this.readCursor; i < channel.length; i += this.ratio) {
      this.push(channel[Math.floor(i)])
    }
    this.readCursor = (this.readCursor - channel.length) % this.ratio
    if (this.readCursor < 0) this.readCursor += this.ratio
    return true
  }
}

registerProcessor('pcm-processor', PCMProcessor)
