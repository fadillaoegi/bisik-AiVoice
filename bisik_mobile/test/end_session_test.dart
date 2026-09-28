import 'dart:async';

import 'package:bisik_mobile/core/i18n/app_strings.dart';
import 'package:bisik_mobile/core/i18n/language.dart';
import 'package:bisik_mobile/core/theme/bisik_wave.dart';
import 'package:bisik_mobile/features/auth/domain/entities/auth_user.dart';
import 'package:bisik_mobile/features/auth/presentation/providers/auth_providers.dart';
import 'package:bisik_mobile/features/session/domain/entities/session.dart';
import 'package:bisik_mobile/features/session/presentation/pages/officer_page.dart';
import 'package:bisik_mobile/features/session/presentation/providers/session_controller.dart';
import 'package:bisik_mobile/features/session/presentation/providers/session_state.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

class _SignedIn extends AuthController {
  @override
  AuthState build() => const AuthState(
        user: AuthUser(id: 'u-1', name: 'Handoko', role: Role.officer),
      );
}

/// Sesi berjalan yang penutupannya ditahan sampai [release] dipanggil —
/// meniru gateway yang masih menunggu AssemblyAI menuntaskan ucapan terakhir.
class _SlowEnd extends SessionController {
  final gate = Completer<void>();
  int endCalls = 0;

  @override
  SessionState build() => SessionState(
        session: Session(
          id: 's-1',
          officerId: 'u-1',
          productId: 'KREDIT-MULTIGUNA',
          status: SessionStatus.active,
          startedAt: DateTime(2026, 9, 28),
        ),
        calibration: CalibrationStatus.confirmed,
      );

  @override
  Future<void> loadObligations() async {}

  @override
  Future<void> end() async {
    if (state.ending) return;
    endCalls++;
    state = state.copyWith(ending: true);
    await gate.future;
    state = state.copyWith(ending: false);
  }
}

void main() {
  testWidgets('Akhiri sesi: loading tampil dan ketukan kedua diabaikan',
      (tester) async {
    final session = _SlowEnd();
    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          authControllerProvider.overrideWith(_SignedIn.new),
          sessionControllerProvider.overrideWith(() => session),
        ],
        child: MaterialApp(
          builder: (context, child) =>
              StringsScope(strings: AppStrings.id, child: child!),
          home: const OfficerPage(),
        ),
      ),
    );
    await tester.pump();

    await tester.tap(find.text('Akhiri sesi'));
    await tester.pump();
    expect(find.text('Mengakhiri sesi…'), findsOneWidget);
    expect(
      find.descendant(
          of: find.byType(OutlinedButton), matching: find.byType(BisikWave)),
      findsOneWidget,
    );

    // Tombol nonaktif selama menunggu: ketukan kedua tidak memanggil apa pun.
    await tester.tap(find.text('Mengakhiri sesi…'));
    await tester.pump();
    expect(session.endCalls, 1);

    session.gate.complete();
    await tester.pump();
    expect(find.text('Akhiri sesi'), findsOneWidget);
    // Animasi gelombang berulang tanpa henti; hentikan agar test selesai.
    await tester.pumpWidget(const SizedBox());
  });
}
