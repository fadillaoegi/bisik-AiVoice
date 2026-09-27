import 'package:dio/dio.dart';

import '../../../core/error/failure.dart';
import '../domain/entities/auth_user.dart';

class AuthRemoteDataSource {
  const AuthRemoteDataSource(this._dio);
  final Dio _dio;

  Future<({String token, AuthUser user})> login(
    String username,
    String password,
  ) async {
    try {
      final res = await _dio.post<Map<String, dynamic>>(
        '/api/auth/login',
        data: {'username': username, 'password': password},
      );
      final body = res.data!;
      final user = body['user'] as Map<String, dynamic>;
      return (
        token: body['token'] as String,
        user: AuthUser(
          id: user['id'] as String? ?? '',
          name: user['name'] as String? ?? '',
          role: roleFromString(user['role'] as String?),
        ),
      );
    } on DioException catch (e) {
      // Backend sengaja tidak membedakan akun tidak ada dan sandi salah,
      // supaya endpoint ini tidak bisa dipakai memetakan akun yang valid.
      if (e.response?.statusCode == 401) {
        throw const NetworkFailure('Nama pengguna atau kata sandi salah');
      }
      if (e.type == DioExceptionType.connectionError ||
          e.type == DioExceptionType.connectionTimeout) {
        throw NetworkFailure(
          'Gateway ${_dio.options.baseUrl} tidak bisa dihubungi. Pastikan '
          'backend berjalan; di perangkat fisik jalankan ulang dengan '
          '--dart-define=API_URL=http://<IP-LAN>:8080',
        );
      }
      throw NetworkFailure('Gagal masuk: ${e.message ?? e.type.name}');
    }
  }
}
