import 'package:flutter/material.dart';
import 'field_widget.dart';
import 'schema.dart';

/// InheritedWidget that provides custom default field widget builders, label resolution,
/// and default field configurations per key.
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
///   fieldConfigs: {
///     'id': FieldConfig(type: FieldType.string, required: false),
///     'createdAt': FieldConfig(type: FieldType.dateTime),
///     'updatedAt': FieldConfig(type: FieldType.dateTime),
///   },
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

  /// Default field configurations per key.
  /// When a field key matches one in this map, that FieldConfig is used as the default.
  /// Priority: widget.fieldConfigs[key] > theme.fieldConfigs[key] > FieldConfig()
  final Map<String, FieldConfig>? fieldConfigs;

  const DynamicFormTheme({Key? key, this.builders, this.labelResolver, this.fieldConfigs, required Widget child})
    : super(key: key, child: child);

  /// Gets the custom builder for a field type, or null if none registered.
  DynamicFormFieldBuilder? builderFor(FieldType type) => builders?[type];

  /// Gets the nearest DynamicFormTheme from the widget tree, or null.
  static DynamicFormTheme? of(BuildContext context) {
    return context.dependOnInheritedWidgetOfExactType<DynamicFormTheme>();
  }

  @override
  bool updateShouldNotify(DynamicFormTheme oldWidget) {
    return builders != oldWidget.builders || labelResolver != oldWidget.labelResolver || fieldConfigs != oldWidget.fieldConfigs;
  }
}
