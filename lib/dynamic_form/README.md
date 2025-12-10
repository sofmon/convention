# DynamicFormWidget

DynamicFormWidget is a Flutter widget that automatically creates view and edit UIs from Map data. Define your data as `Map<String, dynamic>`, configure fields with `FieldConfig`, and get fully functional form interfaces with zero boilerplate.

## Features

- **Map-Based**: Works with `Map<String, dynamic>` - no code generation required
- **Auto-Discovery**: Automatically discovers fields from value map - no fieldConfigs required
- **Type Inference**: Automatically infers field types from values
- **Type-Safe**: Explicit type specification available when needed
- **Extensible**: Custom field widgets via widget builders
- **Zero Boilerplate**: Minimal configuration needed
- **Built-in Widgets**: Default implementations for all common types
- **Validation**: Built-in validation with custom error messages
- **Mode Switching**: Seamlessly switch between view and edit modes
- **Nested Fields**: Support for dot notation paths (e.g., "user.address.street")
- **Custom Default Widgets**: Replace default widgets per type via `DynamicFormTheme`
- **Auto-Labeling**: Automatic label resolution via ARB resources or humanized field names
- **Selective Overrides**: Override only specific fields while auto-discovering others

## Quick Start

### 1. Add Dependencies

```yaml
# pubspec.yaml
dependencies:
  flutter:
    sdk: flutter
  intl: ^0.19.0  # For date formatting
```

### 2. Define Your Data

```dart
import 'package:convention/builder/schema.dart';

final userProfile = {
  'name': 'John Doe',
  'email': 'john@example.com',
  'age': 30,
  'isActive': true,
};
```

### 3. Use DynamicFormWidget (Zero Config)

```dart
import 'package:flutter/material.dart';
import 'package:convention/builder/main.dart';
import 'package:convention/builder/field_widget.dart';

class ProfileScreen extends StatefulWidget {
  @override
  State<ProfileScreen> createState() => _ProfileScreenState();
}

class _ProfileScreenState extends State<ProfileScreen> {
  final _formKey = GlobalKey<DynamicFormWidgetState>();
  var _mode = AutoWidgetMode.view;

  Map<String, dynamic> _profile = {
    'name': 'John Doe',
    'email': 'john@example.com',
    'age': 30,
    'isActive': true,
  };

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: Text('Profile'),
        actions: [
          if (_mode == AutoWidgetMode.view)
            IconButton(
              icon: Icon(Icons.edit),
              onPressed: () => setState(() => _mode = AutoWidgetMode.edit),
            )
          else
            IconButton(
              icon: Icon(Icons.save),
              onPressed: _saveProfile,
            ),
        ],
      ),
      body: Padding(
        padding: EdgeInsets.all(16),
        child: DynamicFormWidget(
          key: _formKey,
          value: _profile,
          // No fieldConfigs needed! All fields auto-discovered
          mode: _mode,
        ),
      ),
    );
  }

  Future<void> _saveProfile() async {
    try {
      final updated = await _formKey.currentState!.save();
      setState(() {
        _profile = updated;
        _mode = AutoWidgetMode.view;
      });
    } catch (e) {
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text('Validation failed: $e')),
      );
    }
  }
}
```

### 4. Override Specific Fields (Optional)

```dart
// Only override fields that need custom configuration
final _fieldConfigs = {
  'age': const FieldConfig(type: FieldType.int),  // Explicit type
  'email': const FieldConfig(hint: 'Enter your email'),  // Add hint
};

DynamicFormWidget(
  key: _formKey,
  value: _profile,
  fieldConfigs: _fieldConfigs,  // Only overrides 'age' and 'email'
  mode: _mode,
)
// 'name' and 'isActive' are auto-discovered with defaults
```

## Supported Field Types

DynamicFormWidget provides default widgets for these types:

| FieldType | View Widget | Edit Widget |
|-----------|-------------|-------------|
| `string` | Text | TextFormField |
| `int` | Text | TextFormField (numeric) |
| `double` | Text | TextFormField (decimal) |
| `bool` | Icon (check/cancel) | Switch |
| `dateTime` | Formatted text | DatePicker + TimePicker |
| `enumType` | Enum name | DropdownButtonFormField |
| `list` | Bullet list | List with add/remove |
| `nested` | Nested display | Nested form fields |

## Type Inference

One of the key features is automatic type inference. If you don't specify a `type` in `FieldConfig`, the widget will infer it from the value:

```dart
// Type is inferred as FieldType.string
'name': const FieldConfig(label: 'Name'),

// Type is explicitly set
'age': const FieldConfig(label: 'Age', type: FieldType.int),
```

Inference works for:
- `String` → `FieldType.string`
- `int` → `FieldType.int`
- `double` → `FieldType.double`
- `bool` → `FieldType.bool`
- `DateTime` → `FieldType.dateTime`
- `List` → `FieldType.list`
- `Map` → `FieldType.nested`

For enums, you must specify the type explicitly and provide `enumValues`.

## Customization

### Custom Labels and Hints

```dart
final fieldConfigs = {
  'name': const FieldConfig(
    label: 'Product Name',
    hint: 'Enter the product name',
    required: true,
  ),
  'price': const FieldConfig(
    label: 'Price (USD)',
    hint: 'Enter price in dollars',
    type: FieldType.double,
    validationError: 'Price must be specified',
  ),
};
```

### Custom Field Widgets

Use the `widget` parameter to provide a custom widget builder:

```dart
final fieldConfigs = {
  'phoneNumber': FieldConfig(
    label: 'Phone Number',
    widget: (context, value, onChanged, mode) {
      if (mode == AutoWidgetMode.view) {
        return Text(value ?? '—');
      }
      return TextFormField(
        initialValue: value,
        decoration: InputDecoration(
          labelText: 'Phone Number',
          hintText: 'Enter phone number',
        ),
        onChanged: onChanged,
        keyboardType: TextInputType.phone,
      );
    },
  ),
};
```

### Nested Fields with Dot Notation

Support for nested objects using dot notation:

```dart
final data = {
  'user': {
    'address': {
      'street': '123 Main St',
      'city': 'Springfield',
    },
  },
};

final fieldConfigs = {
  'user.address.street': const FieldConfig(label: 'Street'),
  'user.address.city': const FieldConfig(label: 'City'),
};
```

### Enum Fields

```dart
enum AccountType { free, premium, enterprise }

final data = {
  'accountType': AccountType.premium,
};

final fieldConfigs = {
  'accountType': FieldConfig(
    label: 'Account Type',
    type: FieldType.enumType,
    enumValues: AccountType.values,
  ),
};
```

### Custom Layout

```dart
DynamicFormWidget(
  value: profile,
  fieldConfigs: fieldConfigs,
  mode: AutoWidgetMode.edit,
  layoutBuilder: (context, fieldWidgets) {
    return Column(
      children: [
        Row(
          children: [
            Expanded(child: fieldWidgets[0]),
            SizedBox(width: 16),
            Expanded(child: fieldWidgets[1]),
          ],
        ),
        ...fieldWidgets.skip(2),
      ],
    );
  },
)
```

### Custom Default Widgets via DynamicFormTheme

Replace default widgets for specific field types across your entire app:

```dart
import 'package:convention/dynamic_form/theme.dart';

// Wrap your app with DynamicFormTheme
DynamicFormTheme(
  builders: {
    FieldType.bool: ({required label, required value, required mode, required onChanged, ...}) {
      // Custom bool widget with chip-style display
      if (mode == AutoWidgetMode.view) {
        return Chip(
          label: Text(value == true ? 'Active' : 'Inactive'),
          backgroundColor: value == true ? Colors.green.shade100 : Colors.grey.shade200,
        );
      }
      return SwitchListTile(
        value: value ?? false,
        onChanged: onChanged,
        title: Text(label),
      );
    },
    FieldType.dateTime: ({required label, ...}) => MyCustomDatePicker(...),
  },
  child: MyApp(),
)
```

**Widget Priority Order:**
1. `FieldConfig.widget` (per-field custom widget) - HIGHEST
2. `DynamicFormTheme.builders[type]` (custom default by type)
3. Built-in default widget - LOWEST

### Auto-Labeling

Labels can be automatically resolved, eliminating the need to specify them manually:

```dart
// Labels are resolved in this order:
// 1. FieldConfig.label (if set)
// 2. widget.labelResolver (if set and returns non-null)
// 3. DynamicFormTheme.labelResolver (if set and returns non-null)
// 4. Humanized field name: "firstName" -> "First Name"

// Minimal config - labels auto-generated
DynamicFormWidget(
  value: {'firstName': 'John', 'lastName': 'Doe'},
  fieldConfigs: {
    'firstName': FieldConfig(),  // Auto-label: "First Name"
    'lastName': FieldConfig(),   // Auto-label: "Last Name"
  },
)

// With ARB resources
DynamicFormWidget(
  value: {'firstName': 'John'},
  fieldConfigs: {'firstName': FieldConfig()},
  labelResolver: (fieldNameSnakeCase) {
    // fieldNameSnakeCase is 'first_name'
    final key = 'form_$fieldNameSnakeCase';
    return AppLocalizations.of(context)?.translate(key);
  },
)

// Project-wide via DynamicFormTheme
DynamicFormTheme(
  labelResolver: (fieldNameSnakeCase) {
    return myArbLookup('form_$fieldNameSnakeCase');
  },
  child: MyApp(),
)

// Key-specific defaults via DynamicFormTheme.fieldConfigs
DynamicFormTheme(
  fieldConfigs: {
    'id': const FieldConfig(type: FieldType.string, required: false),
    'createdAt': const FieldConfig(type: FieldType.dateTime),
    'updatedAt': const FieldConfig(type: FieldType.dateTime),
    'name': const FieldConfig(hint: 'Enter name'),
  },
  child: DynamicFormWidget(
    value: {'id': '123', 'name': 'John', 'createdAt': DateTime.now()},
    // 'id', 'name', 'createdAt' get config from theme.fieldConfigs
    mode: AutoWidgetMode.edit,
  ),
)
```

**Field Name Conversion:**
- camelCase is converted to snake_case: `firstName` → `first_name`
- Dot notation uses the last segment: `user.address.streetName` → `street_name`
- Humanization: `first_name` → `First Name`

## Validation

DynamicFormWidget provides automatic validation:

1. **Required Fields**: Fields with `required: true` are validated
2. **Type Validation**: Numeric fields validate input format
3. **Custom Validation**: Add custom validators via `validationError`

```dart
final fieldConfigs = {
  'email': const FieldConfig(
    label: 'Email',
    required: true,
    validationError: 'Email is required',
  ),
  'bio': const FieldConfig(
    label: 'Bio',
    required: false,
  ),
};
```

Validation occurs when calling `save()`:

```dart
try {
  final updated = await formKey.currentState!.save();
  // Success - use updated Map
} on ValidationException catch (e) {
  // Handle validation errors
  print('Validation errors: ${e.errors}');
}
```

## API Reference

### DynamicFormWidget

```dart
DynamicFormWidget({
  Key? key,
  required Map<String, dynamic> value,           // The data to display/edit
  Map<String, FieldConfig>? fieldConfigs,        // Optional field overrides
  AutoWidgetMode mode = view,                     // view or edit mode
  Function(Map<String, dynamic>)? onChanged,      // Called when value changes
  Widget Function(...)? layoutBuilder,            // Custom layout
  LabelResolver? labelResolver,                   // Auto-labeling
})
```

**Field Discovery**: All fields are auto-discovered from `value.keys`. The `fieldConfigs` parameter only provides overrides for specific fields.

### DynamicFormWidgetState

```dart
class DynamicFormWidgetState {
  Future<Map<String, dynamic>> save();   // Validates and returns updated Map
  ValidationResult validate();           // Validates without saving
  void reset();                          // Resets to original values
}
```

### FieldConfig

```dart
class FieldConfig {
  final String? label;                   // Display label (null = auto-resolve)
  final FieldType? type;                 // Field type (null = infer from value)
  final bool required;                   // Whether field is required
  final String? hint;                    // Hint text for edit mode
  final String? validationError;         // Custom validation error message
  final List<dynamic>? enumValues;       // For enum fields
  final Map<String, FieldConfig>? nestedFields;  // For nested objects
  final Widget Function(...)? widget;    // Custom widget builder
}
```

### DynamicFormTheme

```dart
class DynamicFormTheme extends InheritedWidget {
  final Map<FieldType, DynamicFormFieldBuilder>? builders;  // Custom default widgets
  final LabelResolver? labelResolver;                       // Project-wide label resolver
  final Map<String, FieldConfig>? fieldConfigs;             // Default configs per field key

  static DynamicFormTheme? of(BuildContext context);
  DynamicFormFieldBuilder? builderFor(FieldType type);
}
```

**Config Priority**: `widget.fieldConfigs[key]` > `theme.fieldConfigs[key]` > `FieldConfig()`

### LabelResolver

```dart
/// Resolves a label for the given field name (in snake_case).
/// Return null to fall back to humanized field name.
typedef LabelResolver = String? Function(String fieldNameSnakeCase);
```

### DynamicFormFieldBuilder

```dart
typedef DynamicFormFieldBuilder = Widget Function({
  required String label,
  required dynamic value,
  required AutoWidgetMode mode,
  required ValueChanged<dynamic> onChanged,
  bool required,
  String? hint,
  String? validationError,
  List<dynamic>? enumValues,
  dynamic nestedFields,
  GlobalKey<FormFieldState>? fieldKey,
});
```

### FieldType Enum

```dart
enum FieldType {
  string,
  int,
  double,
  bool,
  dateTime,
  enumType,
  list,
  nested,
}
```

### Helper Functions

```dart
/// Converts camelCase to snake_case. For dot notation, uses last segment.
String toSnakeCase(String input);  // "firstName" -> "first_name"

/// Humanizes snake_case to title case with spaces.
String humanizeFieldName(String snakeCase);  // "first_name" -> "First Name"
```

## Examples

See [example/](example/) directory for complete working examples:

- `models.dart` - Sample enum types
- `example_app.dart` - Full Flutter app demonstrating usage

Run the example:

```bash
flutter run lib/util/builder/example/example_app.dart
```

## Best Practices

1. **Use Type Inference**: Let the widget infer types when possible
2. **Explicit Types for Ambiguity**: Use explicit types for enums, numbers that could be int or double
3. **Meaningful Labels**: Provide clear labels for better UX
4. **Validate Early**: Test validation logic as you build forms
5. **Custom Widgets Sparingly**: Use only when default widgets don't fit

## Architecture

DynamicFormWidget follows a clean, runtime-based architecture:

1. **Schema Layer** (`schema.dart`): `FieldConfig` and `FieldType` definitions
2. **Widget Layer** (`main.dart`): Core `DynamicFormWidget` implementation
3. **Field Widgets** (`field_widgets/`): Type-specific UI components

No code generation is needed - everything works at runtime using reflection-free Map access and type inference.

## Migration from AutoWidgetBuilder

If you were using the old `AutoWidgetBuilder<T>` with code generation:

**Before** (with code generation):
```dart
@AutoFormModel()
class UserProfile {
  final String name;
  final int age;
  const UserProfile({required this.name, required this.age});
}

// Required: flutter pub run build_runner build

AutoWidgetBuilder<UserProfile>(
  value: UserProfile(name: 'John', age: 30),
  mode: AutoWidgetMode.edit,
)
```

**After** (Map-based, no code generation):
```dart
final userProfile = {
  'name': 'John',
  'age': 30,
};

final fieldConfigs = {
  'name': const FieldConfig(label: 'Name'),
  'age': const FieldConfig(label: 'Age', type: FieldType.int),
};

DynamicFormWidget(
  value: userProfile,
  fieldConfigs: fieldConfigs,
  mode: AutoWidgetMode.edit,
)
```

Benefits:
- No code generation step
- No tree-shaking issues
- Simpler, more flexible
- Runtime type inference
- Easier to debug

## Version History

### v2.3.0 (2025-12-08)
- **BREAKING**: `DynamicFormTheme.fieldConfig` replaced with `DynamicFormTheme.fieldConfigs`
- Key-specific default configurations: specify defaults for common field names (e.g., 'id', 'createdAt')
- Config priority: `widget.fieldConfigs[key]` > `theme.fieldConfigs[key]` > `FieldConfig()`

### v2.2.0 (2025-12-04)
- **BREAKING**: `fieldConfigs` is now optional - fields are auto-discovered from `value` map
- **BREAKING**: `fieldConfigs` now only provides overrides, not the list of fields to show
- Added `fieldConfig` parameter to `DynamicFormTheme` for project-wide defaults
- All fields from `value.keys` are now shown by default

### v2.1.0 (2025-12-04)
- Added `DynamicFormTheme` InheritedWidget for project-wide customization
- Added custom default widget builders per `FieldType`
- Added auto-labeling with `LabelResolver`
- Made `FieldConfig.label` optional (nullable)
- Added `toSnakeCase()` and `humanizeFieldName()` helper functions
- Added `labelResolver` parameter to `DynamicFormWidget`

### v2.0.0 (2025-01-25)
- **BREAKING**: Complete refactor from code generation to Map-based approach
- Removed `@AutoFormModel` annotation and code generation
- Renamed `AutoWidgetBuilder<T>` to `DynamicFormWidget`
- Added type inference from runtime values
- Added support for custom widget builders via `FieldConfig.widget`
- Added dot notation support for nested fields
- Simplified API and removed tree-shaking complexity

### v1.0.0 (2025-01-25)
- Initial implementation with code generation
- Support for primitive types, DateTime, enum, List, nested objects
- Custom field widgets via `@UseWidget`
- Field configuration via `@FieldConfig`

## License

Part of the convention project.
