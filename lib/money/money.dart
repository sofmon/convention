import 'package:decimal/decimal.dart';

class Money {
  final Decimal amount;
  final String currency;

  const Money(this.amount, this.currency);

  factory Money.fromString(String input) {
    final parts = input.trim().split(RegExp(r'\s+'));
    if (parts.length != 2) {
      throw FormatException('Invalid money string');
    }
    final amt = Decimal.parse(parts[0]);
    final cur = parts[1];
    return Money(amt, cur);
  }

  factory Money.fromJson(Map<String, dynamic> json) {
    return Money(Decimal.parse(json['amount'].toString()), json['currency']);
  }

  Map<String, dynamic> toJson() {
    return {'amount': amount.toDouble(), 'currency': currency};
  }

  @override
  String toString() {
    return '${amount.toString()} $currency';
  }
}
