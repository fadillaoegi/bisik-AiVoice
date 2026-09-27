import '../entities/session.dart';
import '../repositories/session_repository.dart';

class StartSessionUseCase {
  const StartSessionUseCase(this._repo);
  final SessionRepository _repo;

  /// Identitas petugas TIDAK lagi menjadi parameter: backend mengambilnya
  /// dari token. Validasi "ID petugas wajib diisi" sengaja dihapus — dulu
  /// wajar ketika ID diketik manusia, sekarang justru menolak permintaan
  /// yang sah sebelum sempat dikirim.
  Future<Session> call(String productId) => _repo.start(productId);
}
