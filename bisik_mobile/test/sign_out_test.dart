import 'package:bisik_mobile/core/i18n/app_strings.dart';
import 'package:bisik_mobile/core/i18n/language.dart';
import 'package:bisik_mobile/features/auth/domain/entities/auth_user.dart';
import 'package:bisik_mobile/features/auth/presentation/providers/auth_providers.dart';
import 'package:bisik_mobile/features/session/presentation/pages/officer_page.dart';
import 'package:bisik_mobile/features/session/presentation/providers/session_controller.dart';
import 'package:bisik_mobile/features/session/presentation/providers/session_state.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

/// Sudah masuk sebagai Handoko, tanpa memanggil server.
class _SignedIn extends AuthController {
  @override
  AuthState build() => const AuthState(
        user: AuthUser(id: 'u-1', name: 'Handoko', role: Role.officer),
      );
}

/// Tidak memuat apa pun dari jaringan; hanya mencatat pembersihan layar.
class _QuietSession extends SessionController {
  int cleared = 0;

  @override
  SessionState build() => const SessionState();

  @override
  Future<void> loadObligations() async {}

  @override
  void clear() {
    cleared++;
    super.clear();
  }
}

void main() {
  late _QuietSession session;
  late ProviderContainer container;

  Future<void> pump(WidgetTester tester, AppLanguage language) async {
    session = _QuietSession();
    container = ProviderContainer(overrides: [
      authControllerProvider.overrideWith(_SignedIn.new),
      sessionControllerProvider.overrideWith(() => session),
    ]);
    addTearDown(container.dispose);
    await tester.pumpWidget(
      UncontrolledProviderScope(
        container: container,
        child: MaterialApp(
          builder: (context, child) => StringsScope(
            strings: AppStrings.of(language),
            child: child!,
          ),
          home: const OfficerPage(),
        ),
      ),
    );
    await tester.pump();
  }

  bool signedIn() => container.read(authControllerProvider).isLoggedIn;
  Finder inDialog(String text) =>
      find.descendant(of: find.byType(AlertDialog), matching: find.text(text));

  testWidgets('Batal: dialog tertutup dan tetap masuk', (tester) async {
    await pump(tester, AppLanguage.id);
    await tester.tap(find.text('Keluar'));
    await tester.pumpAndSettle();
    expect(find.text('Keluar dari akun?'), findsOneWidget);

    await tester.tap(inDialog('Batal'));
    await tester.pumpAndSettle();
    expect(find.byType(AlertDialog), findsNothing);
    expect(signedIn(), isTrue);
    expect(session.cleared, 0);
  });

  testWidgets('ketuk di luar dialog sama dengan Batal', (tester) async {
    await pump(tester, AppLanguage.id);
    await tester.tap(find.text('Keluar'));
    await tester.pumpAndSettle();
    await tester.tapAt(const Offset(5, 5));
    await tester.pumpAndSettle();
    expect(find.byType(AlertDialog), findsNothing);
    expect(signedIn(), isTrue);
  });

  testWidgets('konfirmasi: keluar dan layar dibersihkan', (tester) async {
    await pump(tester, AppLanguage.id);
    await tester.tap(find.text('Keluar'));
    await tester.pumpAndSettle();
    await tester.tap(inDialog('Keluar'));
    await tester.pumpAndSettle();
    expect(signedIn(), isFalse);
    expect(session.cleared, 1);
  });

  testWidgets('dialog mengikuti bahasa antarmuka', (tester) async {
    await pump(tester, AppLanguage.en);
    await tester.tap(find.text('Sign out'));
    await tester.pumpAndSettle();
    expect(find.text('Sign out of your account?'), findsOneWidget);
    expect(inDialog('Cancel'), findsOneWidget);
  });
}
