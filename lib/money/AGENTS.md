# Money Package - Agent Reference

> IMPORTANT: AI agents must treat AGENTS.md and README.md as authoritative living documents. Any change to the implementation that affects behaviors must be mirrored in both files. The code and documentation must never drift apart. When the implementation changes, these documents must be updated immediately so they always reflect the current system.

## Package Purpose

Cross-platform monetary value utility for handling money with currency support. Provides parallel implementations in Go (backend) and Dart (frontend) with guaranteed JSON interoperability.

## File Overview

| File | Language | Purpose |
|------|----------|---------|
| `money.go` | Go | Backend: Currency types, Money with arithmetic, JSON serialization |
| `money.dart` | Dart | Frontend: Money class with Decimal precision |
| `money_test.go` | Go | Tests: 38 tests covering all functionality |
| `money_test.dart` | Dart | Tests: 24 tests with shared fixtures |

## Go Implementation Details

### Key Types

**`Currency`** (line 14)
- Struct with `code` (string) and `factor` (int64)
- `factor` is 10^exponent (100 for 2 decimal places, 1 for 0)
- Predefined: `EUR`, `USD` (factor=100), `JPY` (factor=1)

**`Money`** (line 59)
- Struct with `amount` (int64, minor units) and `currency` (Currency)
- All fields are unexported (private)

### Currency Helper Types (lines 27-29)

```go
type eur struct{ Currency }
type usd struct{ Currency }
type jpy struct{ Currency }
```

These wrapper types provide type-safe factory methods:
- `EUR.FromCents(int64)`, `EUR.FromEuro(int64)`
- `USD.FromCents(int64)`, `USD.FromDollars(int64)`
- `JPY.FromYen(int64)`

### Method Summary

| Type | Method | Line | Purpose |
|------|--------|------|---------|
| `Currency` | `fromMinor(int64)` | 19 | Create Money from minor units |
| `Currency` | `fromMajor(int64)` | 23 | Create Money from major units |
| `Money` | `Add(Money)` | 80 | Addition (panics on currency mismatch) |
| `Money` | `Sub(Money)` | 88 | Subtraction (panics on currency mismatch) |
| `Money` | `Mul(float64)` | 96 | Multiplication with rounding |
| `Money` | `Div(float64)` | 105 | Division with rounding (panics on zero) |
| `Money` | `String()` | 136 | Format as "12.34 EUR" |
| `Money` | `MarshalJSON()` | 178 | JSON serialization |
| `Money` | `UnmarshalJSON(*Money)` | 192 | JSON deserialization |

### Package Functions

| Function | Line | Purpose |
|----------|------|---------|
| `Parse(string)` | 155 | Parse "12.34 EUR" format |
| `currencyByCode(string)` | 117 | Lookup Currency by ISO code |

### Precision Implementation

The Go implementation stores money in **minor units** (int64) to avoid floating-point errors:
- EUR 12.34 is stored as `amount: 1234, currency: EUR`
- JPY 567 is stored as `amount: 567, currency: JPY`

Conversion uses `math.Round()` to handle floating-point imprecision when parsing/unmarshaling.

## Dart Implementation Details

### Key Types

**`Money`** (line 3)
- Immutable class with `Decimal amount` and `String currency`
- Uses `package:decimal` for arbitrary-precision arithmetic

### Method Summary

| Method | Line | Purpose |
|--------|------|---------|
| `Money(Decimal, String)` | 7 | Constructor |
| `Money.fromString(String)` | 9 | Parse "12.34 EUR" format |
| `Money.fromJson(Map)` | 19 | Parse JSON with numeric amount |
| `toJson()` | 23 | Convert to JSON map |
| `toString()` | 28 | Format as "12.34 EUR" |

### Key Implementation Detail

The Dart `toJson()` outputs amount as a **number** (not string):
```dart
return {'amount': amount.toDouble(), 'currency': currency};
```

This ensures compatibility with the Go implementation.

## Cross-Platform Consistency

### JSON Data Contract

Both implementations use identical JSON format:
```json
{"amount": 12.34, "currency": "EUR"}
```

**CRITICAL:** The `amount` field must be a JSON **number**, not a string.

### Key Differences

| Aspect | Go | Dart |
|--------|-----|------|
| Amount storage | `int64` (minor units) | `Decimal` (arbitrary precision) |
| Arithmetic | Methods with panic on error | Not implemented (use Decimal ops) |
| Currency type | `Currency` struct | `String` |
| Mutability | Value type (immutable) | Immutable class |
| Rounding | `math.Round()` on float64 | Decimal precision |

## Common Modification Scenarios

### Adding a New Currency

1. **Go** (`money.go`):
   - Add new wrapper type (e.g., `type gbp struct{ Currency }`)
   - Add variable: `GBP = gbp{newCurrency("GBP", 2)}`
   - Add helper methods (e.g., `FromPence`, `FromPounds`)
   - Add case to `currencyByCode()` switch statement

2. **Dart** (`money.dart`):
   - No changes needed (currency is just a string)

3. **Tests**:
   - Add test fixtures to both test files
   - Add test cases for new currency

### Changing JSON Format

**CRITICAL:** Both implementations must change together.

1. Go: Modify `MarshalJSON()` and `UnmarshalJSON()`
2. Dart: Modify `toJson()` and `fromJson()`
3. Update test fixtures in both test files
4. Run both test suites

### Adding Arithmetic to Dart

If adding arithmetic operations to Dart:
- Use `Decimal` arithmetic (already precise)
- Consider currency validation
- Add corresponding tests

## Dependencies

### Go
- `encoding/json` - JSON marshaling
- `math` - Rounding and power functions
- `strconv` - String conversion
- `strings` - String parsing

### Dart
- `package:decimal/decimal.dart` - Arbitrary-precision decimals

## Testing & Interoperability

The test suite ensures both Go and Dart implementations are fully interoperable - they serialize and deserialize the same JSON format, enabling seamless data exchange between backend and frontend.

### Test Files

| File | Tests | Purpose |
|------|-------|---------|
| `money_test.go` | 38 | Currency helpers, arithmetic, serialization, errors |
| `money_test.dart` | 24 | Serialization, parsing, round-trips |

### Interoperability Guarantee

Both test files define **identical JSON fixture constants**. This ensures:
1. JSON produced by Go can be parsed by Dart
2. JSON produced by Dart can be parsed by Go
3. Round-trip serialization preserves all data

### Shared Test Fixtures

**CRITICAL: Keep these constants identical in both test files.**

| Constant | Value | Tests |
|----------|-------|-------|
| `jsonEUR` | `{"amount":12.34,"currency":"EUR"}` | EUR with decimals |
| `jsonJPY` | `{"amount":567,"currency":"JPY"}` | JPY (no decimals) |
| `jsonUSD` | `{"amount":100,"currency":"USD"}` | USD whole amount |
| `jsonZero` | `{"amount":0,"currency":"EUR"}` | Zero value |
| `jsonNegative` | `{"amount":-50.25,"currency":"EUR"}` | Negative amount |

String fixtures:
| Constant | Value |
|----------|-------|
| `strEUR` | `"12.34 EUR"` |
| `strJPY` | `"567 JPY"` |
| `strUSD` | `"100.00 USD"` (Go) / `"100 USD"` (Dart) |

### Test Coverage

**Go (38 tests):**
- Currency helper creation (FromCents, FromEuro, etc.)
- Arithmetic operations with rounding
- Panic tests (currency mismatch, division by zero)
- String formatting and parsing
- JSON marshal/unmarshal
- Round-trip tests
- Error handling

**Dart (24 tests):**
- JSON deserialization (all fixtures)
- JSON serialization
- Round-trip (JSON and string)
- String parsing with edge cases
- toString formatting

### Running Tests

```bash
# Go tests
go test ./lib/util/money/...

# Dart tests
flutter test lib/util/money/money_test.dart
```

### Modifying Test Fixtures

When adding new test cases:
1. Add the fixture constant to **both** `money_test.go` and `money_test.dart`
2. Ensure the JSON string is byte-for-byte identical
3. Add corresponding test functions in both files
4. Run both test suites to verify interoperability
