import 'package:flutter/material.dart';
import 'field_widget.dart';

/// Resolves a label for the given field name (in snake_case).
/// Return null to fall back to humanized field name.
typedef LabelResolver = String? Function(String fieldNameSnakeCase);

/// Converts camelCase/PascalCase to snake_case.
/// For dot notation paths, uses only the last segment.
///
/// Examples:
/// - "userName" → "user_name"
/// - "firstName" → "first_name"
/// - "user.address.streetName" → "street_name"
String toSnakeCase(String input) {
  // Handle dot notation: take only the last part
  final fieldName = input.contains('.') ? input.split('.').last : input;

  // Convert camelCase to snake_case
  return fieldName
      .replaceAllMapped(
        RegExp(r'[A-Z]'),
        (match) => '_${match.group(0)!.toLowerCase()}',
      )
      .replaceAll(RegExp(r'^_'), ''); // Remove leading underscore
}

/// Humanizes snake_case to title case with spaces.
///
/// Examples:
/// - "user_name" → "User Name"
/// - "first_name" → "First Name"
/// - "is_active" → "Is Active"
String humanizeFieldName(String snakeCase) {
  return snakeCase
      .split('_')
      .map((word) =>
          word.isEmpty ? '' : '${word[0].toUpperCase()}${word.substring(1)}')
      .join(' ');
}

/// Field types supported by DynamicFormWidget
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

/// Configuration for a single field in the dynamic form.
///
/// Fields can be specified with a path using dot notation for nested objects,
/// e.g., "user.address.street"
class FieldConfig {
  /// Display label for the field.
  /// If null, label will be resolved automatically via LabelResolver or humanized field name.
  final String? label;

  /// The field type - if null, will be inferred from the value
  final FieldType? type;

  /// Whether the field is required
  final bool required;

  /// Hint text for edit mode
  final String? hint;

  /// Custom validation error message
  final String? validationError;

  /// For enum fields, the list of possible values
  final List<dynamic>? enumValues;

  /// For nested object fields, the configuration of nested fields
  final Map<String, FieldConfig>? nestedFields;

  /// Custom widget builder for this field
  ///
  /// Parameters:
  /// - BuildContext: the build context
  /// - dynamic value: the current field value
  /// - ValueChanged<dynamic>: callback to update the value
  /// - AutoWidgetMode: current mode (view or edit)
  final Widget Function(
    BuildContext context,
    dynamic value,
    ValueChanged<dynamic> onChanged,
    AutoWidgetMode mode,
  )? widget;

  /// Creates a field configuration
  const FieldConfig({
    this.label,
    this.type,
    this.required = true,
    this.hint,
    this.validationError,
    this.enumValues,
    this.nestedFields,
    this.widget,
  });

  /// Infers the FieldType from a runtime value
  static FieldType? inferType(dynamic value) {
    if (value == null) return null;
    if (value is String) return FieldType.string;
    if (value is int) return FieldType.int;
    if (value is double) return FieldType.double;
    if (value is bool) return FieldType.bool;
    if (value is DateTime) return FieldType.dateTime;
    if (value is List) return FieldType.list;
    if (value is Map) return FieldType.nested;
    // For other types, return null (including enums which need explicit config)
    return null;
  }

  /// Gets the effective type (explicit or inferred)
  FieldType? getEffectiveType(dynamic value) {
    return type ?? inferType(value);
  }
}
