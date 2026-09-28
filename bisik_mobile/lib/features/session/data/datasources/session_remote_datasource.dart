import 'package:dio/dio.dart';

import '../../../../core/error/failure.dart';
import '../../../../core/network/network_failure.dart';

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
      throw networkFailure(FailureAction.startSession, e, _dio.options.baseUrl);
    }
  }

  Future<Map<String, dynamic>> endSession(String sessionId) async {
    try {
      final res = await _dio.post<Map<String, dynamic>>(
        '/api/sessions/$sessionId/end',
      );
      return res.data!;
    } on DioException catch (e) {
      throw networkFailure(FailureAction.endSession, e, _dio.options.baseUrl);
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
      throw networkFailure(FailureAction.loadReport, e, _dio.options.baseUrl);
    }
  }

  Future<List<dynamic>> obligations() async {
    try {
      final res = await _dio.get<List<dynamic>>('/api/obligations');
      return res.data ?? const [];
    } on DioException catch (e) {
      throw networkFailure(
        FailureAction.loadObligations,
        e,
        _dio.options.baseUrl,
      );
    }
  }
}
