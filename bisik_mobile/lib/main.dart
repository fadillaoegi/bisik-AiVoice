import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'core/theme/bisik_theme.dart';
import 'features/auth/presentation/pages/login_page.dart';
import 'features/auth/presentation/providers/auth_providers.dart';
import 'features/session/presentation/pages/officer_page.dart';

void main() {
  runApp(const ProviderScope(child: BisikApp()));
}

class BisikApp extends StatelessWidget {
  const BisikApp({super.key});

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: 'Bisik',
      debugShowCheckedModeBanner: false,
      theme: bisikTheme(),
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
