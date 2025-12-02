import 'dart:convert';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:convention/localized/localized.dart';

// Cross-language test fixtures - KEEP IN SYNC with localized_test.go
const jsonSingleLocale = '{"en_GB":"Hello"}';
const jsonMultipleLocales = '{"en_GB":"Hello","nl_NL":"Hallo"}';
const jsonEmpty = '{}';
const jsonUnicode = '{"en_GB":"Hello","zh_CN":"你好"}';
const jsonEmptyValue = '{"en_GB":""}';

void main() {
  group('Localized deserialization', () {
    test('deserialize single locale', () {
      final json = jsonDecode(jsonSingleLocale) as Map<String, dynamic>;
      final l = Localized.fromJson(json);

      expect(l.values[const Locale('en', 'GB')], equals('Hello'));
    });

    test('deserialize multiple locales', () {
      final json = jsonDecode(jsonMultipleLocales) as Map<String, dynamic>;
      final l = Localized.fromJson(json);

      expect(l.values[const Locale('en', 'GB')], equals('Hello'));
      expect(l.values[const Locale('nl', 'NL')], equals('Hallo'));
    });

    test('deserialize empty', () {
      final json = jsonDecode(jsonEmpty) as Map<String, dynamic>;
      final l = Localized.fromJson(json);

      expect(l.values.isEmpty, isTrue);
    });

    test('deserialize unicode', () {
      final json = jsonDecode(jsonUnicode) as Map<String, dynamic>;
      final l = Localized.fromJson(json);

      expect(l.values[const Locale('en', 'GB')], equals('Hello'));
      expect(l.values[const Locale('zh', 'CN')], equals('你好'));
    });

    test('deserialize empty value', () {
      final json = jsonDecode(jsonEmptyValue) as Map<String, dynamic>;
      final l = Localized.fromJson(json);

      expect(l.values[const Locale('en', 'GB')], equals(''));
    });
  });

  group('Localized serialization', () {
    test('serialize single locale', () {
      final l = Localized({const Locale('en', 'GB'): 'Hello'});

      final jsonStr = jsonEncode(l.toJson());

      // Compare by re-parsing (order-independent)
      final expected = jsonDecode(jsonSingleLocale) as Map<String, dynamic>;
      final actual = jsonDecode(jsonStr) as Map<String, dynamic>;

      expect(actual, equals(expected));
    });

    test('serialize multiple locales', () {
      final l = Localized({const Locale('en', 'GB'): 'Hello', const Locale('nl', 'NL'): 'Hallo'});

      final jsonStr = jsonEncode(l.toJson());

      final expected = jsonDecode(jsonMultipleLocales) as Map<String, dynamic>;
      final actual = jsonDecode(jsonStr) as Map<String, dynamic>;

      expect(actual, equals(expected));
    });

    test('serialize empty', () {
      final l = Localized({});

      final jsonStr = jsonEncode(l.toJson());

      expect(jsonStr, equals(jsonEmpty));
    });

    test('serialize unicode', () {
      final l = Localized({const Locale('en', 'GB'): 'Hello', const Locale('zh', 'CN'): '你好'});

      final jsonStr = jsonEncode(l.toJson());

      final expected = jsonDecode(jsonUnicode) as Map<String, dynamic>;
      final actual = jsonDecode(jsonStr) as Map<String, dynamic>;

      expect(actual, equals(expected));
    });
  });

  group('Localized round-trip', () {
    test('serialize then deserialize preserves values', () {
      final original = Localized({const Locale('en', 'GB'): 'Hello', const Locale('nl', 'NL'): 'Hallo'});

      final jsonStr = jsonEncode(original.toJson());
      final json = jsonDecode(jsonStr) as Map<String, dynamic>;
      final restored = Localized.fromJson(json);

      expect(restored.values.length, equals(original.values.length));
      for (final entry in original.values.entries) {
        expect(restored.values[entry.key], equals(entry.value));
      }
    });

    test('deserialize then serialize preserves JSON', () {
      final json = jsonDecode(jsonMultipleLocales) as Map<String, dynamic>;
      final l = Localized.fromJson(json);
      final output = l.toJson();

      expect(output, equals(json));
    });
  });

  group('Localized fallback behavior', () {
    test('exact locale match', () {
      final l = Localized({const Locale('en', 'GB'): 'British English', const Locale('en', 'US'): 'American English'});

      expect(l.forLocale(const Locale('en', 'GB')), equals('British English'));
      expect(l.forLocale(const Locale('en', 'US')), equals('American English'));
    });

    test('language-only fallback', () {
      final l = Localized({const Locale('en'): 'Generic English'});

      expect(l.forLocale(const Locale('en', 'AU')), equals('Generic English'));
    });

    test('english default fallback', () {
      final l = Localized({const Locale('en'): 'English default', const Locale('nl', 'NL'): 'Dutch'});

      expect(l.forLocale(const Locale('fr', 'FR')), equals('English default'));
    });

    test('empty string fallback when no match', () {
      final l = Localized({const Locale('nl', 'NL'): 'Dutch'});

      expect(l.forLocale(const Locale('fr', 'FR')), equals(''));
    });
  });

  group('Locale parsing', () {
    test('parses locale with region code', () {
      final json = {'en_GB': 'Hello'};
      final l = Localized.fromJson(json);

      expect(l.values.keys.first, equals(const Locale('en', 'GB')));
    });

    test('parses locale without region code', () {
      final json = {'en': 'Hello'};
      final l = Localized.fromJson(json);

      expect(l.values.keys.first, equals(const Locale('en')));
    });

    test('serializes locale with region code', () {
      final l = Localized({const Locale('en', 'GB'): 'Hello'});
      final json = l.toJson();

      expect(json.keys.first, equals('en_GB'));
    });

    test('serializes locale without region code', () {
      final l = Localized({const Locale('en'): 'Hello'});
      final json = l.toJson();

      expect(json.keys.first, equals('en'));
    });
  });
}
