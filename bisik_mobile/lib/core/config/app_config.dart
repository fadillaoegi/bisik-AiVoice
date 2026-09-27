/// Konfigurasi runtime. Override saat build:
/// flutter run --dart-define=API_URL=https://bisik.example.com
class AppConfig {
  const AppConfig._();

  static const String apiUrl = String.fromEnvironment(
    'API_URL',
    defaultValue: 'http://10.0.2.2:8080', // 10.0.2.2 = localhost host dari emulator Android
  );

  static const String wsUrl = String.fromEnvironment(
    'WS_URL',
    defaultValue: 'ws://10.0.2.2:8080',
  );

  /// Harus sama dengan sample rate yang diminta gateway ke AssemblyAI.
  static const int sampleRate = 16000;
}
