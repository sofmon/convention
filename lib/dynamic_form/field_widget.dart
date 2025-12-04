import 'package:flutter/material.dart';

/// The mode in which AutoWidgetBuilder operates
enum AutoWidgetMode {
  /// Display mode - shows field values as read-only widgets
  view,

  /// Edit mode - shows interactive form fields for editing
  edit,
}

/// Base interface for custom field widgets.
///
/// Custom field widgets specified via @UseWidget must implement this interface.
/// The widget is responsible for rendering a field in both view and edit modes
/// and tracking the current value.
abstract class FieldWidget<T> extends StatefulWidget {
  /// The current value of the field
  final T? value;

  /// The mode (view or edit)
  final AutoWidgetMode mode;

  /// Label text for the field
  final String label;

  /// Hint text (used in edit mode)
  final String? hint;

  /// Callback when value changes (edit mode)
  final ValueChanged<T?>? onChanged;

  /// Whether the field is required
  final bool required;

  /// Custom validation error message
  final String? validationError;

  const FieldWidget({
    Key? key,
    required this.value,
    required this.mode,
    required this.label,
    this.hint,
    this.onChanged,
    this.required = true,
    this.validationError,
  }) : super(key: key);
}

/// State for FieldWidget with validation support
abstract class FieldWidgetState<T, W extends FieldWidget<T>> extends State<W> {
  /// Validates the current value
  ///
  /// Returns null if valid, or an error message if invalid
  String? validate();

  /// Gets the current value
  T? getValue();
}

/// Builder function for creating field widgets
typedef FieldWidgetBuilder<T> = Widget Function({
  required T? value,
  required AutoWidgetMode mode,
  required String label,
  String? hint,
  ValueChanged<T?>? onChanged,
  bool required,
  String? validationError,
});

/// Builder function for creating custom default field widgets by type.
/// Used by DynamicFormTheme to provide project-wide custom widget implementations.
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

/// Result of validation
class ValidationResult {
  final bool isValid;
  final Map<String, String> errors; // field name -> error message

  ValidationResult({
    required this.isValid,
    required this.errors,
  });

  factory ValidationResult.success() {
    return ValidationResult(isValid: true, errors: {});
  }

  factory ValidationResult.failure(Map<String, String> errors) {
    return ValidationResult(isValid: false, errors: errors);
  }
}
