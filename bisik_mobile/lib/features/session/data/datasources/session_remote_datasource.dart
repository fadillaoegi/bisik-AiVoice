import 'package:dio/dio.dart';

import '../../../../core/error/failure.dart';

/// Menerjemahkan kegagalan Dio menjadi kalimat yang bisa ditindaklanjuti.
///
/// Penyebab tersering di sini bukan bug aplikasi melainkan alamat gateway:
/// nilai bawaan `10.0.2.2` hanya benar untuk emulator Android dan tidak
/// berarti apa-apa di perangkat fisik maupun simulator iOS. Tanpa alamatnya
/// ikut disebut, petugas hanya melihat "connection error" dan tidak tahu
/// harus mengubah apa.
String _networkMessage(String action, DioException e, String baseUrl) {
  switch (e.type) {
    case DioExceptionType.connectionError:
    case DioExceptionType.connectionTimeout:
      return 'Gagal $action: gateway $baseUrl tidak bisa dihubungi. '
          'Pastikan backend berjalan. Di perangkat fisik atau simulator iOS, '
          'jalankan ulang dengan --dart-define=API_URL=http://<IP-LAN>:8080';
    case DioExceptionType.receiveTimeout:
    case DioExceptionType.sendTimeout:
      return 'Gagal $action: gateway $baseUrl tidak menjawab tepat waktu.';
    default:
      final status = e.response?.statusCode;
      if (status != null) {
        return 'Gagal $action: server menjawab HTTP $status.';
      }
      return 'Gagal $action: ${e.message ?? e.type.name}';
  }
}

class SessionRemoteDataSource {
  const SessionRemoteDataSource(this._dio);
  final Dio _dio;

  // Pemilik sesi tidak dikirim klien: backend mengambilnya dari token.
  // Kalau klien boleh menyebutkan sendiri siapa dirinya, atribusi laporan
  // tidak membuktikan apa pun.
  Future<Map<String, dynamic>> startSession(String productId) async {
    try {
      final res = await _dio.post<Map<String, dynamic>>(
        '/api/sessions',
        data: {'product_id': productId},
      );
      return res.data!;
    } on DioException catch (e) {
      throw NetworkFailure(
          _networkMessage('memulai sesi', e, _dio.options.baseUrl));
    }
  }

  Future<Map<String, dynamic>> endSession(String sessionId) async {
    try {
      final res =
          await _dio.post<Map<String, dynamic>>('/api/sessions/$sessionId/end');
      return res.data!;
    } on DioException catch (e) {
      throw NetworkFailure(
          _networkMessage('mengakhiri sesi', e, _dio.options.baseUrl));
    }
  }

  /// Laporan berbukti. Endpoint yang sama juga melayani sesi yang masih
  /// berjalan, jadi bisa dipakai sebagai snapshot — bukan hanya hasil akhir.
  Future<Map<String, dynamic>> report(String sessionId) async {
    try {
      final res = await _dio.get<Map<String, dynamic>>(
        '/api/sessions/$sessionId/report',
      );
      return res.data!;
    } on DioException catch (e) {
      throw NetworkFailure(
          _networkMessage('memuat laporan', e, _dio.options.baseUrl));
    }
  }

  Future<List<dynamic>> obligations() async {
    try {
      final res = await _dio.get<List<dynamic>>('/api/obligations');
      return res.data ?? const [];
    } on DioException catch (e) {
      throw NetworkFailure(
          _networkMessage('memuat kewajiban', e, _dio.options.baseUrl));
    }
  }
}
