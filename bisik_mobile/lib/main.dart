import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:shared_preferences/shared_preferences.dart';

import 'core/i18n/app_strings.dart';
import 'core/i18n/language.dart';
import 'core/theme/bisik_theme.dart';
import 'features/auth/presentation/pages/login_page.dart';
import 'features/auth/presentation/providers/auth_providers.dart';
import 'features/session/presentation/pages/officer_page.dart';

Future<void> main() async {
  WidgetsFlutterBinding.ensureInitialized();

  // Pilihan bahasa dibaca sebelum frame pertama. Kalau penyimpanan perangkat
  // gagal dibuka, jangan gagalkan aplikasi — cukup pakai bahasa bawaan.
  SharedPreferences? prefs;
  try {
    prefs = await SharedPreferences.getInstance();
  } catch (_) {
    prefs = null;
  }

  runApp(
    ProviderScope(
      overrides: [sharedPreferencesProvider.overrideWithValue(prefs)],
      child: const BisikApp(),
    ),
  );
}

class BisikApp extends ConsumerWidget {
  const BisikApp({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final strings = AppStrings.of(ref.watch(languageProvider));
    return MaterialApp(
      title: 'Bisik',
      debugShowCheckedModeBanner: false,
      theme: bisikTheme(),
      builder: (context, child) =>
          StringsScope(strings: strings, child: child ?? const SizedBox()),
      home: const _Gate(),
    );
  }
}

/// Gerbang login. Token hanya hidup di memori, jadi petugas masuk sekali
/// setiap aplikasi dibuka — lihat catatan di `core/network/auth_token.dart`.
class _Gate extends ConsumerWidget {
  const _Gate();

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final loggedIn = ref.watch(
      authControllerProvider.select((state) => state.isLoggedIn),
    );
    return loggedIn ? const OfficerPage() : const LoginPage();
  }
}
