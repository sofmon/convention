# DynamicFormWidget - AI Agent Development Guide

## Current Implementation Status

**Status**: ✅ IMPLEMENTED (v2.2.0 - 2025-12-04)
**Architecture**: Map-based runtime system (no code generation)
**Latest Features**: Auto-discovery, optional fieldConfigs, custom default widgets, auto-labeling

---

## Executive Summary

DynamicFormWidget is a Flutter widget that generates view and edit UIs from `Map<String, dynamic>` data structures. Unlike traditional form builders that require typed models and code generation, this system works entirely at runtime using type inference and explicit configuration.

**Key Innovations**:
- No code generation, no reflection, no tree-shaking issues
- Auto-discovery of fields from value map (no fieldConfigs required)
- Custom default widgets per FieldType via `DynamicFormTheme`
- Auto-labeling via `LabelResolver` with ARB resource support

---

## Architectural Evolution

### v1.0.0 (Deprecated)
- Used generic type `AutoWidgetBuilder<T>`
- Required `@AutoFormModel` annotations
- Required code generation with build_runner
- Suffered from tree-shaking issues on web
- Complex setup and maintenance

### v2.2.0 (Current)
- Auto-discovery of fields from `value` map
- Optional `fieldConfigs` - only provides overrides
- `fieldConfig` in `DynamicFormTheme` for project-wide defaults

### v2.1.0
- Custom default widgets via `DynamicFormTheme`
- Auto-labeling via `LabelResolver`
- Optional `FieldConfig.label` (auto-resolved if null)
- Project-wide configuration through InheritedWidget

### v2.0.0
- Uses `Map<String, dynamic>` data
- No annotations required
- No code generation required
- Runtime type inference from values
- Simple, reliable, tree-shake friendly

### Migration Rationale

The v1.0.0 approach was abandoned because:
1. **Tree-shaking unreliability** - Web builds would remove schema registrations
2. **Complexity overhead** - Code generation added significant complexity
3. **Developer friction** - Requires build_runner step on every model change
4. **Limited flexibility** - Typed models are rigid for dynamic forms

The v2.0.0 Map-based approach solves all these issues while maintaining full functionality.

---

## Core Architecture

### 1. Data Layer - Maps Instead of Types

**Philosophy**: Data is `Map<String, dynamic>`, configuration is explicit.

```dart
// Data
final userData = {
  'name': 'John Doe',
  'email': 'john@example.com',
  'age': 30,
  'isActive': true,
};

// Configuration
final fieldConfigs = {
  'name': const FieldConfig(label: 'Full Name'),  // type inferred
  'email': const FieldConfig(label: 'Email'),     // type inferred
  'age': const FieldConfig(label: 'Age', type: FieldType.int),
  'isActive': const FieldConfig(label: 'Active'), // type inferred
};
```

### 2. Schema Layer (`schema.dart`)

**Core Types**:

```dart
enum FieldType {
  string, int, double, bool, dateTime, enumType, list, nested
}

/// Resolves a label for the given field name (in snake_case).
typedef LabelResolver = String? Function(String fieldNameSnakeCase);

class FieldConfig {
  final String? label;                // null = auto-resolve via LabelResolver
  final FieldType? type;              // null = infer from value
  final bool required;
  final String? hint;
  final String? validationError;
  final List<dynamic>? enumValues;    // for enums
  final Map<String, FieldConfig>? nestedFields;  // for nested objects
  final Widget Function(...)? widget;  // custom widget builder
}

/// Helper functions
String toSnakeCase(String input);       // "firstName" -> "first_name"
String humanizeFieldName(String snake); // "first_name" -> "First Name"
```

**Type Inference Algorithm**:
```dart
static FieldType? inferType(dynamic value) {
  if (value is String) return FieldType.string;
  if (value is int) return FieldType.int;
  if (value is double) return FieldType.double;
  if (value is bool) return FieldType.bool;
  if (value is DateTime) return FieldType.dateTime;
  if (value is List) return FieldType.list;
  if (value is Map) return FieldType.nested;
  return null;  // enums need explicit config
}
```

### 3. Widget Layer (`main.dart`)

**DynamicFormWidget**:
```dart
class DynamicFormWidget extends StatefulWidget {
  final Map<String, dynamic> value;
  final Map<String, FieldConfig>? fieldConfigs;  // Optional overrides
  final AutoWidgetMode mode;  // view or edit
  final void Function(Map<String, dynamic>)? onChanged;
  final Widget Function(...)? layoutBuilder;
  final LabelResolver? labelResolver;
}
```

**Field Discovery** (NEW in v2.2.0):
```dart
// All fields auto-discovered from value.keys
Iterable<String> _getFieldKeys() => widget.value.keys;

// Config priority: fieldConfigs[key] > theme.fieldConfig > FieldConfig()
FieldConfig _getFieldConfig(String fieldKey) {
  if (widget.fieldConfigs?.containsKey(fieldKey) ?? false) {
    return widget.fieldConfigs![fieldKey]!;
  }
  final theme = DynamicFormTheme.of(context);
  if (theme?.fieldConfig != null) return theme!.fieldConfig!;
  return const FieldConfig();
}
```

**State Management**:
```dart
class DynamicFormWidgetState extends State<DynamicFormWidget> {
  Future<Map<String, dynamic>> save();   // validates and returns updated Map
  ValidationResult validate();           // validates without saving
  void reset();                          // resets to original values

  // Internal: dot notation support
  dynamic _getValueByPath(String path);  // 'user.address.street'
  void _setValueByPath(String path, dynamic value);
}
```

### 4. Field Widgets (`field_widgets/default_field_widgets.dart`)

**Default Implementation Matrix**:

| FieldType | View Mode | Edit Mode | Special Features |
|-----------|-----------|-----------|------------------|
| `string` | Text | TextFormField | - |
| `int` | Text | TextFormField | Numeric keyboard, digits only |
| `double` | Text | TextFormField | Decimal keyboard, format validation |
| `bool` | Icon (✓/✗) | Switch | - |
| `dateTime` | Formatted text | DatePicker + TimePicker | intl formatting |
| `enumType` | Enum name | DropdownButtonFormField | Requires enumValues |
| `list` | Bullet list | Add/remove controls | Simplified editing |
| `nested` | Card display | Recursive form | Dot notation support |

**Builder API**:
```dart
static Widget buildWidgetByType({
  required FieldType type,
  required String label,
  required dynamic value,
  required AutoWidgetMode mode,
  required ValueChanged<dynamic> onChanged,
  bool required = true,
  String? hint,
  String? validationError,
  List<dynamic>? enumValues,
  Map<String, FieldConfig>? nestedFields,
  GlobalKey<FormFieldState>? fieldKey,
})
```

---

## Key Design Decisions

### Decision 1: Map-Based Data Model
**Chosen**: `Map<String, dynamic>` instead of generic `T`

**Rationale**:
- No code generation overhead
- Works with JSON APIs directly
- Flexible for dynamic forms
- No tree-shaking issues
- Easier to test and debug

**Trade-offs**:
- ❌ Loss of compile-time type safety
- ✅ Gain runtime flexibility
- ✅ Simpler architecture
- ✅ Better for JSON/API workflows

### Decision 2: Type Inference with Opt-Out
**Chosen**: Infer types from runtime values, allow explicit override

**Rationale**:
- Reduces boilerplate for common cases
- Still allows explicit types when needed (enums, ambiguous numbers)
- Fails gracefully with clear error messages

**Example**:
```dart
// Inferred as string
'name': const FieldConfig(label: 'Name'),

// Explicit type
'accountType': FieldConfig(
  label: 'Account Type',
  type: FieldType.enumType,
  enumValues: AccountType.values,
),
```

### Decision 3: Dot Notation for Nested Fields
**Chosen**: Support `"user.address.street"` path syntax

**Rationale**:
- Intuitive for developers
- Matches JSON path conventions
- Avoids deep nesting in configuration
- Simplifies form layout

**Implementation**:
```dart
// Split path, traverse Map structure
final parts = path.split('.');
dynamic current = _fieldValues;
for (final part in parts) {
  if (current is Map) current = current[part];
}
```

### Decision 4: Custom Widgets via Function Builders
**Chosen**: `Widget Function(context, value, onChanged, mode)` instead of class types

**Rationale**:
- More flexible than class-based widgets
- Easier to write inline
- Can capture closure variables
- Familiar pattern for Flutter developers

**Example**:
```dart
'phoneNumber': FieldConfig(
  label: 'Phone',
  widget: (context, value, onChanged, mode) {
    if (mode == AutoWidgetMode.view) {
      return Text(value ?? '—');
    }
    return TextFormField(
      initialValue: value,
      keyboardType: TextInputType.phone,
      onChanged: onChanged,
    );
  },
),
```

### Decision 5: Exception-Based Validation (Retained from v1.0.0)
**Chosen**: Throw `ValidationException` on save failure

**Rationale**:
- Forces explicit error handling
- Provides detailed field-level errors
- Idiomatic Dart pattern
- Separates validation from business logic

---

## File Structure

```
lib/dynamic_form/
├── AGENTS.md                          # This file
├── README.md                          # User documentation
├── main.dart                          # DynamicFormWidget
├── schema.dart                        # FieldConfig, FieldType, LabelResolver
├── field_widget.dart                  # AutoWidgetMode, ValidationResult, DynamicFormFieldBuilder
├── theme.dart                         # DynamicFormTheme (NEW in v2.1.0)
├── field_widgets/
│   └── default_field_widgets.dart     # Default field implementations
└── example/
    ├── models.dart                    # Example enums (no annotations)
    └── example_app.dart               # Demo app with theme and auto-labeling

# Removed in v2.0.0:
# ❌ annotations.dart
# ❌ generator.dart
# ❌ generator/schema_generator.dart
# ❌ build.yaml
# ❌ *.auto_widget.dart (generated files)
```

---

## Usage Patterns

### Pattern 1: Zero Config (Auto-Discovery) - NEW in v2.2.0

```dart
final data = {
  'name': 'John',
  'age': 30,
  'active': true,
};

// No fieldConfigs needed! All fields auto-discovered
DynamicFormWidget(
  value: data,
  mode: AutoWidgetMode.edit,
)
```

### Pattern 2: Selective Overrides - NEW in v2.2.0

```dart
final data = {
  'name': 'John',
  'age': 30,
  'active': true,
};

// Only override specific fields
final configs = {
  'age': const FieldConfig(type: FieldType.int, hint: 'Enter age'),
};

DynamicFormWidget(
  value: data,
  fieldConfigs: configs,  // Only overrides 'age', others use defaults
  mode: AutoWidgetMode.edit,
)
```

### Pattern 3: Theme-Wide Defaults - NEW in v2.2.0

```dart
// All fields optional by default
DynamicFormTheme(
  fieldConfig: const FieldConfig(required: false),
  child: DynamicFormWidget(
    value: {'name': 'John', 'email': 'john@example.com'},
    mode: AutoWidgetMode.edit,
  ),
)
```

### Pattern 4: Enum Fields (Explicit Type Required)

```dart
enum Status { active, inactive, pending }

final data = {'status': Status.active};

final configs = {
  'status': FieldConfig(
    type: FieldType.enumType,           // explicit
    enumValues: Status.values,          // required for enums
  ),
};

DynamicFormWidget(
  value: data,
  fieldConfigs: configs,  // Must provide config for enums
  mode: AutoWidgetMode.edit,
)
```

### Pattern 5: Nested Fields with Dot Notation

```dart
final data = {
  'user': {
    'address': {
      'street': '123 Main St',
      'city': 'Springfield',
    },
  },
};

final configs = {
  'user.address.street': const FieldConfig(label: 'Street'),
  'user.address.city': const FieldConfig(label: 'City'),
};
```

### Pattern 6: Custom Widget Builder

```dart
final configs = {
  'rating': FieldConfig(
    widget: (context, value, onChanged, mode) {
      if (mode == AutoWidgetMode.view) {
        return Row(
          children: List.generate(
            5,
            (i) => Icon(i < value ? Icons.star : Icons.star_border),
          ),
        );
      }
      return Slider(
        value: (value ?? 0).toDouble(),
        min: 0,
        max: 5,
        divisions: 5,
        onChanged: (v) => onChanged(v.toInt()),
      );
    },
  ),
};
```

### Pattern 7: Save and Validation

```dart
final _formKey = GlobalKey<DynamicFormWidgetState>();

// In build:
DynamicFormWidget(key: _formKey, ...)

// Save:
Future<void> save() async {
  try {
    final updated = await _formKey.currentState!.save();
    // Send to API: await api.post('/users', updated);
    setState(() => _data = updated);
  } on ValidationException catch (e) {
    // Show errors: e.errors is Map<String, String>
    showDialog(
      context: context,
      builder: (context) => AlertDialog(
        title: Text('Validation Failed'),
        content: Text(e.errors.values.join('\n')),
      ),
    );
  }
}
```

---

## Type Inference Details

### Inference Rules

1. **Primitive Types** - Direct type check
   ```dart
   'name': 'John'     → FieldType.string
   'age': 30          → FieldType.int
   'price': 29.99     → FieldType.double
   'active': true     → FieldType.bool
   ```

2. **DateTime** - Instance check
   ```dart
   'created': DateTime.now()  → FieldType.dateTime
   ```

3. **Collections**
   ```dart
   'tags': ['a', 'b']              → FieldType.list
   'address': {'street': '...'}    → FieldType.nested
   ```

4. **Enums** - Cannot infer, must specify
   ```dart
   'status': Status.active  → Must use type: FieldType.enumType
   ```

### Fallback Behavior

When type cannot be inferred:
```dart
// DynamicFormWidget displays error widget
Text(
  'Unknown field type for "field_name"',
  style: TextStyle(color: Colors.red),
)
```

User should add explicit `type` to FieldConfig.

---

## Nested Fields Implementation

### Path Traversal

```dart
// Get value
dynamic _getValueByPath(String path) {
  final parts = path.split('.');
  dynamic current = _fieldValues;
  for (final part in parts) {
    if (current is Map) {
      current = current[part];
    } else {
      return null;  // path invalid
    }
  }
  return current;
}

// Set value
void _setValueByPath(String path, dynamic value) {
  final parts = path.split('.');
  dynamic current = _fieldValues;

  // Navigate to parent
  for (int i = 0; i < parts.length - 1; i++) {
    if (!current.containsKey(parts[i]) || current[parts[i]] is! Map) {
      current[parts[i]] = <String, dynamic>{};  // auto-create
    }
    current = current[parts[i]];
  }

  // Set final value
  current[parts.last] = value;
}
```

### Auto-Vivification

Missing intermediate maps are automatically created:
```dart
// Given: {}
// Set: 'user.address.street' = '123 Main'
// Result: {'user': {'address': {'street': '123 Main'}}}
```

---

## Validation System

### Built-in Validation

1. **Required Fields**
   ```dart
   if (config.required && value == null) {
     errors[fieldPath] = config.validationError ?? '${config.label} is required';
   }
   ```

2. **Type Validation** (in field widgets)
   ```dart
   // int field
   if (v != null && v.isNotEmpty && int.tryParse(v) == null) {
     return 'Must be a valid integer';
   }
   ```

### Custom Validation

Via `validationError` parameter:
```dart
'email': const FieldConfig(
  label: 'Email',
  required: true,
  validationError: 'Valid email is required',
),
```

### Validation Flow

```
User clicks Save
  ↓
DynamicFormWidgetState.save()
  ↓
validate() called
  ↓
For each field:
  - Check required
  - Check FormField.validate()
  ↓
If errors:
  - Update UI with error messages
  - Throw ValidationException
Else:
  - Return updated Map
```

---

## Extension Points

### 1. Custom Field Widgets (Per-Field)

Use `widget` parameter in FieldConfig:
```dart
FieldConfig(
  widget: (context, value, onChanged, mode) => MyWidget(),
)
```

### 2. Custom Default Widgets (Per-Type) - NEW in v2.1.0

Use `DynamicFormTheme` to replace default widgets for specific types:
```dart
DynamicFormTheme(
  builders: {
    FieldType.bool: ({required label, required value, required mode, required onChanged, ...}) {
      return MySwitchWidget(label: label, value: value, onChanged: onChanged);
    },
    FieldType.dateTime: ({required label, ...}) => MyDatePicker(...),
  },
  child: MyApp(),
)
```

**Widget Priority Order**:
1. `FieldConfig.widget` (per-field) - HIGHEST
2. `DynamicFormTheme.builders[type]` (custom default by type)
3. Built-in default widget - LOWEST

### 3. Auto-Labeling - NEW in v2.1.0

Use `LabelResolver` for automatic label resolution:
```dart
// Widget-level
DynamicFormWidget(
  labelResolver: (fieldNameSnakeCase) {
    return myArbLookup('form_$fieldNameSnakeCase');
  },
)

// Project-wide via theme
DynamicFormTheme(
  labelResolver: (fieldNameSnakeCase) => myArbLookup(fieldNameSnakeCase),
  child: MyApp(),
)
```

**Label Resolution Order**:
1. `FieldConfig.label` (explicit)
2. `DynamicFormWidget.labelResolver`
3. `DynamicFormTheme.labelResolver`
4. Humanized field name (`firstName` → `First Name`)

### 4. Custom Layout

Use `layoutBuilder` parameter:
```dart
DynamicFormWidget(
  layoutBuilder: (context, fieldWidgets) {
    return GridView.count(
      crossAxisCount: 2,
      children: fieldWidgets,
    );
  },
)
```

### 5. Custom Field Types

Extend `FieldType` enum and `buildWidgetByType` switch:
```dart
// In schema.dart
enum FieldType {
  // ... existing types
  richText,  // new type
}

// In default_field_widgets.dart
case FieldType.richText:
  return _buildRichTextField(...);
```

### 6. Formatters and Transformers

Wrap custom widget logic:
```dart
widget: (context, value, onChanged, mode) {
  return TextFormField(
    initialValue: formatCurrency(value),
    onChanged: (text) => onChanged(parseCurrency(text)),
  );
}
```

---

## Performance Considerations

### Map Access Overhead

**Impact**: Minimal. Map access is O(1) hash lookup.

**Measurement**: For 100 fields, Map access adds <1ms vs. direct field access.

### Dot Notation Path Parsing

**Impact**: Negligible. Parsing happens once per field change.

**Optimization**: Could cache split paths if needed.

### Type Inference

**Impact**: One-time cost per field on first render.

**Optimization**: Results are cached in getEffectiveType().

### Comparison to v1.0.0

| Metric | v1.0.0 (Codegen) | v2.0.0 (Map) |
|--------|------------------|--------------|
| Build time | +2-5s per model change | 0s |
| Runtime overhead | None (compiled) | <1ms per 100 fields |
| Memory | Lower (typed) | Slightly higher (dynamic) |
| Code size | Higher (generated) | Lower (no generation) |

**Verdict**: Runtime overhead is negligible; development velocity gain is significant.

---

## Testing Strategy

### Unit Tests
```dart
test('type inference', () {
  expect(FieldConfig.inferType('hello'), FieldType.string);
  expect(FieldConfig.inferType(42), FieldType.int);
  expect(FieldConfig.inferType(3.14), FieldType.double);
});

test('dot notation get', () {
  final data = {'user': {'name': 'John'}};
  final state = DynamicFormWidgetState(...);
  expect(state._getValueByPath('user.name'), 'John');
});

test('validation', () {
  final config = FieldConfig(label: 'Name', required: true);
  // ... validation logic
});
```

### Widget Tests
```dart
testWidgets('switches between view and edit modes', (tester) async {
  await tester.pumpWidget(
    DynamicFormWidget(
      value: {'name': 'John'},
      fieldConfigs: {'name': FieldConfig(label: 'Name')},
      mode: AutoWidgetMode.view,
    ),
  );

  expect(find.text('John'), findsOneWidget);
  expect(find.byType(TextFormField), findsNothing);

  // Switch to edit
  // ... test edit mode
});
```

### Integration Tests
Run example app and test:
1. Field editing
2. Save/reset operations
3. Validation errors
4. Enum dropdowns
5. DateTime pickers

---

## Migration Guide (v1.0.0 → v2.0.0)

### Before (v1.0.0)
```dart
// 1. Define model
@AutoFormModel()
class User {
  final String name;
  final int age;
  User({required this.name, required this.age});
}

// 2. Generate code
// flutter pub run build_runner build

// 3. Import generated file
import 'user.auto_widget.dart';

// 4. Use
AutoWidgetBuilder<User>(
  value: User(name: 'John', age: 30),
  mode: AutoWidgetMode.edit,
)
```

### After (v2.0.0)
```dart
// 1. Define data
final userData = {'name': 'John', 'age': 30};

// 2. Define config
final userConfig = {
  'name': const FieldConfig(label: 'Name'),
  'age': const FieldConfig(label: 'Age', type: FieldType.int),
};

// 3. Use (no codegen)
DynamicFormWidget(
  value: userData,
  fieldConfigs: userConfig,
  mode: AutoWidgetMode.edit,
)
```

### Migration Steps
1. Remove annotations from models
2. Convert model instances to Maps
3. Create FieldConfig map
4. Replace AutoWidgetBuilder with DynamicFormWidget
5. Delete build.yaml and generated files
6. Remove build_runner from dependencies

---

## Known Limitations

### 1. List Reordering
**Current**: Lists support add/remove and complex objects, but no drag-to-reorder
**Limitation**: No built-in reordering UI

**Workaround**:
```dart
'items': FieldConfig(
  label: 'Items',
  widget: (context, value, onChanged, mode) {
    return ReorderableListView(...);
  },
),
```

### 2. Type Ambiguity
**Current**: Cannot distinguish int from double without value
**Limitation**: Empty fields default to inferred type

**Workaround**: Always provide explicit type for numbers:
```dart
'price': FieldConfig(label: 'Price', type: FieldType.double),
```

### 3. DateTime Context
**Current**: Uses global ScaffoldMessengerKey
**Limitation**: May not work in all widget tree configurations

**Workaround**: Provide custom DateTime widget:
```dart
'date': FieldConfig(
  label: 'Date',
  widget: (context, value, onChanged, mode) {
    // Use context-aware date picker
  },
),
```

---

## Future Enhancements

### Planned for v2.1.0
- [ ] Recursive nested form rendering
- [ ] Advanced list editing (reorder, complex types)
- [ ] Async field validators
- [ ] Field dependencies (show field B if field A has value X)

### Planned for v2.2.0
- [ ] Field groups and sections
- [ ] Multi-step forms / wizard support
- [ ] Theme customization
- [ ] Accessibility improvements

### Under Consideration
- [ ] Integration with popular form packages (flutter_form_builder, etc.)
- [ ] GraphQL schema integration
- [ ] JSON Schema support
- [ ] Excel/CSV import/export

---

## AI Agent Instructions

When working with this codebase:

### 1. Understand the Philosophy
- **No code generation** - Everything happens at runtime
- **Type inference first** - Explicit types only when needed
- **Map-based data** - Not typed models
- **Simplicity over features** - Only add complexity when clearly valuable

### 2. Code Modification Guidelines
- Keep changes minimal and focused
- Maintain backward compatibility when possible
- Update both AGENTS.md and README.md for any changes
- Run `flutter analyze` before committing
- Test with example app

### 3. Extension Pattern
When adding features:
1. Consider if it fits the runtime philosophy
2. Add to appropriate layer (schema, widget, or field widgets)
3. Provide example in example_app.dart
4. Document in README.md with code example
5. Update AGENTS.md with architecture notes

### 4. Breaking Changes
If a breaking change is needed:
1. Document rationale in AGENTS.md under "Architectural Evolution"
2. Bump major version
3. Provide migration guide
4. Update all examples

### 5. Related Patterns
This module is different from StateSync:
- **StateSync**: InheritedWidget for state propagation down tree
- **DynamicFormWidget**: GlobalKey for imperative state access

Don't mix patterns without clear justification.

---

## Dependencies

### Runtime
- `flutter` - Core framework
- `intl` ^0.19.0 - Date/time formatting

### Development
None (no code generation in v2.0.0)

---

## Version History

### v2.2.0 (2025-12-04) - Auto-Discovery & Optional FieldConfigs
**BREAKING CHANGES**:
- `fieldConfigs` is now optional - fields are auto-discovered from `value` map
- `fieldConfigs` now only provides overrides, not the list of fields to show

**Features**:
- Auto-discovery of all fields from `value.keys`
- Added `fieldConfig` parameter to `DynamicFormTheme` for project-wide defaults
- Config priority: `widget.fieldConfigs[key]` > `theme.fieldConfig` > `FieldConfig()`

**API Changes**:
- `DynamicFormWidget.fieldConfigs` is now optional (nullable)
- `DynamicFormTheme` has new `fieldConfig` parameter

### v2.1.0 (2025-12-04) - Custom Default Widgets & Auto-Labeling
**Features**:
- Added `DynamicFormTheme` InheritedWidget for project-wide customization
- Custom default widget builders per `FieldType` via `DynamicFormTheme.builders`
- Auto-labeling with `LabelResolver` (widget-level and theme-level)
- Made `FieldConfig.label` optional (nullable)
- Added `toSnakeCase()` and `humanizeFieldName()` helper functions
- Added `DynamicFormFieldBuilder` typedef

**New Files**:
- `theme.dart` - DynamicFormTheme InheritedWidget

**API Changes**:
- `FieldConfig.label` is now optional
- `DynamicFormWidget` has new `labelResolver` parameter

### v2.0.0 (2025-01-25) - Map-Based Refactor
**BREAKING CHANGES**:
- Removed code generation entirely
- Renamed `AutoWidgetBuilder<T>` → `DynamicFormWidget`
- Changed from typed models to `Map<String, dynamic>`
- Removed all annotations (@AutoFormModel, @UseWidget, @FieldConfig)

**Features**:
- Type inference from runtime values
- Custom widgets via function builders
- Dot notation for nested fields
- Simplified, tree-shake friendly architecture

**Migration**: See Migration Guide above

### v1.0.0 (2025-01-25) - Initial Release (Deprecated)
- Generic AutoWidgetBuilder<T>
- Code generation with build_runner
- Annotation-based configuration
- Tree-shaking issues on web

**Status**: Deprecated, do not use for new projects

---

## Conclusion

DynamicFormWidget v2.0.0 represents a fundamental shift from compile-time code generation to runtime flexibility. The Map-based approach trades marginal compile-time type safety for significant gains in:
- Development velocity (no build step)
- Reliability (no tree-shaking issues)
- Simplicity (less code, less complexity)
- Flexibility (works with any data structure)

This architecture is production-ready and suitable for forms ranging from simple settings screens to complex multi-step wizards.
