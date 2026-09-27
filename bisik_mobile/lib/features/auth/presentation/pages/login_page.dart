import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/theme/bisik_theme.dart';
import '../../../../core/theme/bisik_wave.dart';
import '../providers/auth_providers.dart';

class LoginPage extends ConsumerStatefulWidget {
  const LoginPage({super.key});

  @override
  ConsumerState<LoginPage> createState() => _LoginPageState();
}

class _LoginPageState extends ConsumerState<LoginPage> {
  final _username = TextEditingController();
  final _password = TextEditingController();

  @override
  void dispose() {
    _username.dispose();
    _password.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final auth = ref.watch(authControllerProvider);
    final controller = ref.read(authControllerProvider.notifier);

    return Scaffold(
      body: SafeArea(
        child: Padding(
          padding: const EdgeInsets.all(24),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            mainAxisAlignment: MainAxisAlignment.center,
            children: [
              const Text(
                'Bisik',
                style: TextStyle(
                  color: BisikColors.accent,
                  fontSize: 32,
                  fontWeight: FontWeight.w700,
                ),
              ),
              const SizedBox(height: 6),
              const Text(
                'Masuk sebagai petugas. Sesi dan laporannya tercatat atas '
                'nama akun ini.',
                style: TextStyle(color: BisikColors.muted, fontSize: 14),
              ),
              const SizedBox(height: 28),

              const Text('Nama pengguna',
                  style: TextStyle(color: BisikColors.muted)),
              const SizedBox(height: 8),
              TextField(
                controller: _username,
                enabled: !auth.busy,
                autocorrect: false,
                style: const TextStyle(color: BisikColors.text),
              ),
              const SizedBox(height: 16),

              const Text('Kata sandi',
                  style: TextStyle(color: BisikColors.muted)),
              const SizedBox(height: 8),
              TextField(
                controller: _password,
                enabled: !auth.busy,
                obscureText: true,
                style: const TextStyle(color: BisikColors.text),
                onSubmitted: (_) =>
                    controller.login(_username.text, _password.text),
              ),
              const SizedBox(height: 24),

              FilledButton(
                onPressed: auth.busy
                    ? null
                    : () => controller.login(_username.text, _password.text),
                child: Padding(
                  padding: const EdgeInsets.all(12),
                  child: auth.busy
                      ? const Row(
                          mainAxisAlignment: MainAxisAlignment.center,
                          children: [
                            BisikWave(height: 18, color: BisikColors.accentInk),
                            SizedBox(width: 12),
                            Text('Memeriksa…'),
                          ],
                        )
                      : const Text('Masuk'),
                ),
              ),

              if (auth.error != null) ...[
                const SizedBox(height: 16),
                Container(
                  padding: const EdgeInsets.all(12),
                  decoration: BoxDecoration(
                    color: BisikColors.bad.withValues(alpha: 0.12),
                    border: Border.all(color: BisikColors.bad),
                    borderRadius: BorderRadius.circular(8),
                  ),
                  child: Text(
                    auth.error!,
                    style: const TextStyle(color: BisikColors.bad),
                  ),
                ),
              ],
            ],
          ),
        ),
      ),
    );
  }
}
