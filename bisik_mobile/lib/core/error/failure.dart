import '../i18n/app_strings.dart';

/// Jenis kegagalan. Kalimatnya disusun di lapisan presentasi lewat
/// [AppStrings.failure], supaya ikut bahasa antarmuka yang dipilih petugas.
enum FailureKind {
  invalidCredentials,
  gatewayUnreachable,
  gatewayTimeout,
  httpStatus,
  requestFailed,
  micPermissionDenied,
  micUnavailable,
  supervisorAccount,

  /// Gateway melaporkan jalur audio mati tanpa menyebut alasannya.
  audioPathLost,
}

/// Apa yang sedang dicoba saat gagal — bagian dari kalimat error.
enum FailureAction {
  login,
  startSession,
  endSession,
  loadReport,
  loadObligations,
}

sealed class Failure implements Exception {
  const Failure(this.kind, {this.action, this.detail});

  final FailureKind kind;
  final FailureAction? action;

  /// Alamat gateway, kode HTTP, atau pesan mentah — tergantung [kind].
  final String? detail;

  /// Untuk log dan `'$e'`: selalu Bahasa Indonesia, bahasa bawaan aplikasi.
  @override
  String toString() => AppStrings.id.failure(this);
}

class NetworkFailure extends Failure {
  const NetworkFailure(super.kind, {super.action, super.detail});
}

class PermissionFailure extends Failure {
  const PermissionFailure(super.kind, {super.action, super.detail});
}

class AudioFailure extends Failure {
  const AudioFailure(super.kind, {super.action, super.detail});
}

/// Akun sah, tetapi perannya tidak boleh memakai aplikasi ini.
class AccessFailure extends Failure {
  const AccessFailure(super.kind, {super.action, super.detail});
}
