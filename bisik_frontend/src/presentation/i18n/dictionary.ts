/**
 * Kamus antarmuka Bisik — Bahasa Indonesia (bawaan) dan Bahasa Inggris.
 *
 * Yang diterjemahkan hanya antarmuka. Isi percakapan, frasa terlarang, dan
 * SUARA bisikan di earpiece tetap Bahasa Indonesia: percakapannya memang
 * berbahasa Indonesia, dan echo guard di backend mencocokkan bisikan dalam
 * bahasa itu. Earpiece berbahasa Inggris bisa lolos dari penjagaan gema dan
 * memenuhi checklist dengan suaranya sendiri.
 *
 * Kamus Inggris diberi tipe `Dictionary`, jadi kunci yang lupa diterjemahkan
 * gagal di `tsc`, bukan diam-diam tampil kosong di depan juri.
 */

export type Lang = 'id' | 'en'

export const DEFAULT_LANG: Lang = 'id'

const id = {
  /** Dipakai `toLocaleString` dan atribut `<html lang>`. */
  locale: 'id-ID',
  brand: {
    tagline: 'Kopilot kepatuhan petugas lapangan',
    supervisorTitle: 'Bisik Supervisor',
    supervisorTagline: 'Pemantauan kepatuhan berbasis bukti',
  },
  language: {
    label: 'Bahasa antarmuka',
    short: { id: 'ID', en: 'EN' } as Record<Lang, string>,
    name: { id: 'Bahasa Indonesia', en: 'English' } as Record<Lang, string>,
  },
  auth: {
    checkingSession: 'Memeriksa sesi login…',
    officerTitle: 'Masuk sebagai petugas',
    officerHint: 'Sesi dan laporannya akan tercatat atas nama akun ini.',
    supervisorTitle: 'Masuk sebagai supervisor',
    supervisorHint: 'Supervisor memantau dan membaca laporan, tidak pernah mengirim audio.',
    username: 'Nama pengguna',
    password: 'Kata sandi',
    checking: 'Memeriksa…',
    submit: 'Masuk',
    signedInAs: 'Masuk sebagai',
    logout: 'Keluar',
    logoutTitle: 'Keluar dari akun?',
    logoutBody: 'Anda perlu masuk lagi dengan nama pengguna dan kata sandi untuk melanjutkan.',
    cancel: 'Batal',
  },
  roles: { officer: 'petugas', supervisor: 'supervisor' },
  speakers: {
    officer: 'Petugas',
    customer: 'Nasabah',
    unknown: '—',
    unknownLong: 'Tidak diketahui',
  },
  officer: {
    start: 'Mulai sesi',
    starting: 'Membuka sesi…',
    gatewayDown:
      'Gateway tidak bisa dihubungi, jadi sesi sungguhan belum bisa dimulai. Demo terarah di bawah tetap berjalan penuh — dia tidak memakai backend maupun API eksternal.',
    or: 'atau',
    demo: 'Putar demo terarah · 11 detik',
    demoNote: 'Simulasi lokal tanpa mikrofon atau API eksternal.',
    demoBanner: 'Simulasi lokal · alur dipercepat',
    reportFailed: (message: string) => `Laporan gagal dimuat: ${message}`,
    preparingReport: 'Menyiapkan laporan berbukti…',
    newSession: 'Mulai sesi baru',
    demoActive: 'Demo aktif',
    connected: 'Terhubung',
    connecting: 'Menyambung…',
    noMic: 'tanpa mikrofon',
    recording: 'merekam',
    micOff: 'mic mati',
    micProcessing: {
      noiseSuppression: 'peredam bising',
      voiceIsolation: 'isolasi suara',
      echoCancellation: 'peredam gema',
    },
    audioHeld: 'penilaian ditahan sampai audio membaik',
    scoringUnavailable:
      'Penilai AI tidak bisa dipakai, jadi kewajiban tidak akan terpenuhi. Periksa akses LLM Gateway di akun AssemblyAI.',
    excluded: (count: number) =>
      `${count} ucapan tidak dihitung sebagai bukti — suara tidak dikenali atau audio tidak layak. Ulangi bagian itu.`,
    obligationDetails: 'Rincian kewajiban',
    transcript: 'Transkrip',
    end: 'Akhiri sesi',
    ending: 'Mengakhiri sesi…',
  },
  focus: {
    loading: 'Memuat kewajiban',
    preparing: 'Menyiapkan…',
    pending: 'Belum disampaikan',
    allDone: 'Semua kewajiban terpenuhi',
    complete: 'Lengkap',
    count: (done: number, total: number) => `${done} dari ${total}`,
  },
  /** Kunci = kode kewajiban dari backend; kode tak dikenal memakai label backend. */
  obligations: {
    label: {
      IDENTITY: 'Identitas & lembaga',
      RATE: 'Suku bunga / biaya',
      TENOR: 'Jangka waktu & cicilan',
      PENALTY: 'Denda keterlambatan',
      RIGHT: 'Hak membatalkan',
    } as Record<string, string>,
    hint: {
      IDENTITY: 'Sebutkan nama kamu dan nama lembaga tempatmu bekerja.',
      RATE: 'Sebutkan suku bunga atau total biaya yang harus dibayar.',
      TENOR: 'Sebutkan jangka waktu dan besar cicilan per bulan.',
      PENALTY: 'Jelaskan denda kalau nasabah telat membayar.',
      RIGHT: 'Beri tahu nasabah berhak menolak atau membatalkan.',
    } as Record<string, string>,
  },
  calibration: {
    eyebrow: 'Langkah keamanan · belum dinilai',
    step: (step: number) => ` · langkah ${step} dari 2`,
    title: 'Kenali dua suara',
    officerPrompt: 'ucapkan kalimat ini dengan jelas:',
    // Isi kalimat kalibrasi tidak dinilai atau dicocokkan backend — yang
    // dipelajari hanya suaranya — jadi aman mengikuti bahasa antarmuka.
    // Bahasa juga dideteksi per ucapan, bukan dikunci per sesi (diuji
    // 28 Sep 2026), sehingga kalimat berbahasa Inggris tidak mengganggu
    // transkripsi Bahasa Indonesia sesudahnya.
    officerScript: '“Saya petugas yang menjalankan sesi ini.”',
    listeningOfficer: 'Mendengarkan suara petugas…',
    officerRegistered: 'Petugas terdaftar',
    voice: (label: string) => `Suara ${label}`,
    customerPromptLead: 'Sekarang giliran',
    customerWord: 'nasabah',
    customerPromptTail: '. Ucapkan:',
    customerScript: '“Saya nasabah dan siap memulai.”',
    duplicateLead: 'Suara itu sudah terdaftar sebagai petugas. Minta',
    duplicateStrong: 'orang kedua',
    duplicateTail:
      'yang berbicara — sistem perlu mendengar suara yang berbeda untuk bisa membedakan keduanya.',
    listeningCustomer: 'Mendengarkan suara nasabah…',
    customerRegistered: 'Nasabah terdaftar',
    failed: (message: string) => `Kalibrasi gagal: ${message}`,
    confirm: 'Konfirmasi dan mulai penilaian',
    waiting: 'Menunggu dua suara',
    restart: 'Ulangi kalibrasi dari awal',
    note: 'Ucapan kalibrasi tidak masuk laporan dan tidak dinilai.',
  },
  transcript: {
    corrected: 'label dikoreksi',
    revised: 'label direvisi',
  },
  report: {
    simulated: 'Simulasi lokal · bukan transkripsi AssemblyAI live',
    eyebrow: 'Laporan kepatuhan',
    session: (id: string) => `Sesi ${id}`,
    meta: (officer: string, product: string) => `Petugas ${officer} · Produk ${product}`,
    summary: 'Ringkasan laporan',
    obligations: 'Kewajiban',
    violations: 'Pelanggaran',
    evidenceUtterances: 'Ucapan berbukti',
    evidenceTitle: 'Bukti per kewajiban',
    confidence: (percent: number) => `${percent}% confidence`,
    notSatisfied: 'Belum terpenuhi',
    noQuote: 'Tidak ada kutipan yang memenuhi evidence gate.',
    clean: '✓ Tidak ada janji terlarang yang terdeteksi.',
    transcriptTitle: 'Transkrip berbukti',
  },
  supervisor: {
    recent: 'Sesi terbaru',
    loadingList: 'Memuat daftar sesi…',
    empty: 'Belum ada sesi. Mulai satu dari halaman petugas, lalu segarkan.',
    loadingSession: 'Memuat sesi…',
    active: 'berjalan',
    ended: 'selesai',
    refresh: 'Segarkan daftar',
    back: '← Kembali ke daftar sesi',
    obligations: 'Kewajiban',
    live: 'Live',
    disconnected: 'Terputus',
    violations: 'Pelanggaran',
    none: 'Belum ada.',
    transcript: 'Transkrip',
  },
  severity: { high: 'tinggi' } as Record<string, string>,
  /** Bisikan yang TAMPIL di layar. Yang terdengar selalu teks dari backend. */
  nudge: {
    avoidPhrase: (phrase: string) => `Hati-hati, hindari frasa "${phrase}".`,
    pending: (label: string) => `Belum disampaikan: ${label}`,
  },
  demo: {
    correction: 'Hindari janji pasti disetujui. Koreksi dan jelaskan bunga serta cicilan.',
    reminder: 'Sampaikan denda keterlambatan dan hak nasabah untuk membatalkan.',
  },
  /**
   * Alasan audio tidak layak. Kliennya mengirim kalimat Bahasa Indonesia ke
   * gateway dan gateway menyiarkannya kembali apa adanya, jadi terjemahannya
   * dicari dari kalimat itu — lihat `localizeAudioReason`.
   */
  audio: {
    tooLoud: 'Suara terlalu keras dan pecah',
    tooQuiet: 'Suara terlalu pelan dari mikrofon',
    noisy: 'Kebisingan latar terlalu tinggi — matikan musik atau pindah ke tempat lebih tenang',
    echo: 'terdengar seperti gema bisikan aplikasi — pakai earphone',
  },
  errors: {
    invalidCredentials: 'Nama pengguna atau kata sandi salah',
    loginFailed: (status: string) => `Gagal masuk (HTTP ${status})`,
    roleMismatch: (actual: string, expected: string) =>
      `Akun ini adalah ${actual}. Halaman ini untuk ${expected}.`,
    gatewayLost: 'Koneksi gateway terputus',
  },
}

export type Dictionary = typeof id

const en: Dictionary = {
  locale: 'en-US',
  brand: {
    tagline: 'Compliance copilot for field officers',
    supervisorTitle: 'Bisik Supervisor',
    supervisorTagline: 'Evidence-based compliance monitoring',
  },
  language: {
    label: 'Interface language',
    short: { id: 'ID', en: 'EN' },
    name: { id: 'Bahasa Indonesia', en: 'English' },
  },
  auth: {
    checkingSession: 'Checking your sign-in…',
    officerTitle: 'Sign in as officer',
    officerHint: 'Sessions and their reports are recorded under this account.',
    supervisorTitle: 'Sign in as supervisor',
    supervisorHint: 'Supervisors monitor and read reports; they never send audio.',
    username: 'Username',
    password: 'Password',
    checking: 'Checking…',
    submit: 'Sign in',
    signedInAs: 'Signed in as',
    logout: 'Sign out',
    logoutTitle: 'Sign out of your account?',
    logoutBody: 'You will need your username and password to sign in again.',
    cancel: 'Cancel',
  },
  roles: { officer: 'an officer', supervisor: 'a supervisor' },
  speakers: {
    officer: 'Officer',
    customer: 'Customer',
    unknown: '—',
    unknownLong: 'Unknown',
  },
  officer: {
    start: 'Start session',
    starting: 'Opening session…',
    gatewayDown:
      "The gateway can't be reached, so a real session can't start yet. The guided demo below still runs in full — it uses neither the backend nor any external API.",
    or: 'or',
    demo: 'Play guided demo · 11 seconds',
    demoNote: 'Local simulation, no microphone or external API.',
    demoBanner: 'Local simulation · accelerated flow',
    reportFailed: (message) => `Couldn't load the report: ${message}`,
    preparingReport: 'Preparing the evidence report…',
    newSession: 'Start a new session',
    demoActive: 'Demo running',
    connected: 'Connected',
    connecting: 'Connecting…',
    noMic: 'no microphone',
    recording: 'recording',
    micOff: 'mic off',
    micProcessing: {
      noiseSuppression: 'noise suppression',
      voiceIsolation: 'voice isolation',
      echoCancellation: 'echo cancellation',
    },
    audioHeld: 'scoring paused until the audio improves',
    scoringUnavailable:
      'The AI scorer is unavailable, so no obligation can be met. Check LLM Gateway access on the AssemblyAI account.',
    excluded: (count) =>
      `${count} ${count === 1 ? 'utterance was' : 'utterances were'} not counted as evidence — unrecognised voice or unusable audio. Repeat that part.`,
    obligationDetails: 'Obligation details',
    transcript: 'Transcript',
    end: 'End session',
    ending: 'Ending session…',
  },
  focus: {
    loading: 'Loading obligations',
    preparing: 'Preparing…',
    pending: 'Not yet disclosed',
    allDone: 'All obligations met',
    complete: 'Complete',
    count: (done, total) => `${done} of ${total}`,
  },
  obligations: {
    label: {
      IDENTITY: 'Identity & institution',
      RATE: 'Interest rate / fees',
      TENOR: 'Term & instalments',
      PENALTY: 'Late-payment penalty',
      RIGHT: 'Right to cancel',
    },
    hint: {
      IDENTITY: 'State your name and the institution you work for.',
      RATE: 'State the interest rate or the total cost to be paid.',
      TENOR: 'State the loan term and the monthly instalment.',
      PENALTY: 'Explain the penalty if the customer pays late.',
      RIGHT: 'Tell the customer they have the right to refuse or cancel.',
    },
  },
  calibration: {
    eyebrow: 'Safety step · not scored',
    step: (step) => ` · step ${step} of 2`,
    title: 'Learn the two voices',
    officerPrompt: 'say this sentence clearly:',
    officerScript: '“I am the officer running this session.”',
    listeningOfficer: 'Listening for the officer…',
    officerRegistered: 'Officer registered',
    voice: (label) => `Voice ${label}`,
    customerPromptLead: 'Now it’s the',
    customerWord: 'customer',
    customerPromptTail: '’s turn. Say:',
    customerScript: '“I am the customer and I am ready to begin.”',
    duplicateLead: 'That voice is already registered as the officer. Ask the',
    duplicateStrong: 'second person',
    duplicateTail: 'to speak — the system needs to hear a different voice to tell them apart.',
    listeningCustomer: 'Listening for the customer…',
    customerRegistered: 'Customer registered',
    failed: (message) => `Calibration failed: ${message}`,
    confirm: 'Confirm and start scoring',
    waiting: 'Waiting for two voices',
    restart: 'Restart calibration',
    note: 'Calibration speech is not included in the report and is not scored.',
  },
  transcript: {
    corrected: 'label corrected',
    revised: 'label revised',
  },
  report: {
    simulated: 'Local simulation · not a live AssemblyAI transcription',
    eyebrow: 'Compliance report',
    session: (id) => `Session ${id}`,
    meta: (officer, product) => `Officer ${officer} · Product ${product}`,
    summary: 'Report summary',
    obligations: 'Obligations',
    violations: 'Violations',
    evidenceUtterances: 'Evidence utterances',
    evidenceTitle: 'Evidence per obligation',
    confidence: (percent) => `${percent}% confidence`,
    notSatisfied: 'Not met',
    noQuote: 'No quote passed the evidence gate.',
    clean: '✓ No prohibited promises detected.',
    transcriptTitle: 'Evidence transcript',
  },
  supervisor: {
    recent: 'Recent sessions',
    loadingList: 'Loading sessions…',
    empty: 'No sessions yet. Start one from the officer page, then refresh.',
    loadingSession: 'Loading session…',
    active: 'in progress',
    ended: 'ended',
    refresh: 'Refresh list',
    back: '← Back to sessions',
    obligations: 'Obligations',
    live: 'Live',
    disconnected: 'Disconnected',
    violations: 'Violations',
    none: 'None yet.',
    transcript: 'Transcript',
  },
  severity: { high: 'high' },
  nudge: {
    avoidPhrase: (phrase) => `Careful — avoid the phrase "${phrase}".`,
    pending: (label) => `Not yet disclosed: ${label}`,
  },
  demo: {
    correction: 'Avoid promising guaranteed approval. Correct it and explain the rate and instalments.',
    reminder: 'Disclose the late-payment penalty and the customer’s right to cancel.',
  },
  audio: {
    tooLoud: 'Audio is too loud and distorting',
    tooQuiet: 'Audio is too quiet at the microphone',
    noisy: 'Background noise is too high — turn off music or move somewhere quieter',
    echo: 'sounds like an echo of the app’s own whisper — use earphones',
  },
  errors: {
    invalidCredentials: 'Wrong username or password',
    loginFailed: (status) => `Sign-in failed (HTTP ${status})`,
    roleMismatch: (actual, expected) =>
      `This account belongs to ${actual}. This page is for ${expected}.`,
    gatewayLost: 'Lost connection to the gateway',
  },
}

export const dictionaries: Record<Lang, Dictionary> = { id, en }
