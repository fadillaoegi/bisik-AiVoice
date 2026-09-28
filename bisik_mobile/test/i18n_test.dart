import 'package:bisik_mobile/core/audio/audio_quality.dart';
import 'package:bisik_mobile/core/error/failure.dart';
import 'package:bisik_mobile/core/i18n/app_strings.dart';
import 'package:bisik_mobile/core/i18n/language.dart';
import 'package:bisik_mobile/core/i18n/language_switch.dart';
import 'package:bisik_mobile/features/session/domain/entities/compliance.dart';
import 'package:bisik_mobile/features/session/domain/entities/session_event.dart';
import 'package:bisik_mobile/features/session/presentation/session_strings.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:shared_preferences/shared_preferences.dart';

void main() {
  const id = AppStrings.id;
  const en = AppStrings.en;

  group('bisikan', () {
    // Kalimat Bahasa Indonesia yang TAMPIL harus identik dengan yang disusun
    // backend (`inspectGuardrail`, `RemindPending`) — itulah yang terdengar.
    test('teks Indonesia sama persis dengan kalimat backend', () {
      const avoid = Nudge(
        text: 'Hati-hati, hindari frasa "pasti cair".',
        kind: NudgeKind.avoidPhrase,
        phrase: 'pasti cair',
      );
      const pending = Nudge(
        text: 'Belum disampaikan: Denda keterlambatan',
        kind: NudgeKind.pendingObligation,
        code: 'PENALTY',
      );
      expect(id.nudge(avoid), avoid.text);
      expect(id.nudge(pending), pending.text);
    });

    test('antarmuka Inggris menyusun ulang dari kind', () {
      expect(
        en.nudge(const Nudge(
          text: 'x',
          kind: NudgeKind.pendingObligation,
          code: 'RATE',
        )),
        'Not yet disclosed: Interest rate / fees',
      );
      expect(
        en.nudge(const Nudge(
          text: 'x',
          kind: NudgeKind.avoidPhrase,
          phrase: 'pasti cair',
        )),
        'Careful — avoid the phrase "pasti cair".',
      );
    });

    test('gateway lama tanpa kind menampilkan teks apa adanya', () {
      expect(en.nudge(const Nudge(text: 'teks lama')), 'teks lama');
    });
  });

  group('kewajiban', () {
    test('lima kode punya label dan petunjuk di dua bahasa', () {
      for (final code in ['IDENTITY', 'RATE', 'TENOR', 'PENALTY', 'RIGHT']) {
        for (final s in [id, en]) {
          expect(s.obligationLabel(code), isNotNull, reason: code);
          expect(s.obligationHint(code), isNotEmpty, reason: code);
        }
      }
    });

    test('kode baru dari backend memakai label backend', () {
      const o = Obligation(code: 'BARU', label: 'Label backend');
      expect(en.obligation(o), 'Label backend');
    });
  });

  group('audio & peringatan', () {
    test('alasan audio: kalimat kawat Indonesia, tampilan diterjemahkan', () {
      expect(id.audioReason(AudioReasons.noisy), AudioReasons.noisy);
      expect(
        en.audioReason(AudioReasons.tooQuiet),
        'Audio is too quiet at the microphone',
      );
      expect(en.audioReason('alasan baru'), 'alasan baru');
    });

    test('peringatan disusun dari jenis', () {
      expect(
        en.warning(const SessionWarning(
          SessionWarningKind.audioDegraded,
          AudioReasons.tooLoud,
        )),
        'Audio is too loud and distorting — scoring paused',
      );
      expect(
        id.warning(const SessionWarning(SessionWarningKind.evidenceSkipped)),
        'Ucapan dilewati: audio tidak layak',
      );
    });
  });

  group('kegagalan', () {
    const unreachable = NetworkFailure(
      FailureKind.gatewayUnreachable,
      action: FailureAction.startSession,
      detail: 'http://10.0.2.2:8080',
    );

    test('diterjemahkan menurut jenis dan aksinya', () {
      expect(id.describe(unreachable),
          startsWith('Gagal memulai sesi: gateway http://10.0.2.2:8080'));
      expect(en.describe(unreachable),
          startsWith('Couldn’t start the session: gateway http://10.0.2.2:8080'));
    });

    test('log tetap Bahasa Indonesia', () {
      expect('$unreachable', id.failure(unreachable));
    });

    test('error yang bukan Failure tampil apa adanya', () {
      expect(en.describe('pesan dari gateway'), 'pesan dari gateway');
    });
  });

  group('pengganti bahasa', () {
    Future<SharedPreferences> pump(
      WidgetTester tester,
      Map<String, Object> stored,
    ) async {
      SharedPreferences.setMockInitialValues(stored);
      final prefs = await SharedPreferences.getInstance();
      await tester.pumpWidget(
        ProviderScope(
          overrides: [sharedPreferencesProvider.overrideWithValue(prefs)],
          child: Consumer(
            builder: (context, ref, _) => MaterialApp(
              builder: (context, child) => StringsScope(
                strings: AppStrings.of(ref.watch(languageProvider)),
                child: child!,
              ),
              home: Scaffold(
                body: Builder(
                  builder: (context) => Column(
                    children: [
                      const LanguageSwitch(),
                      Text(context.s.startSession),
                    ],
                  ),
                ),
              ),
            ),
          ),
        ),
      );
      return prefs;
    }

    testWidgets('bawaan Bahasa Indonesia', (tester) async {
      await pump(tester, {});
      expect(find.text('Mulai sesi'), findsOneWidget);
    });

    testWidgets('ketuk EN mengganti teks dan menyimpan pilihan',
        (tester) async {
      final prefs = await pump(tester, {});
      await tester.tap(find.text('EN'));
      await tester.pumpAndSettle();
      expect(find.text('Start session'), findsOneWidget);
      expect(prefs.getString('bisik.lang'), 'en');
    });

    testWidgets('pilihan tersimpan dipakai sejak frame pertama',
        (tester) async {
      await pump(tester, {'bisik.lang': 'en'});
      expect(find.text('Start session'), findsOneWidget);
    });

    testWidgets('nilai tersimpan yang rusak jatuh ke bawaan', (tester) async {
      await pump(tester, {'bisik.lang': 'fr'});
      expect(find.text('Mulai sesi'), findsOneWidget);
    });
  });
}
