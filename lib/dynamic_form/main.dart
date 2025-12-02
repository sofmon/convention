import 'dart:async';
import 'package:flutter/material.dart';
import 'field_widget.dart';
import 'schema.dart';
import 'field_widgets/default_field_widgets.dart';

/// A widget that generates dynamic view and edit UIs from Map data.
///
/// Given a [Map<String, dynamic>] and a [Map<String, FieldConfig>], DynamicFormWidget can:
/// 1. Display the data in view mode
/// 2. Provide editable form fields in edit mode
/// 3. Collect edited values and produce an updated Map on save
///
/// Supports nested fields using dot notation, e.g., "user.address.street"
///
/// Usage:
/// ```dart
/// final key = GlobalKey<DynamicFormWidgetState>();
///
/// DynamicFormWidget(
///   key: key,
///   value: {'name': 'John', 'age': 30},
///   fieldConfigs: {
///     'name': FieldConfig(label: 'Name'),
///     'age': FieldConfig(label: 'Age', type: FieldType.int),
///   },
///   mode: AutoWidgetMode.edit,
/// );
///
/// // Later, to save:
/// final updated = await key.currentState!.save();
/// ```
class DynamicFormWidget extends StatefulWidget {
  /// The current data to display/edit
  final Map<String, dynamic> value;

  /// Configuration for each field
  /// Keys can use dot notation for nested fields (e.g., "user.address.street")
  final Map<String, FieldConfig> fieldConfigs;

  /// The mode (view or edit)
  final AutoWidgetMode mode;

  /// Optional callback when value changes in edit mode
  final void Function(Map<String, dynamic> updated)? onChanged;

  /// Custom layout builder (optional)
  final Widget Function(BuildContext context, List<Widget> fieldWidgets)? layoutBuilder;

  const DynamicFormWidget({
    Key? key,
    required this.value,
    required this.fieldConfigs,
    this.mode = AutoWidgetMode.view,
    this.onChanged,
    this.layoutBuilder,
  }) : super(key: key);

  @override
  State<DynamicFormWidget> createState() => DynamicFormWidgetState();
}

/// State for DynamicFormWidget with save and validation support
class DynamicFormWidgetState extends State<DynamicFormWidget> {
  late Map<String, dynamic> _fieldValues;
  final Map<String, GlobalKey<FormFieldState>> _fieldKeys = {};
  final Map<String, String> _validationErrors = {};

  @override
  void initState() {
    super.initState();
    _initializeFieldValues();
  }

  @override
  void didUpdateWidget(DynamicFormWidget oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (oldWidget.value != widget.value) {
      _initializeFieldValues();
    }
  }

  void _initializeFieldValues() {
    _fieldValues = Map<String, dynamic>.from(widget.value);
  }

  /// Gets a value from the data using a field path (supports dot notation)
  dynamic _getValueByPath(String path) {
    final parts = path.split('.');
    dynamic current = _fieldValues;

    for (final part in parts) {
      if (current is Map) {
        current = current[part];
      } else {
        return null;
      }
    }

    return current;
  }

  /// Sets a value in the data using a field path (supports dot notation)
  void _setValueByPath(String path, dynamic value) {
    final parts = path.split('.');
    dynamic current = _fieldValues;

    for (int i = 0; i < parts.length - 1; i++) {
      final part = parts[i];
      if (current is Map) {
        if (!current.containsKey(part) || current[part] is! Map) {
          current[part] = <String, dynamic>{};
        }
        current = current[part];
      } else {
        return;
      }
    }

    if (current is Map) {
      current[parts.last] = value;
    }
  }

  void _onFieldChanged(String fieldPath, dynamic value) {
    setState(() {
      _setValueByPath(fieldPath, value);
      _validationErrors.remove(fieldPath);
    });

    if (widget.onChanged != null) {
      widget.onChanged!(Map<String, dynamic>.from(_fieldValues));
    }
  }

  /// Validates all fields and returns a ValidationResult
  ValidationResult validate() {
    final errors = <String, String>{};

    for (final entry in widget.fieldConfigs.entries) {
      final fieldPath = entry.key;
      final config = entry.value;
      final value = _getValueByPath(fieldPath);

      // Check required fields
      if (config.required && value == null) {
        errors[fieldPath] = config.validationError ?? '${config.label} is required';
      }

      // Check FormField validation if present
      final fieldKey = _fieldKeys[fieldPath];
      if (fieldKey?.currentState != null) {
        if (!fieldKey!.currentState!.validate()) {
          // Error is already shown by the FormField
          errors[fieldPath] = config.validationError ?? 'Invalid value';
        }
      }
    }

    return errors.isEmpty
        ? ValidationResult.success()
        : ValidationResult.failure(errors);
  }

  /// Saves the current field values and returns an updated Map.
  ///
  /// Validates all fields first. If validation fails, throws a [ValidationException]
  /// and displays errors in the UI.
  ///
  /// Returns a new Map with updated values.
  Future<Map<String, dynamic>> save() async {
    final validationResult = validate();

    if (!validationResult.isValid) {
      setState(() {
        _validationErrors.addAll(validationResult.errors);
      });
      throw ValidationException(validationResult.errors);
    }

    final result = Map<String, dynamic>.from(_fieldValues);

    if (widget.onChanged != null) {
      widget.onChanged!(result);
    }

    return result;
  }

  /// Resets all field values to the original widget.value
  void reset() {
    setState(() {
      _initializeFieldValues();
      _validationErrors.clear();
      // Reset FormField states
      for (final key in _fieldKeys.values) {
        key.currentState?.reset();
      }
    });
  }

  @override
  Widget build(BuildContext context) {
    final fieldWidgets = <Widget>[];

    // Build widgets in the order they appear in fieldConfigs
    for (final entry in widget.fieldConfigs.entries) {
      final fieldPath = entry.key;
      final config = entry.value;
      final fieldWidget = _buildFieldWidget(fieldPath, config);
      fieldWidgets.add(fieldWidget);
    }

    if (widget.layoutBuilder != null) {
      return widget.layoutBuilder!(context, fieldWidgets);
    }

    // Default layout: vertical list with padding
    return ListView.separated(
      shrinkWrap: true,
      physics: const NeverScrollableScrollPhysics(),
      itemCount: fieldWidgets.length,
      separatorBuilder: (context, index) => const SizedBox(height: 16),
      itemBuilder: (context, index) => fieldWidgets[index],
    );
  }

  Widget _buildFieldWidget(String fieldPath, FieldConfig config) {
    final value = _getValueByPath(fieldPath);
    final error = _validationErrors[fieldPath];

    // Use custom widget if provided
    if (config.widget != null) {
      return _wrapWithPadding(
        config.widget!(
          context,
          value,
          (newValue) => _onFieldChanged(fieldPath, newValue),
          widget.mode,
        ),
      );
    }

    // Determine the effective type
    final effectiveType = config.getEffectiveType(value);

    if (effectiveType == null) {
      // Cannot determine type
      return _wrapWithPadding(
        Text(
          'Unknown field type for "${config.label}"',
          style: TextStyle(color: Colors.red),
        ),
      );
    }

    // Build default widget based on type
    return _wrapWithPadding(
      DefaultFieldWidgets.buildWidgetByType(
        type: effectiveType,
        label: config.label,
        value: value,
        mode: widget.mode,
        onChanged: (newValue) => _onFieldChanged(fieldPath, newValue),
        required: config.required,
        hint: config.hint,
        validationError: error ?? config.validationError,
        enumValues: config.enumValues,
        nestedFields: config.nestedFields,
        fieldKey: _getFieldKey(fieldPath),
      ),
    );
  }

  Widget _wrapWithPadding(Widget child) {
    // Add some padding if not already present
    return child;
  }

  GlobalKey<FormFieldState> _getFieldKey(String fieldPath) {
    return _fieldKeys.putIfAbsent(
      fieldPath,
      () => GlobalKey<FormFieldState>(),
    );
  }
}

/// Exception thrown when validation fails during save
class ValidationException implements Exception {
  final Map<String, String> errors;

  ValidationException(this.errors);

  @override
  String toString() {
    return 'ValidationException: ${errors.entries.map((e) => '${e.key}: ${e.value}').join(', ')}';
  }
}

/// Extension methods for convenient access to DynamicFormWidget
extension DynamicFormWidgetExtension on BuildContext {
  /// Gets the DynamicFormWidgetState from the widget tree
  DynamicFormWidgetState? findDynamicFormWidget() {
    return findAncestorStateOfType<DynamicFormWidgetState>();
  }
}
