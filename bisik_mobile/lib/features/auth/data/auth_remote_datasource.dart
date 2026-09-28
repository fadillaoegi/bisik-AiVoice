import 'package:dio/dio.dart';

import '../../../core/error/failure.dart';
import '../../../core/network/network_failure.dart';
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
        throw const NetworkFailure(
          FailureKind.invalidCredentials,
          action: FailureAction.login,
        );
      }
      throw networkFailure(FailureAction.login, e, _dio.options.baseUrl);
    }
  }
}
