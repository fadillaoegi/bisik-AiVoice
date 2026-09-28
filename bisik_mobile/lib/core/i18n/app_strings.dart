import '../audio/audio_quality.dart';
import '../error/failure.dart';

enum AppLanguage { id, en }

/// Kamus antarmuka — kembaran `presentation/i18n/dictionary.ts` di PWA.
///
/// Yang diterjemahkan hanya antarmuka. Isi percakapan, frasa terlarang, dan
/// SUARA bisikan di earpiece tetap Bahasa Indonesia: percakapannya memang
/// berbahasa Indonesia, dan echo guard di backend mencocokkan bisikan dalam
/// bahasa itu. Bisikan berbahasa Inggris bisa lolos dari penjagaan gema dan
/// memenuhi checklist dengan suara aplikasi sendiri.
///
/// Kelas abstrak, bukan Map: kalimat yang lupa diterjemahkan gagal saat
/// kompilasi, bukan tampil kosong di depan juri.
abstract class AppStrings {
  const AppStrings();

  static const AppStrings id = _IdStrings();
  static const AppStrings en = _EnStrings();

  static AppStrings of(AppLanguage language) => switch (language) {
    AppLanguage.id => id,
    AppLanguage.en => en,
  };

  // ── Bahasa ──────────────────────────────────────────────────────────
  String get languageLabel;
  String languageName(AppLanguage language) => switch (language) {
    AppLanguage.id => 'Bahasa Indonesia',
    AppLanguage.en => 'English',
  };
  String languageShort(AppLanguage language) => language.name.toUpperCase();

  // ── Login ───────────────────────────────────────────────────────────
  String get loginHint;
  String get username;
  String get password;
  String get checking;
  String get signIn;
  String get signOut;
  String signedInAs(String name);

  // ── Sesi ────────────────────────────────────────────────────────────
  String get startSession;
  String get openingSession;
  String get connected;
  String get connecting;
  String get recording;
  String get micOff;
  String get obligationDetails;
  String get transcript;
  String get endSession;
  String get preparingReport;

  // ── Pembicara ───────────────────────────────────────────────────────
  String get officer;
  String get customer;
  String get unknownSpeaker => '—';
  String get labelRevised;

  // ── Kalibrasi ───────────────────────────────────────────────────────
  String calibrationEyebrow(int? step);
  String get calibrationTitle;

  /// Kosong di Bahasa Indonesia. Di bahasa lain menjelaskan kenapa kalimat
  /// kalibrasi tetap Bahasa Indonesia: transkripsi seluruh sesi dikunci ke
  /// bahasa yang pertama terdengar.
  String get calibrationScriptNote;
  String get officerPromptTrailing;

  // Kalimat kalibrasi SENGAJA sama di semua bahasa antarmuka.
  String get officerScript => '“Saya petugas yang menjalankan sesi ini.”';
  String get customerScript => '“Saya nasabah dan siap memulai.”';
  String get listeningOfficer;
  String get officerRegistered;
  String get customerPromptLead;
  String get customerPromptTrailing;
  String get duplicateVoice;
  String get listeningCustomer;
  String get customerRegistered;
  String voice(String label);
  String calibrationFailed(String? message);
  String get confirmRoles;
  String get waitingForVoices;
  String get restartCalibration;
  String get calibrationNote;

  // ── Kartu fokus ─────────────────────────────────────────────────────
  String get focusLoading;
  String get focusPreparing;
  String get focusPending;
  String get focusAllDone;
  String get focusComplete;
  String focusCount(int done, int total);

  /// Label kewajiban menurut kodenya; null = kode baru, pakai label backend.
  String? obligationLabel(String code);
  String obligationHint(String code);

  // ── Laporan ─────────────────────────────────────────────────────────
  String get reportEyebrow;
  String get statObligations;
  String get statViolations;
  String get evidenceTitle;
  String get violationsTitle;
  String get transcriptTitle;
  String get emptyTranscript;
  String get neverDisclosed;
  String get newSession;
  String severity(String raw);

  // ── Bisikan (yang TAMPIL; yang terdengar selalu teks backend) ────────
  String nudgeAvoidPhrase(String phrase);
  String nudgePending(String label);

  // ── Peringatan & error ──────────────────────────────────────────────
  String get warningUnknownSpeaker;
  String warningEvidenceSkipped(String? reason);
  String warningAudioDegraded(String? reason);

  /// Alasan audio dari gateway selalu Bahasa Indonesia; terjemahkan yang
  /// dikenali, sisanya tampil apa adanya.
  String audioReason(String raw);

  String failure(Failure f);

  /// Error apa pun menjadi kalimat: [Failure] diterjemahkan, sisanya apa adanya.
  String describe(Object error) => error is Failure ? failure(error) : '$error';
}

class _IdStrings extends AppStrings {
  const _IdStrings();

  @override
  String get languageLabel => 'Bahasa antarmuka';

  @override
  String get loginHint =>
      'Masuk sebagai petugas. Sesi dan laporannya tercatat atas nama akun ini.';
  @override
  String get username => 'Nama pengguna';
  @override
  String get password => 'Kata sandi';
  @override
  String get checking => 'Memeriksa…';
  @override
  String get signIn => 'Masuk';
  @override
  String get signOut => 'Keluar';
  @override
  String signedInAs(String name) => 'Masuk sebagai $name';

  @override
  String get startSession => 'Mulai sesi';
  @override
  String get openingSession => 'Membuka sesi…';
  @override
  String get connected => 'Terhubung';
  @override
  String get connecting => 'Menyambung…';
  @override
  String get recording => 'merekam';
  @override
  String get micOff => 'mic mati';
  @override
  String get obligationDetails => 'Rincian kewajiban';
  @override
  String get transcript => 'Transkrip';
  @override
  String get endSession => 'Akhiri sesi';
  @override
  String get preparingReport => 'Menyiapkan laporan berbukti…';

  @override
  String get officer => 'Petugas';
  @override
  String get customer => 'Nasabah';
  @override
  String get labelRevised => '(label direvisi)';

  @override
  String calibrationEyebrow(int? step) => step == null
      ? 'LANGKAH KEAMANAN · BELUM DINILAI'
      : 'LANGKAH KEAMANAN · BELUM DINILAI · LANGKAH $step DARI 2';
  @override
  String get calibrationTitle => 'Kenali dua suara';
  @override
  String get calibrationScriptNote => '';
  @override
  String get officerPromptTrailing => ', ucapkan kalimat ini:';
  @override
  String get listeningOfficer => 'Mendengarkan suara petugas…';
  @override
  String get officerRegistered => 'Petugas terdaftar';
  @override
  String get customerPromptLead => 'Sekarang giliran nasabah';
  @override
  String get customerPromptTrailing => '. Ucapkan:';
  @override
  String get duplicateVoice =>
      'Suara itu sudah terdaftar sebagai petugas. Minta orang kedua '
      'yang berbicara — sistem perlu mendengar suara yang berbeda '
      'untuk bisa membedakan keduanya.';
  @override
  String get listeningCustomer => 'Mendengarkan suara nasabah…';
  @override
  String get customerRegistered => 'Nasabah terdaftar';
  @override
  String voice(String label) => 'Suara $label';
  @override
  String calibrationFailed(String? message) =>
      'Kalibrasi gagal: ${message ?? 'alasan tidak diketahui'}';
  @override
  String get confirmRoles => 'Konfirmasi dan mulai penilaian';
  @override
  String get waitingForVoices => 'Menunggu dua suara';
  @override
  String get restartCalibration => 'Ulangi kalibrasi dari awal';
  @override
  String get calibrationNote =>
      'Ucapan kalibrasi tidak masuk laporan dan tidak dinilai.';

  @override
  String get focusLoading => 'MEMUAT KEWAJIBAN';
  @override
  String get focusPreparing => 'Menyiapkan…';
  @override
  String get focusPending => 'BELUM DISAMPAIKAN';
  @override
  String get focusAllDone => 'SEMUA KEWAJIBAN TERPENUHI';
  @override
  String get focusComplete => 'Lengkap';
  @override
  String focusCount(int done, int total) => '$done dari $total';

  @override
  String? obligationLabel(String code) => switch (code) {
    'IDENTITY' => 'Identitas & lembaga',
    'RATE' => 'Suku bunga / biaya',
    'TENOR' => 'Jangka waktu & cicilan',
    'PENALTY' => 'Denda keterlambatan',
    'RIGHT' => 'Hak membatalkan',
    _ => null,
  };
  @override
  String obligationHint(String code) => switch (code) {
    'IDENTITY' => 'Sebutkan nama kamu dan nama lembaga tempatmu bekerja.',
    'RATE' => 'Sebutkan suku bunga atau total biaya yang harus dibayar.',
    'TENOR' => 'Sebutkan jangka waktu dan besar cicilan per bulan.',
    'PENALTY' => 'Jelaskan denda kalau nasabah telat membayar.',
    'RIGHT' => 'Beri tahu nasabah berhak menolak atau membatalkan.',
    _ => '',
  };

  @override
  String get reportEyebrow => 'LAPORAN KEPATUHAN';
  @override
  String get statObligations => 'kewajiban';
  @override
  String get statViolations => 'pelanggaran';
  @override
  String get evidenceTitle => 'Bukti per kewajiban';
  @override
  String get violationsTitle => 'Pelanggaran';
  @override
  String get transcriptTitle => 'Transkrip';
  @override
  String get emptyTranscript => 'Tidak ada ucapan yang tersimpan.';
  @override
  String get neverDisclosed => 'Tidak pernah disampaikan.';
  @override
  String get newSession => 'Mulai sesi baru';
  @override
  String severity(String raw) => raw == 'high' ? 'tinggi' : raw;

  // Dua kalimat ini HARUS sama persis dengan yang disusun backend
  // (`inspectGuardrail` dan `RemindPending`) — itulah yang terdengar.
  @override
  String nudgeAvoidPhrase(String phrase) =>
      'Hati-hati, hindari frasa "$phrase".';
  @override
  String nudgePending(String label) => 'Belum disampaikan: $label';

  @override
  String get warningUnknownSpeaker =>
      'Ada ucapan dari suara yang tidak dikenali — tidak dihitung sebagai bukti';
  @override
  String warningEvidenceSkipped(String? reason) =>
      'Ucapan dilewati: ${reason == null ? 'audio tidak layak' : audioReason(reason)}';
  @override
  String warningAudioDegraded(String? reason) =>
      '${reason == null ? 'Audio tidak layak' : audioReason(reason)} — penilaian ditahan';

  @override
  String audioReason(String raw) => raw;

  @override
  String failure(Failure f) {
    final action = switch (f.action) {
      FailureAction.login => 'masuk',
      FailureAction.startSession => 'memulai sesi',
      FailureAction.endSession => 'mengakhiri sesi',
      FailureAction.loadReport => 'memuat laporan',
      FailureAction.loadObligations => 'memuat kewajiban',
      null => 'memproses permintaan',
    };
    return switch (f.kind) {
      FailureKind.invalidCredentials => 'Nama pengguna atau kata sandi salah',
      FailureKind.gatewayUnreachable =>
        'Gagal $action: gateway ${f.detail} tidak bisa dihubungi. '
            'Pastikan backend berjalan. Di perangkat fisik atau simulator iOS, '
            'jalankan ulang dengan --dart-define=API_URL=http://<IP-LAN>:8080',
      FailureKind.gatewayTimeout =>
        'Gagal $action: gateway ${f.detail} tidak menjawab tepat waktu.',
      FailureKind.httpStatus =>
        'Gagal $action: server menjawab HTTP ${f.detail}.',
      FailureKind.requestFailed => 'Gagal $action: ${f.detail}',
      FailureKind.micPermissionDenied => 'Izin mikrofon ditolak',
      FailureKind.micUnavailable => 'Gagal membuka mikrofon: ${f.detail}',
      FailureKind.supervisorAccount =>
        'Akun supervisor dipantau lewat dashboard web, bukan aplikasi ini.',
      FailureKind.audioPathLost => 'Jalur audio terputus',
    };
  }
}

class _EnStrings extends AppStrings {
  const _EnStrings();

  @override
  String get languageLabel => 'Interface language';

  @override
  String get loginHint =>
      'Sign in as an officer. Sessions and their reports are recorded under this account.';
  @override
  String get username => 'Username';
  @override
  String get password => 'Password';
  @override
  String get checking => 'Checking…';
  @override
  String get signIn => 'Sign in';
  @override
  String get signOut => 'Sign out';
  @override
  String signedInAs(String name) => 'Signed in as $name';

  @override
  String get startSession => 'Start session';
  @override
  String get openingSession => 'Opening session…';
  @override
  String get connected => 'Connected';
  @override
  String get connecting => 'Connecting…';
  @override
  String get recording => 'recording';
  @override
  String get micOff => 'mic off';
  @override
  String get obligationDetails => 'Obligation details';
  @override
  String get transcript => 'Transcript';
  @override
  String get endSession => 'End session';
  @override
  String get preparingReport => 'Preparing the evidence report…';

  @override
  String get officer => 'Officer';
  @override
  String get customer => 'Customer';
  @override
  String get labelRevised => '(label revised)';

  @override
  String calibrationEyebrow(int? step) => step == null
      ? 'SAFETY STEP · NOT SCORED'
      : 'SAFETY STEP · NOT SCORED · STEP $step OF 2';
  @override
  String get calibrationTitle => 'Learn the two voices';
  @override
  String get calibrationScriptNote =>
      'Say it in Indonesian — the conversation is transcribed as Indonesian.';
  @override
  String get officerPromptTrailing => ', say this sentence:';
  @override
  String get listeningOfficer => 'Listening for the officer…';
  @override
  String get officerRegistered => 'Officer registered';
  @override
  String get customerPromptLead => 'Now it’s the customer';
  @override
  String get customerPromptTrailing => '’s turn. Say:';
  @override
  String get duplicateVoice =>
      'That voice is already registered as the officer. Ask the second '
      'person to speak — the system needs to hear a different voice to '
      'tell them apart.';
  @override
  String get listeningCustomer => 'Listening for the customer…';
  @override
  String get customerRegistered => 'Customer registered';
  @override
  String voice(String label) => 'Voice $label';
  @override
  String calibrationFailed(String? message) =>
      'Calibration failed: ${message ?? 'unknown reason'}';
  @override
  String get confirmRoles => 'Confirm and start scoring';
  @override
  String get waitingForVoices => 'Waiting for two voices';
  @override
  String get restartCalibration => 'Restart calibration';
  @override
  String get calibrationNote =>
      'Calibration speech is not included in the report and is not scored.';

  @override
  String get focusLoading => 'LOADING OBLIGATIONS';
  @override
  String get focusPreparing => 'Preparing…';
  @override
  String get focusPending => 'NOT YET DISCLOSED';
  @override
  String get focusAllDone => 'ALL OBLIGATIONS MET';
  @override
  String get focusComplete => 'Complete';
  @override
  String focusCount(int done, int total) => '$done of $total';

  @override
  String? obligationLabel(String code) => switch (code) {
    'IDENTITY' => 'Identity & institution',
    'RATE' => 'Interest rate / fees',
    'TENOR' => 'Term & instalments',
    'PENALTY' => 'Late-payment penalty',
    'RIGHT' => 'Right to cancel',
    _ => null,
  };
  @override
  String obligationHint(String code) => switch (code) {
    'IDENTITY' => 'State your name and the institution you work for.',
    'RATE' => 'State the interest rate or the total cost to be paid.',
    'TENOR' => 'State the loan term and the monthly instalment.',
    'PENALTY' => 'Explain the penalty if the customer pays late.',
    'RIGHT' => 'Tell the customer they have the right to refuse or cancel.',
    _ => '',
  };

  @override
  String get reportEyebrow => 'COMPLIANCE REPORT';
  @override
  String get statObligations => 'obligations';
  @override
  String get statViolations => 'violations';
  @override
  String get evidenceTitle => 'Evidence per obligation';
  @override
  String get violationsTitle => 'Violations';
  @override
  String get transcriptTitle => 'Transcript';
  @override
  String get emptyTranscript => 'No utterances were saved.';
  @override
  String get neverDisclosed => 'Never disclosed.';
  @override
  String get newSession => 'Start a new session';
  @override
  String severity(String raw) => raw;

  @override
  String nudgeAvoidPhrase(String phrase) =>
      'Careful — avoid the phrase "$phrase".';
  @override
  String nudgePending(String label) => 'Not yet disclosed: $label';

  @override
  String get warningUnknownSpeaker =>
      'An utterance came from an unrecognised voice — not counted as evidence';
  @override
  String warningEvidenceSkipped(String? reason) =>
      'Utterance skipped: ${reason == null ? 'unusable audio' : audioReason(reason)}';
  @override
  String warningAudioDegraded(String? reason) =>
      '${reason == null ? 'Unusable audio' : audioReason(reason)} — scoring paused';

  @override
  String audioReason(String raw) => switch (raw) {
    AudioReasons.tooLoud => 'Audio is too loud and distorting',
    AudioReasons.tooQuiet => 'Audio is too quiet at the microphone',
    AudioReasons.noisy =>
      'Background noise is too high — turn off music or move somewhere quieter',
    AudioReasons.echo =>
      'sounds like an echo of the app’s own whisper — use earphones',
    _ => raw,
  };

  @override
  String failure(Failure f) {
    final action = switch (f.action) {
      FailureAction.login => 'sign in',
      FailureAction.startSession => 'start the session',
      FailureAction.endSession => 'end the session',
      FailureAction.loadReport => 'load the report',
      FailureAction.loadObligations => 'load obligations',
      null => 'complete the request',
    };
    return switch (f.kind) {
      FailureKind.invalidCredentials => 'Wrong username or password',
      FailureKind.gatewayUnreachable =>
        'Couldn’t $action: gateway ${f.detail} is unreachable. '
            'Make sure the backend is running. On a physical device or the iOS '
            'simulator, relaunch with --dart-define=API_URL=http://<LAN-IP>:8080',
      FailureKind.gatewayTimeout =>
        'Couldn’t $action: gateway ${f.detail} did not respond in time.',
      FailureKind.httpStatus =>
        'Couldn’t $action: the server responded with HTTP ${f.detail}.',
      FailureKind.requestFailed => 'Couldn’t $action: ${f.detail}',
      FailureKind.micPermissionDenied => 'Microphone permission was denied',
      FailureKind.micUnavailable => 'Couldn’t open the microphone: ${f.detail}',
      FailureKind.supervisorAccount =>
        'Supervisor accounts monitor from the web dashboard, not this app.',
      FailureKind.audioPathLost => 'Audio path disconnected',
    };
  }
}
