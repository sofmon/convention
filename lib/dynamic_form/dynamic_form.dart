import 'dart:async';
import 'package:flutter/material.dart';
import 'field_widget.dart';
import 'schema.dart';
import 'theme.dart';
import 'field_widgets/default_field_widgets.dart';

/// A widget that generates dynamic view and edit UIs from Map data.
///
/// Given a [Map<String, dynamic>], DynamicFormWidget automatically discovers fields
/// and renders them. Optionally provide [fieldConfigs] to override specific field configurations.
///
/// Features:
/// 1. Auto-discovers fields from value map keys
/// 2. Display data in view mode or edit mode
/// 3. Collect edited values and produce an updated Map on save
/// 4. Override specific fields with custom FieldConfig
///
/// Supports nested fields using dot notation, e.g., "user.address.street"
///
/// Usage:
/// ```dart
/// final key = GlobalKey<DynamicFormWidgetState>();
///
/// // Minimal usage - auto-discovers all fields
/// DynamicFormWidget(
///   key: key,
///   value: {'name': 'John', 'age': 30},
///   mode: AutoWidgetMode.edit,
/// );
///
/// // With overrides for specific fields
/// DynamicFormWidget(
///   key: key,
///   value: {'name': 'John', 'age': 30},
///   fieldConfigs: {
///     'age': FieldConfig(type: FieldType.int),  // Override age config
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

  /// Optional configuration overrides for specific fields.
  /// Keys can use dot notation for nested fields (e.g., "user.address.street").
  /// Fields not in this map will use defaults from DynamicFormTheme.fieldConfig
  /// or the default FieldConfig().
  final Map<String, FieldConfig>? fieldConfigs;

  /// The mode (view or edit)
  final AutoWidgetMode mode;

  /// Optional callback when value changes in edit mode
  final void Function(Map<String, dynamic> updated)? onChanged;

  /// Custom layout builder (optional)
  final Widget Function(BuildContext context, List<Widget> fieldWidgets)? layoutBuilder;

  /// Optional label resolver for this widget.
  /// Takes precedence over DynamicFormTheme.labelResolver.
  /// Called with the snake_case field name. Return null to fall back to humanized name.
  final LabelResolver? labelResolver;

  const DynamicFormWidget({
    Key? key,
    required this.value,
    this.fieldConfigs,
    this.mode = AutoWidgetMode.view,
    this.onChanged,
    this.layoutBuilder,
    this.labelResolver,
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

  /// Gets ALL field keys from value map (always auto-discover)
  Iterable<String> _getFieldKeys() {
    return widget.value.keys;
  }

  /// Gets the effective config for a specific field.
  /// Priority: widget.fieldConfigs[key] > theme.fieldConfigs[key] > default FieldConfig()
  FieldConfig _getFieldConfig(String fieldKey) {
    // Priority 1: Explicit fieldConfig for this field from widget
    if (widget.fieldConfigs != null &&
        widget.fieldConfigs!.containsKey(fieldKey)) {
      return widget.fieldConfigs![fieldKey]!;
    }

    // Priority 2: Theme's fieldConfigs for this key
    final theme = DynamicFormTheme.of(context);
    if (theme?.fieldConfigs != null &&
        theme!.fieldConfigs!.containsKey(fieldKey)) {
      return theme.fieldConfigs![fieldKey]!;
    }

    // Priority 3: Default FieldConfig
    return const FieldConfig();
  }

  /// Resolves the label for a field using the priority chain:
  /// 1. FieldConfig.label (explicit)
  /// 2. widget.labelResolver (per-widget)
  /// 3. DynamicFormTheme.labelResolver (project-wide)
  /// 4. Humanized snake_case fallback
  String _resolveLabel(String fieldPath, FieldConfig config) {
    // 1. Explicit label in config takes highest priority
    if (config.label != null) {
      return config.label!;
    }

    // Convert field path to snake_case for resolver
    final snakeCaseName = toSnakeCase(fieldPath);

    // 2. Widget-level labelResolver
    if (widget.labelResolver != null) {
      final resolved = widget.labelResolver!(snakeCaseName);
      if (resolved != null) {
        return resolved;
      }
    }

    // 3. Theme-level labelResolver
    final theme = DynamicFormTheme.of(context);
    if (theme?.labelResolver != null) {
      final resolved = theme!.labelResolver!(snakeCaseName);
      if (resolved != null) {
        return resolved;
      }
    }

    // 4. Fallback: humanize snake_case field name
    return humanizeFieldName(snakeCaseName);
  }

  /// Validates all fields and returns a ValidationResult
  ValidationResult validate() {
    final errors = <String, String>{};

    for (final fieldPath in _getFieldKeys()) {
      final config = _getFieldConfig(fieldPath);
      final value = _getValueByPath(fieldPath);
      final label = _resolveLabel(fieldPath, config);

      // Check required fields
      if (config.required && value == null) {
        errors[fieldPath] = config.validationError ?? '$label is required';
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

    // Build widgets for all fields in value map (auto-discovery)
    for (final fieldPath in _getFieldKeys()) {
      final config = _getFieldConfig(fieldPath);
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
    final label = _resolveLabel(fieldPath, config);

    // Priority 1: Use custom widget if provided in FieldConfig (highest priority)
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
          'Unknown field type for "$label"',
          style: TextStyle(color: Colors.red),
        ),
      );
    }

    // Priority 2: Check for custom builder from DynamicFormTheme
    final theme = DynamicFormTheme.of(context);
    final customBuilder = theme?.builderFor(effectiveType);

    if (customBuilder != null) {
      return _wrapWithPadding(
        customBuilder(
          label: label,
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

    // Priority 3: Fall back to default widget based on type (lowest priority)
    return _wrapWithPadding(
      DefaultFieldWidgets.buildWidgetByType(
        type: effectiveType,
        label: label,
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
