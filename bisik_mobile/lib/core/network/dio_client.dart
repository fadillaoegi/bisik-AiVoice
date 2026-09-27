import 'package:dio/dio.dart';

import '../config/app_config.dart';
import 'auth_token.dart';

Dio createDio(AuthToken token) {
  final dio = Dio(
    BaseOptions(
      baseUrl: AppConfig.apiUrl,
      connectTimeout: const Duration(seconds: 8),
      receiveTimeout: const Duration(seconds: 8),
      headers: {'Content-Type': 'application/json'},
    ),
  );

  // Token disisipkan di satu tempat, bukan di tiap pemanggilan: kalau tidak,
  // satu endpoint yang terlupa akan gagal dengan 401 yang membingungkan.
  dio.interceptors.add(
    InterceptorsWrapper(
      onRequest: (options, handler) {
        final value = token.value;
        if (value != null && value.isNotEmpty) {
          options.headers['Authorization'] = 'Bearer $value';
        }
        handler.next(options);
      },
    ),
  );

  return dio;
}
