import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/network/dio_client.dart';
import '../../../auth/presentation/providers/auth_providers.dart';
import '../../data/datasources/audio_datasource.dart';
import '../../data/datasources/session_remote_datasource.dart';
import '../../data/datasources/session_ws_datasource.dart';
import '../../data/datasources/tts_datasource.dart';
import '../../data/repositories/session_repository_impl.dart';
import '../../domain/repositories/session_repository.dart';
import '../../domain/usecases/end_session.dart';
import '../../domain/usecases/get_report.dart';
import '../../domain/usecases/start_session.dart';
import '../../domain/usecases/stream_session.dart';

/// Composition root. Layer presentation hanya menyentuh use case.
final dioProvider = Provider((ref) => createDio(ref.watch(authTokenProvider)));

final _remoteProvider =
    Provider((ref) => SessionRemoteDataSource(ref.watch(dioProvider)));

final _wsProvider = Provider((ref) {
  final ds = SessionWsDataSource(ref.watch(authTokenProvider));
  ref.onDispose(ds.disconnect);
  return ds;
});

final _audioSourceProvider = Provider((ref) {
  final ds = AudioDataSource();
  ref.onDispose(ds.dispose);
  return ds;
});

final _ttsSourceProvider = Provider((ref) => TtsDataSource());

final sessionRepositoryProvider = Provider<SessionRepositoryImpl>(
  (ref) => SessionRepositoryImpl(
    ref.watch(_remoteProvider),
    ref.watch(_wsProvider),
  ),
);

final audioRepositoryProvider = Provider<AudioRepository>(
  (ref) => AudioRepositoryImpl(ref.watch(_audioSourceProvider)),
);

final speechRepositoryProvider = Provider<SpeechRepository>(
  (ref) => SpeechRepositoryImpl(ref.watch(_ttsSourceProvider)),
);

final startSessionProvider = Provider(
  (ref) => StartSessionUseCase(ref.watch(sessionRepositoryProvider)),
);

final endSessionProvider = Provider(
  (ref) => EndSessionUseCase(ref.watch(sessionRepositoryProvider)),
);

final getReportProvider = Provider(
  (ref) => GetReportUseCase(ref.watch(sessionRepositoryProvider)),
);

final streamSessionProvider = Provider(
  (ref) => StreamSessionUseCase(
    ref.watch(sessionRepositoryProvider),
    ref.watch(audioRepositoryProvider),
  ),
);

final obligationsProvider = FutureProvider(
  (ref) => ref.watch(sessionRepositoryProvider).obligations(),
);
