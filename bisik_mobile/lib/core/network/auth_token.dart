/// Penyimpan token bearer selama aplikasi hidup.
///
/// Sengaja hanya di memori: petugas masuk sekali setiap aplikasi dibuka.
/// Menyimpannya permanen memerlukan penyimpanan terenkripsi perangkat
/// (`flutter_secure_storage`); `SharedPreferences` tidak terenkripsi dan
/// tidak pantas untuk token. Menambah plugin baru bukan scope hackathon,
/// jadi keterbatasan ini dipilih sadar, bukan kelalaian.
class AuthToken {
  String? _value;

  String? get value => _value;
  bool get isPresent => _value != null && _value!.isNotEmpty;

  void set(String token) => _value = token;
  void clear() => _value = null;
}
