# Localized

A cross-platform localization utility providing consistent locale handling for both Go (backend) and Dart/Flutter (frontend).

## Overview

This package provides types and utilities for storing and retrieving localized strings. It enables content to be stored with multiple language variants and retrieved based on user locale preference.

## Structure

```
localized/
├── localized.go    # Go implementation (backend)
├── localized.dart  # Dart implementation (frontend)
└── README.md
```

## Go Implementation

### Types

#### `Locale`
A string type representing a locale code (e.g., `en_GB`, `nl_NL`).

```go
type Locale string

const (
    EN Locale = "en_GB"
    NL Locale = "nl_NL"
)
```

**Methods:**
- `IsValid() bool` - Checks if the locale exists in the supported locales map
- `Value() (driver.Value, error)` - SQL driver serialization
- `Scan(value interface{}) error` - SQL driver deserialization

#### `Localized`
A map type storing strings keyed by locale.

```go
type Localized map[Locale]string
```

**Methods:**
- `Value() (driver.Value, error)` - Serializes to JSON for SQL storage
- `Scan(value interface{}) error` - Deserializes JSON from SQL
- `Copy() Localized` - Creates a deep copy

### Usage

```go
import "github.com/sofmon/ingreed/lib/util/localized"

// Create localized content
name := localized.Localized{
    localized.EN: "Hello",
    localized.NL: "Hallo",
}

// Validate a locale
locale := localized.Locale("en_GB")
if locale.IsValid() {
    // use locale
}
```

### Database Integration

Both `Locale` and `Localized` implement `sql/driver.Valuer` and `sql.Scanner` interfaces, enabling direct use with database operations:

```go
// Insert
db.Exec("INSERT INTO products (name) VALUES (?)", localizedName)

// Scan
var name localized.Localized
db.QueryRow("SELECT name FROM products WHERE id = ?", id).Scan(&name)
```

## Dart/Flutter Implementation

### Types

#### `Localized`
A class holding locale-to-string mappings.

```dart
class Localized {
  final Map<Locale, String> values;
  const Localized(this.values);
}
```

**Methods:**
- `Localized.fromJson(Map<String, dynamic> json)` - Parse from JSON with string keys (e.g., `"en_GB"`)
- `toJson()` - Convert to JSON map with string keys
- `forLocale(Locale locale)` - Get string for specific locale with fallback chain
- `forContext(BuildContext context)` - Get string using context's current locale

### Usage

```dart
import 'package:convention/localized/localized.dart';

// Create localized content
final name = Localized({
  Locale('en', 'GB'): 'Hello',
  Locale('nl', 'NL'): 'Hallo',
});

// In a widget
Text(name.forContext(context))

// Or with explicit locale
Text(name.forLocale(Locale('en')))

// Parse from JSON (e.g., from API response)
final json = {'en_GB': 'Hello', 'nl_NL': 'Hallo'};
final localized = Localized.fromJson(json);

// Convert to JSON (e.g., for API request)
final map = localized.toJson(); // {'en_GB': 'Hello', 'nl_NL': 'Hallo'}
```

### Fallback Behavior

The Dart implementation uses a fallback chain:
1. Exact locale match (e.g., `en_GB`)
2. Language-only match (e.g., `en`)
3. English (`en`) as default
4. Empty string if nothing found

## Supported Locales

The Go implementation includes a comprehensive list of 500+ locale codes following the [IETF BCP 47](https://en.wikipedia.org/wiki/Locale_(computer_software)) standard. Common examples:

| Code | Language |
|------|----------|
| `en_GB` | English (United Kingdom) |
| `en_US` | English (United States) |
| `nl_NL` | Dutch (Netherlands) |
| `de_DE` | German (Germany) |
| `fr_FR` | French (France) |
| `es_ES` | Spanish (Spain) |

## Data Flow

```
[Database] <--JSON--> [Go Backend] <--API/JSON--> [Dart Frontend] --> [UI]
```

The Go backend stores `Localized` as JSON in the database. When sent to the frontend via API, the Dart code deserializes it into `Localized` for display.

## Testing & Interoperability

Both Go and Dart implementations are tested to ensure they can serialize and deserialize the same JSON format, guaranteeing interoperability between backend and frontend.

### Shared Test Fixtures

Both test files use identical JSON constants to verify cross-language compatibility:

```json
{"en_GB":"Hello"}                       // Single locale
{"en_GB":"Hello","nl_NL":"Hallo"}       // Multiple locales
{}                                       // Empty
{"en_GB":"Hello","zh_CN":"你好"}         // Unicode (CJK characters)
{"en_GB":""}                            // Empty string value
```

### Running Tests

```bash
# Go tests (12 tests)
go test ./lib/util/localized/...

# Dart tests (19 tests) - run from art/ingreed directory
cd art/ingreed && flutter test test/localized_test.dart
```

### What the Tests Verify

- **Serialization**: Both implementations produce the same JSON format
- **Deserialization**: Both implementations correctly parse JSON from the other
- **Round-trip**: Data survives serialize → deserialize → serialize cycles
- **Edge cases**: Empty maps, empty strings, Unicode characters
- **SQL compatibility** (Go): Value/Scan methods work with database drivers
