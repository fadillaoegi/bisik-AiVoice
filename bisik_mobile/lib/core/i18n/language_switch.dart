import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../theme/bisik_theme.dart';
import 'app_strings.dart';
import 'language.dart';

/// Pengganti bahasa antarmuka — pil kecil dua opsi, kembaran
/// `LanguageSwitch.tsx`. Sengaja kalah menonjol dari skor.
class LanguageSwitch extends ConsumerWidget {
  const LanguageSwitch({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final current = ref.watch(languageProvider);
    final s = context.s;

    return Semantics(
      label: s.languageLabel,
      child: Container(
        padding: const EdgeInsets.all(2),
        decoration: BoxDecoration(
          color: BisikColors.surface,
          border: Border.all(color: BisikColors.border),
          borderRadius: BorderRadius.circular(999),
        ),
        child: Row(
          mainAxisSize: MainAxisSize.min,
          children: [
            for (final language in AppLanguage.values)
              Tooltip(
                message: s.languageName(language),
                child: InkWell(
                  borderRadius: BorderRadius.circular(999),
                  onTap: () =>
                      ref.read(languageProvider.notifier).set(language),
                  child: AnimatedContainer(
                    duration: const Duration(milliseconds: 180),
                    constraints: const BoxConstraints(
                      minWidth: 36,
                      minHeight: 28,
                    ),
                    alignment: Alignment.center,
                    padding: const EdgeInsets.symmetric(horizontal: 9),
                    decoration: BoxDecoration(
                      color: language == current ? BisikColors.accent : null,
                      borderRadius: BorderRadius.circular(999),
                    ),
                    child: Text(
                      s.languageShort(language),
                      style: TextStyle(
                        color: language == current
                            ? BisikColors.accentInk
                            : BisikColors.muted,
                        fontSize: 11,
                        fontWeight: FontWeight.w700,
                        letterSpacing: 0.8,
                      ),
                    ),
                  ),
                ),
              ),
          ],
        ),
      ),
    );
  }
}
