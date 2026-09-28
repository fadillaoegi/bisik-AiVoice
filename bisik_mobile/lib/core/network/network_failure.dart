import 'package:dio/dio.dart';

import '../error/failure.dart';

/// Mengklasifikasikan kegagalan Dio menjadi kegagalan yang bisa
/// ditindaklanjuti. Kalimatnya disusun di presentasi (`AppStrings.failure`).
///
/// Penyebab tersering di sini bukan bug aplikasi melainkan alamat gateway:
/// nilai bawaan `10.0.2.2` hanya benar untuk emulator Android dan tidak
/// berarti apa-apa di perangkat fisik maupun simulator iOS. Karena itu
/// alamatnya ikut dibawa — tanpanya petugas hanya melihat "connection error"
/// dan tidak tahu harus mengubah apa.
NetworkFailure networkFailure(
  FailureAction action,
  DioException e,
  String baseUrl,
) {
  switch (e.type) {
    case DioExceptionType.connectionError:
    case DioExceptionType.connectionTimeout:
      return NetworkFailure(
        FailureKind.gatewayUnreachable,
        action: action,
        detail: baseUrl,
      );
    case DioExceptionType.receiveTimeout:
    case DioExceptionType.sendTimeout:
      return NetworkFailure(
        FailureKind.gatewayTimeout,
        action: action,
        detail: baseUrl,
      );
    default:
      final status = e.response?.statusCode;
      if (status != null) {
        return NetworkFailure(
          FailureKind.httpStatus,
          action: action,
          detail: '$status',
        );
      }
      return NetworkFailure(
        FailureKind.requestFailed,
        action: action,
        detail: e.message ?? e.type.name,
      );
  }
}
