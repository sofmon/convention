# Localized Package - Agent Reference

> IMPORTANT: AI agents must treat AGENTS.md and README.md as authoritative living documents. Any change to the implementation that affects behaviors must be mirrored in both files. The code and documentation must never drift apart. When the implementation changes, these documents must be updated immediately so they always reflect the current system.

## Package Purpose

Cross-platform localization utility for storing multi-language string content. Provides parallel implementations in Go (backend) and Dart (frontend).

## File Overview

| File | Language | Purpose |
|------|----------|---------|
| `localized.go` | Go | Backend: Database-compatible locale and localized string types |
| `localized.dart` | Dart | Frontend: Flutter-compatible localized text with context awareness |

## Go Implementation Details

### Key Types

**`Locale`** (line 11)
- Type alias for `string`
- Represents locale codes (format: `language_REGION`, e.g., `en_GB`)
- Predefined constants: `EN` (`en_GB`), `NL` (`nl_NL`)
- Implements `sql/driver.Valuer` and `sql.Scanner`

**`Localized`** (line 59)
- Type alias for `map[Locale]string`
- Stores translations keyed by locale
- Serializes to/from JSON for database storage
- Implements `sql/driver.Valuer` and `sql.Scanner`

### Important Variables

**`locales`** (line 57)
- Large map of ~500 valid locale codes to their display names
- Used by `Locale.IsValid()` for validation
- Source: IETF BCP 47 / CLDR locale data

### Method Summary

| Type | Method | Purpose |
|------|--------|---------|
| `Locale` | `IsValid()` | Validates against `locales` map |
| `Locale` | `Value()` | SQL serialization (returns string) |
| `Locale` | `Scan()` | SQL deserialization |
| `Localized` | `Value()` | SQL serialization (returns JSON string) |
| `Localized` | `Scan()` | SQL deserialization (parses JSON) |
| `Localized` | `Copy()` | Deep copy of the map |

## Dart Implementation Details

### Key Types

**`Localized`** (line 3)
- Immutable class with `Map<Locale, String> values`
- Uses Flutter's `Locale` type from `material.dart`

### Method Summary

| Method | Purpose |
|--------|---------|
| `Localized.fromJson(Map)` | Factory: Parses JSON map with string keys (e.g., `"en_GB"`) |
| `toJson()` | Converts to JSON map with string keys |
| `forLocale(Locale)` | Returns string for locale with fallback chain |
| `forContext(BuildContext)` | Gets locale from Flutter context, then calls `forLocale` |

### Fallback Chain (line 32-33)
1. Exact locale match
2. Language-only match (strips region)
3. English (`en`) default
4. Empty string

## Cross-Platform Consistency

### Data Contract
Both implementations expect the same JSON structure:
```json
{
  "en_GB": "English text",
  "nl_NL": "Dutch text"
}
```

### Key Differences

| Aspect | Go | Dart |
|--------|-----|------|
| Locale type | Custom `Locale` string alias | Flutter's `Locale` class |
| Locale format | `en_GB` (underscore) | `Locale('en', 'GB')` (constructor) |
| Validation | Explicit via `IsValid()` | None (relies on map lookup) |
| Fallback | None (manual) | Automatic chain in `forLocale()` |
| Mutability | Mutable map | Immutable class |

## Common Modification Scenarios

### Adding a New Supported Locale Constant
1. Go: Add constant in `localized.go` after line 14
2. Ensure locale code exists in `locales` map (line 57)

### Changing Fallback Behavior
- Dart: Modify `forLocale()` method (line 7-9)
- Go: No built-in fallback; implement in consuming code

### Adding New Methods
- Go: Add to `localized.go`, ensure SQL compatibility if needed
- Dart: Add to `Localized` class, consider immutability

### JSON Serialization Changes
- Go: Modify `Value()` and `Scan()` methods on `Localized`
- Dart: Modify `fromJson`/`toJson` methods on `Localized`

## Dependencies

### Go
- `database/sql/driver` - SQL interface implementation
- `encoding/json` - JSON marshaling for `Localized`

### Dart
- `package:flutter/material.dart` - `Locale`, `BuildContext` types

## Testing & Interoperability

The test suite ensures both Go and Dart implementations are fully interoperable - they serialize and deserialize the same JSON format, enabling seamless data exchange between backend and frontend.

### Test Files

| File | Language | Purpose |
|------|----------|---------|
| `lib/util/localized/localized_test.go` | Go | Tests JSON/SQL serialization, locale validation |
| `art/ingreed/test/localized_test.dart` | Dart | Tests JSON serialization, fallback behavior |

### Interoperability Guarantee

Both test files define **identical JSON fixture constants**. This ensures:
1. JSON produced by Go can be parsed by Dart
2. JSON produced by Dart can be parsed by Go
3. Round-trip serialization preserves all data

### Shared Test Fixtures

**CRITICAL: Keep these constants identical in both test files.**

| Constant | Value | Tests |
|----------|-------|-------|
| `jsonSingleLocale` | `{"en_GB":"Hello"}` | Basic single entry |
| `jsonMultipleLocales` | `{"en_GB":"Hello","nl_NL":"Hallo"}` | Multiple entries |
| `jsonEmpty` | `{}` | Empty map edge case |
| `jsonUnicode` | `{"en_GB":"Hello","zh_CN":"你好"}` | Unicode/CJK characters |
| `jsonEmptyValue` | `{"en_GB":""}` | Empty string value |

### Test Coverage

**Go (12 tests):**
- Deserialization from all fixture formats
- Serialization to expected JSON
- JSON round-trip (marshal → unmarshal)
- SQL Value/Scan round-trip
- Locale validation
- Copy functionality

**Dart (19 tests):**
- Deserialization from all fixture formats
- Serialization to expected JSON
- JSON round-trip
- Fallback behavior (exact match, language-only, English default, empty)
- Locale parsing with/without region codes

### Running Tests

```bash
# Go tests
go test ./lib/util/localized/...

# Dart tests (from art/ingreed directory)
cd art/ingreed && flutter test test/localized_test.dart
```

### Modifying Test Fixtures

When adding new test cases:
1. Add the fixture constant to **both** `localized_test.go` and `localized_test.dart`
2. Ensure the JSON string is byte-for-byte identical
3. Add corresponding test functions in both files
4. Run both test suites to verify interoperability
