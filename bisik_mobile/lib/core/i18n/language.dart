import 'dart:async';

import 'package:flutter/widgets.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:shared_preferences/shared_preferences.dart';

import 'app_strings.dart';

/// Diisi di `main()` sebelum `runApp`, supaya bahasa tersimpan sudah
/// terbaca pada frame pertama — tanpa kedip dari Bahasa Indonesia ke
/// bahasa pilihan. Null kalau penyimpanan perangkat tidak tersedia; aplikasi
/// tetap jalan dengan bahasa bawaan.
final sharedPreferencesProvider = Provider<SharedPreferences?>((ref) => null);

class LanguageController extends Notifier<AppLanguage> {
  static const _key = 'bisik.lang';

  @override
  AppLanguage build() {
    final stored = ref.watch(sharedPreferencesProvider)?.getString(_key);
    return AppLanguage.values.asNameMap()[stored] ?? AppLanguage.id;
  }

  void set(AppLanguage language) {
    state = language;
    final prefs = ref.read(sharedPreferencesProvider);
    if (prefs != null) unawaited(prefs.setString(_key, language.name));
  }
}

final languageProvider = NotifierProvider<LanguageController, AppLanguage>(
  LanguageController.new,
);

/// Membagikan kamus aktif ke seluruh pohon widget.
///
/// Dipasang sekali lewat `MaterialApp.builder`, jadi widget biasa cukup
/// memanggil `context.s` — tidak perlu diubah menjadi Consumer satu per satu.
class StringsScope extends InheritedWidget {
  const StringsScope({super.key, required this.strings, required super.child});

  final AppStrings strings;

  static AppStrings of(BuildContext context) =>
      context.dependOnInheritedWidgetOfExactType<StringsScope>()?.strings ??
      AppStrings.id;

  @override
  bool updateShouldNotify(StringsScope oldWidget) =>
      oldWidget.strings != strings;
}

extension StringsContext on BuildContext {
  AppStrings get s => StringsScope.of(this);
}
