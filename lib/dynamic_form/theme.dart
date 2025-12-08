import 'package:flutter/material.dart';
import 'field_widget.dart';
import 'schema.dart';

/// InheritedWidget that provides custom default field widget builders, label resolution,
/// and default field configuration.
///
/// Wrap your widget tree with DynamicFormTheme to customize how fields
/// are rendered by type without specifying widget builders for each field.
///
/// Example:
/// ```dart
/// DynamicFormTheme(
///   builders: {
///     FieldType.string: ({required label, required value, required mode, required onChanged, ...}) =>
///       MyCustomStringField(label: label, value: value, mode: mode, onChanged: onChanged),
///     FieldType.dateTime: ({required label, ...}) => MyCustomDatePicker(...),
///   },
///   labelResolver: (fieldName) {
///     final key = 'form_$fieldName';
///     return AppLocalizations.of(context)?.translate(key);
///   },
///   fieldConfig: FieldConfig(required: false),  // All fields optional by default
///   child: DynamicFormWidget(...),
/// )
/// ```
class DynamicFormTheme extends InheritedWidget {
  /// Custom widget builders for specific field types.
  /// When a DynamicFormWidget renders a field with a matching type,
  /// it will use this builder instead of the default implementation.
  final Map<FieldType, DynamicFormFieldBuilder>? builders;

  /// Project-wide label resolver.
  /// Called with the snake_case field name. Return null to fall back to humanized name.
  final LabelResolver? labelResolver;

  /// Default field configuration for all fields.
  /// Used when a field doesn't have an explicit FieldConfig in fieldConfigs.
  final FieldConfig? fieldConfig;

  const DynamicFormTheme({Key? key, this.builders, this.labelResolver, this.fieldConfig, required Widget child})
    : super(key: key, child: child);

  /// Gets the custom builder for a field type, or null if none registered.
  DynamicFormFieldBuilder? builderFor(FieldType type) => builders?[type];

  /// Gets the nearest DynamicFormTheme from the widget tree, or null.
  static DynamicFormTheme? of(BuildContext context) {
    return context.dependOnInheritedWidgetOfExactType<DynamicFormTheme>();
  }

  @override
  bool updateShouldNotify(DynamicFormTheme oldWidget) {
    return builders != oldWidget.builders || labelResolver != oldWidget.labelResolver || fieldConfig != oldWidget.fieldConfig;
  }
}
