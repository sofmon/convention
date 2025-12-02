# Money

A cross-platform monetary value utility providing consistent money handling for both Go (backend) and Dart/Flutter (frontend).

## Overview

This package provides types and utilities for storing and manipulating monetary values with currency support. It uses minor units (cents, yen) internally for precision and provides serialization for API communication and database storage.

## Structure

```
money/
├── money.go        # Go implementation (backend)
├── money.dart      # Dart implementation (frontend)
├── money_test.go   # Go tests
├── money_test.dart # Dart tests
└── README.md
```

## Go Implementation

### Types

#### `Currency`
Represents a currency with its ISO 4217 code and decimal factor.

```go
type Currency struct {
    code   string  // ISO 4217 code (e.g., "EUR", "USD", "JPY")
    factor int64   // Conversion factor (100 for EUR/USD, 1 for JPY)
}
```

**Predefined Currencies:**
- `EUR` - Euro (factor: 100, 2 decimal places)
- `USD` - US Dollar (factor: 100, 2 decimal places)
- `JPY` - Japanese Yen (factor: 1, 0 decimal places)

#### `Money`
Stores a monetary value in minor units with its currency.

```go
type Money struct {
    amount   int64     // Amount in minor units (cents, yen)
    currency Currency
}
```

### Creating Money Values

```go
import "github.com/sofmon/ingreed/lib/util/money"

// From minor units
eur := money.EUR.FromCents(1234)    // 12.34 EUR
usd := money.USD.FromCents(500)     // 5.00 USD
jpy := money.JPY.FromYen(1000)      // 1000 JPY

// From major units
eur := money.EUR.FromEuro(100)      // 100.00 EUR
usd := money.USD.FromDollars(50)    // 50.00 USD
```

### Arithmetic Operations

```go
a := money.EUR.FromCents(1000)  // 10.00 EUR
b := money.EUR.FromCents(500)   // 5.00 EUR

sum := a.Add(b)       // 15.00 EUR
diff := a.Sub(b)      // 5.00 EUR
doubled := a.Mul(2)   // 20.00 EUR
half := a.Div(2)      // 5.00 EUR
```

**Note:** `Add` and `Sub` panic if currencies don't match. `Div` panics on division by zero.

### String Conversion

```go
m := money.EUR.FromCents(1234)

// To string
str := m.String()  // "12.34 EUR"

// From string
m, err := money.Parse("12.34 EUR")
if err != nil {
    // handle error
}
```

### JSON Serialization

```go
m := money.EUR.FromCents(1234)

// Marshal
data, _ := json.Marshal(m)
// {"amount":12.34,"currency":"EUR"}

// Unmarshal
var m2 money.Money
json.Unmarshal(data, &m2)
```

## Dart/Flutter Implementation

### Types

#### `Money`
An immutable class holding a decimal amount and currency code.

```dart
class Money {
  final Decimal amount;    // Uses arbitrary-precision Decimal
  final String currency;   // ISO 4217 currency code
}
```

### Creating Money Values

```dart
import 'package:decimal/decimal.dart';
import 'package:convention/money/money.dart';

// Direct construction
final eur = Money(Decimal.parse('12.34'), 'EUR');
final jpy = Money(Decimal.parse('567'), 'JPY');

// From string
final m = Money.fromString('12.34 EUR');

// From JSON
final json = {'amount': 12.34, 'currency': 'EUR'};
final m = Money.fromJson(json);
```

### String Conversion

```dart
final m = Money(Decimal.parse('12.34'), 'EUR');

// To string
print(m.toString());  // "12.34 EUR"

// From string
final m2 = Money.fromString('12.34 EUR');
```

### JSON Serialization

```dart
final m = Money(Decimal.parse('12.34'), 'EUR');

// To JSON
final json = m.toJson();
// {'amount': 12.34, 'currency': 'EUR'}

// From JSON
final m2 = Money.fromJson(json);
```

## Precision Notes

### Minor Units Storage
Money values are stored internally as integers in minor units (cents, yen) to avoid floating-point precision issues in calculations.

### Float64 Conversion Safety
Serialization uses `float64` for JSON/string output. This is safe because:
- `float64` has 53 bits of mantissa, exactly representing integers up to 2^53 (~9 quadrillion)
- For EUR/USD (factor=100), this handles amounts up to ~90 trillion in major units
- `math.Round()` corrects any tiny floating-point errors when parsing back to minor units

## Data Flow

```
[Database] <--JSON--> [Go Backend] <--API/JSON--> [Dart Frontend] --> [UI]
```

The Go backend handles monetary calculations and storage. The Dart frontend displays values and sends user input to the backend.

## JSON Format

Both implementations use the same JSON structure:

```json
{"amount": 12.34, "currency": "EUR"}
{"amount": 567, "currency": "JPY"}
```

**Important:** The `amount` field is always a **number** (not a string) for cross-language interoperability.

## Testing & Interoperability

Both Go and Dart implementations are tested to ensure they can serialize and deserialize the same JSON format, guaranteeing interoperability between backend and frontend.

### Shared Test Fixtures

Both test files use identical JSON constants to verify cross-language compatibility:

```json
{"amount":12.34,"currency":"EUR"}     // EUR with decimals
{"amount":567,"currency":"JPY"}        // JPY (no decimals)
{"amount":100,"currency":"USD"}        // USD whole amount
{"amount":0,"currency":"EUR"}          // Zero
{"amount":-50.25,"currency":"EUR"}     // Negative
```

### Running Tests

```bash
# Go tests (38 tests)
go test ./lib/util/money/...

# Dart tests (24 tests)
flutter test lib/util/money/money_test.dart
```

### What the Tests Verify

- **Currency helpers**: Creating Money from various units
- **Arithmetic**: Add, Sub, Mul, Div with proper rounding
- **Error handling**: Currency mismatch panics, division by zero, unknown currencies
- **Serialization**: Both implementations produce the same JSON format
- **Deserialization**: Both implementations correctly parse JSON from the other
- **Round-trip**: Data survives serialize -> deserialize -> serialize cycles
- **String parsing**: Various formats including edge cases
