import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/error/failure.dart';
import '../../../../core/network/auth_token.dart';
import '../../../session/presentation/providers/di_providers.dart';
import '../../data/auth_remote_datasource.dart';
import '../../domain/entities/auth_user.dart';

/// Satu instance token untuk seluruh aplikasi: dibaca interceptor Dio dan
/// datasource WebSocket, ditulis controller ini.
final authTokenProvider = Provider((ref) => AuthToken());

final _authRemoteProvider = Provider(
  (ref) => AuthRemoteDataSource(ref.watch(dioProvider)),
);

class AuthState {
  const AuthState({this.user, this.busy = false, this.error});

  final AuthUser? user;
  final bool busy;

  /// [Failure] atau error lain; kalimatnya disusun saat ditampilkan agar
  /// ikut bahasa antarmuka (`AppStrings.describe`).
  final Object? error;

  bool get isLoggedIn => user != null;
}

class AuthController extends Notifier<AuthState> {
  @override
  AuthState build() => const AuthState();

  Future<void> login(String username, String password) async {
    if (state.busy) return; // ketukan ganda tidak perlu dua permintaan
    state = const AuthState(busy: true);
    try {
      final result = await ref
          .read(_authRemoteProvider)
          .login(username, password);
      ref.read(authTokenProvider).set(result.token);

      // Aplikasi mobile ini hanya untuk petugas di lapangan. Supervisor
      // memantau lewat dashboard web; menerimanya di sini akan memberi layar
      // yang seluruh tombolnya ditolak backend.
      if (result.user.role != Role.officer) {
        ref.read(authTokenProvider).clear();
        state = const AuthState(
          error: AccessFailure(FailureKind.supervisorAccount),
        );
        return;
      }
      state = AuthState(user: result.user);
    } catch (e) {
      ref.read(authTokenProvider).clear();
      state = AuthState(error: e);
    }
  }

  void logout() {
    ref.read(authTokenProvider).clear();
    state = const AuthState();
  }
}

final authControllerProvider = NotifierProvider<AuthController, AuthState>(
  AuthController.new,
);
