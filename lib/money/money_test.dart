import 'dart:convert';
import 'package:decimal/decimal.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:convention/money/money.dart';

// Cross-language test fixtures - KEEP IN SYNC with money_test.go
const jsonEUR = '{"amount":12.34,"currency":"EUR"}';
const jsonJPY = '{"amount":567,"currency":"JPY"}';
const jsonUSD = '{"amount":100,"currency":"USD"}';
const jsonZero = '{"amount":0,"currency":"EUR"}';
const jsonNegative = '{"amount":-50.25,"currency":"EUR"}';

const strEUR = '12.34 EUR';
const strJPY = '567 JPY';
const strUSD = '100 USD';

void main() {
  group('Money deserialization', () {
    test('deserialize EUR', () {
      final json = jsonDecode(jsonEUR) as Map<String, dynamic>;
      final m = Money.fromJson(json);

      expect(m.amount, equals(Decimal.parse('12.34')));
      expect(m.currency, equals('EUR'));
    });

    test('deserialize JPY', () {
      final json = jsonDecode(jsonJPY) as Map<String, dynamic>;
      final m = Money.fromJson(json);

      expect(m.amount, equals(Decimal.parse('567')));
      expect(m.currency, equals('JPY'));
    });

    test('deserialize USD', () {
      final json = jsonDecode(jsonUSD) as Map<String, dynamic>;
      final m = Money.fromJson(json);

      expect(m.amount, equals(Decimal.parse('100')));
      expect(m.currency, equals('USD'));
    });

    test('deserialize zero', () {
      final json = jsonDecode(jsonZero) as Map<String, dynamic>;
      final m = Money.fromJson(json);

      expect(m.amount, equals(Decimal.zero));
      expect(m.currency, equals('EUR'));
    });

    test('deserialize negative', () {
      final json = jsonDecode(jsonNegative) as Map<String, dynamic>;
      final m = Money.fromJson(json);

      expect(m.amount, equals(Decimal.parse('-50.25')));
      expect(m.currency, equals('EUR'));
    });
  });

  group('Money serialization', () {
    test('serialize EUR', () {
      final m = Money(Decimal.parse('12.34'), 'EUR');

      final jsonStr = jsonEncode(m.toJson());

      // Compare by re-parsing (order-independent)
      final expected = jsonDecode(jsonEUR) as Map<String, dynamic>;
      final actual = jsonDecode(jsonStr) as Map<String, dynamic>;

      expect(actual['currency'], equals(expected['currency']));
      expect(actual['amount'], equals(expected['amount']));
    });

    test('serialize JPY', () {
      final m = Money(Decimal.parse('567'), 'JPY');

      final jsonStr = jsonEncode(m.toJson());

      final expected = jsonDecode(jsonJPY) as Map<String, dynamic>;
      final actual = jsonDecode(jsonStr) as Map<String, dynamic>;

      expect(actual['currency'], equals(expected['currency']));
      expect(actual['amount'], equals(expected['amount']));
    });

    test('serialize zero', () {
      final m = Money(Decimal.zero, 'EUR');

      final jsonStr = jsonEncode(m.toJson());

      final expected = jsonDecode(jsonZero) as Map<String, dynamic>;
      final actual = jsonDecode(jsonStr) as Map<String, dynamic>;

      expect(actual['currency'], equals(expected['currency']));
      expect(actual['amount'], equals(expected['amount']));
    });

    test('serialize negative', () {
      final m = Money(Decimal.parse('-50.25'), 'EUR');

      final jsonStr = jsonEncode(m.toJson());

      final expected = jsonDecode(jsonNegative) as Map<String, dynamic>;
      final actual = jsonDecode(jsonStr) as Map<String, dynamic>;

      expect(actual['currency'], equals(expected['currency']));
      expect(actual['amount'], equals(expected['amount']));
    });
  });

  group('Money round-trip', () {
    test('serialize then deserialize preserves values', () {
      final original = Money(Decimal.parse('12.34'), 'EUR');

      final jsonStr = jsonEncode(original.toJson());
      final json = jsonDecode(jsonStr) as Map<String, dynamic>;
      final restored = Money.fromJson(json);

      expect(restored.amount, equals(original.amount));
      expect(restored.currency, equals(original.currency));
    });

    test('deserialize then serialize preserves JSON', () {
      final json = jsonDecode(jsonEUR) as Map<String, dynamic>;
      final m = Money.fromJson(json);
      final output = m.toJson();

      expect(output['currency'], equals(json['currency']));
      expect(output['amount'], equals(json['amount']));
    });

    test('round-trip JPY', () {
      final original = Money(Decimal.parse('567'), 'JPY');

      final jsonStr = jsonEncode(original.toJson());
      final json = jsonDecode(jsonStr) as Map<String, dynamic>;
      final restored = Money.fromJson(json);

      expect(restored.amount, equals(original.amount));
      expect(restored.currency, equals(original.currency));
    });
  });

  group('Money string parsing', () {
    test('parse EUR', () {
      final m = Money.fromString(strEUR);

      expect(m.amount, equals(Decimal.parse('12.34')));
      expect(m.currency, equals('EUR'));
    });

    test('parse JPY', () {
      final m = Money.fromString(strJPY);

      expect(m.amount, equals(Decimal.parse('567')));
      expect(m.currency, equals('JPY'));
    });

    test('parse USD', () {
      final m = Money.fromString(strUSD);

      expect(m.amount, equals(Decimal.parse('100')));
      expect(m.currency, equals('USD'));
    });

    test('parse with extra whitespace', () {
      final m = Money.fromString('  12.34   EUR  ');

      expect(m.amount, equals(Decimal.parse('12.34')));
      expect(m.currency, equals('EUR'));
    });

    test('parse throws on invalid format', () {
      expect(() => Money.fromString('invalid'), throwsFormatException);
    });

    test('parse throws on missing currency', () {
      expect(() => Money.fromString('12.34'), throwsFormatException);
    });
  });

  group('Money toString', () {
    test('toString EUR', () {
      final m = Money(Decimal.parse('12.34'), 'EUR');

      expect(m.toString(), equals(strEUR));
    });

    test('toString JPY', () {
      final m = Money(Decimal.parse('567'), 'JPY');

      expect(m.toString(), equals(strJPY));
    });

    test('toString zero', () {
      final m = Money(Decimal.zero, 'EUR');

      expect(m.toString(), equals('0 EUR'));
    });

    test('toString negative', () {
      final m = Money(Decimal.parse('-50.25'), 'EUR');

      expect(m.toString(), equals('-50.25 EUR'));
    });
  });

  group('Money string round-trip', () {
    test('toString then fromString preserves values', () {
      final original = Money(Decimal.parse('12.34'), 'EUR');

      final str = original.toString();
      final restored = Money.fromString(str);

      expect(restored.amount, equals(original.amount));
      expect(restored.currency, equals(original.currency));
    });

    test('fromString then toString preserves string', () {
      final m = Money.fromString(strEUR);

      expect(m.toString(), equals(strEUR));
    });
  });
}
