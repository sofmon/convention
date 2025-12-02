import 'package:flutter/material.dart';

class Localized {
  final Map<Locale, String> values;
  const Localized(this.values);

  /// Parse from JSON map with string keys (e.g., "en_GB": "Hello")
  factory Localized.fromJson(Map<String, dynamic> json) {
    final map = <Locale, String>{};
    for (final entry in json.entries) {
      final parts = entry.key.split('_');
      final locale = parts.length >= 2 ? Locale(parts[0], parts[1]) : Locale(parts[0]);
      map[locale] = entry.value as String;
    }
    return Localized(map);
  }

  /// Convert to JSON map with string keys (e.g., "en_GB": "Hello")
  Map<String, String> toJson() {
    final map = <String, String>{};
    for (final entry in values.entries) {
      final key = entry.key.countryCode != null && entry.key.countryCode!.isNotEmpty
          ? '${entry.key.languageCode}_${entry.key.countryCode}'
          : entry.key.languageCode;
      map[key] = entry.value;
    }
    return map;
  }

  String forLocale(Locale locale) {
    return values[locale] ?? values[Locale(locale.languageCode)] ?? values[Locale('en')] ?? '';
  }

  String forContext(BuildContext context) {
    final locale = Localizations.localeOf(context);
    return forLocale(locale);
  }
}
