import 'package:flutter/material.dart';

/// Token warna Bisik — cerminan `:root` di `bisik_frontend/src/index.css`.
///
/// Sebelumnya tiap widget meng-hardcode warna tema gelap GitHub sendiri-sendiri,
/// jadi aplikasi terasa seperti halaman setting developer dan warna merek Bisik
/// tidak pernah muncul. Mint dan amber di bawah diambil dari ikon aplikasi.
///
/// Kalau warna di web berubah, ubah juga di sini — keduanya disamakan manual.
abstract final class BisikColors {
  static const bg = Color(0xFF0B1211);
  static const surface = Color(0xFF121C1A);
  static const surfaceHigh = Color(0xFF172321);
  static const border = Color(0xFF1F302C);
  static const text = Color(0xFFE8F2EE);
  static const muted = Color(0xFF8BA39C);

  /// Mint: merek, elemen interaktif, dan suara petugas.
  static const accent = Color(0xFF6EE7B7);

  /// Teks di atas mint. Putih tidak terbaca di sini.
  static const accentInk = Color(0xFF04231A);

  static const good = accent;
  static const warn = Color(0xFFFBBF24);
  static const bad = Color(0xFFFB7185);

  static const officer = accent;
  static const customer = Color(0xFF7DD3FC);
}

ThemeData bisikTheme() {
  final scheme = ColorScheme.fromSeed(
    seedColor: BisikColors.accent,
    brightness: Brightness.dark,
  ).copyWith(
    surface: BisikColors.bg,
    primary: BisikColors.accent,
    onPrimary: BisikColors.accentInk,
    error: BisikColors.bad,
  );

  return ThemeData(
    useMaterial3: true,
    brightness: Brightness.dark,
    colorScheme: scheme,
    scaffoldBackgroundColor: BisikColors.bg,
    appBarTheme: const AppBarTheme(
      backgroundColor: BisikColors.bg,
      foregroundColor: BisikColors.text,
      elevation: 0,
    ),
    // Layar ini dilirik sambil petugas menatap nasabah, bukan dibaca duduk.
    // Ukuran dasar sengaja lebih besar daripada default Material.
    textTheme: const TextTheme(
      bodyMedium: TextStyle(fontSize: 16, color: BisikColors.text),
      bodySmall: TextStyle(fontSize: 14, color: BisikColors.muted),
    ),
    inputDecorationTheme: const InputDecorationTheme(
      filled: true,
      fillColor: BisikColors.surface,
      border: OutlineInputBorder(),
    ),
  );
}
