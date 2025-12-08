import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import '../field_widget.dart';
import '../schema.dart';
import 'package:intl/intl.dart';

/// Default field widget implementations for common Dart types.
///
/// This class provides sensible default widgets for displaying and editing
/// fields based on their type:
/// - String: Text / TextFormField
/// - int, double: Text / TextFormField (numeric)
/// - bool: Icon / Switch
/// - DateTime: Text / DatePicker
/// - enum: Text / DropdownButtonFormField
/// - nested object: recursive DynamicFormWidget
/// - List: Column with add/remove controls
class DefaultFieldWidgets {
  /// Builds a widget based on FieldType enum (for DynamicFormWidget)
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
  }) {
    switch (type) {
      case FieldType.string:
        return _buildStringFieldSimple(
          label: label,
          value: value as String?,
          mode: mode,
          onChanged: onChanged,
          hint: hint,
          required: required,
          validationError: validationError,
          fieldKey: fieldKey,
        );
      case FieldType.int:
        return _buildIntFieldSimple(
          label: label,
          value: value as int?,
          mode: mode,
          onChanged: onChanged,
          hint: hint,
          required: required,
          validationError: validationError,
          fieldKey: fieldKey,
        );
      case FieldType.double:
        return _buildDoubleFieldSimple(
          label: label,
          value: value as double?,
          mode: mode,
          onChanged: onChanged,
          hint: hint,
          required: required,
          validationError: validationError,
          fieldKey: fieldKey,
        );
      case FieldType.bool:
        return _buildBoolFieldSimple(label: label, value: value as bool?, mode: mode, onChanged: onChanged);
      case FieldType.dateTime:
        return _buildDateTimeFieldSimple(
          label: label,
          value: DateTime.tryParse(value),
          mode: mode,
          onChanged: onChanged,
          hint: hint,
          validationError: validationError,
        );
      case FieldType.enumType:
        return _buildEnumFieldSimple(
          label: label,
          value: value,
          mode: mode,
          onChanged: onChanged,
          enumValues: enumValues,
          required: required,
          validationError: validationError,
          fieldKey: fieldKey,
        );
      case FieldType.list:
        return _buildListFieldSimple(
          label: label,
          value: value as List?,
          mode: mode,
          onChanged: onChanged,
          validationError: validationError,
        );
      case FieldType.nested:
        return _buildNestedFieldSimple(
          label: label,
          value: value as Map<String, dynamic>?,
          mode: mode,
          onChanged: onChanged,
          nestedFields: nestedFields,
        );
    }
  }

  // Helper: builds a labeled field row
  static Widget _buildFieldRow({required String label, required Widget child}) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(label, style: const TextStyle(fontWeight: FontWeight.bold, fontSize: 14)),
        const SizedBox(height: 4),
        child,
      ],
    );
  }

  // ============================================================================
  // Simplified field builders for DynamicFormWidget (using simple parameters)
  // ============================================================================

  static Widget _buildStringFieldSimple({
    required String label,
    required String? value,
    required AutoWidgetMode mode,
    required ValueChanged<String?> onChanged,
    String? hint,
    bool required = true,
    String? validationError,
    GlobalKey<FormFieldState>? fieldKey,
  }) {
    if (mode == AutoWidgetMode.view) {
      return _buildFieldRow(label: label, child: Text(value ?? '—'));
    }

    return _buildFieldRow(
      label: label,
      child: TextFormField(
        key: fieldKey,
        initialValue: value,
        decoration: InputDecoration(
          hintText: hint,
          border: const OutlineInputBorder(),
          contentPadding: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
        ),
        onChanged: onChanged,
        validator: (v) {
          if (required && (v == null || v.isEmpty)) {
            return validationError ?? '$label is required';
          }
          return null;
        },
      ),
    );
  }

  static Widget _buildIntFieldSimple({
    required String label,
    required int? value,
    required AutoWidgetMode mode,
    required ValueChanged<int?> onChanged,
    String? hint,
    bool required = true,
    String? validationError,
    GlobalKey<FormFieldState>? fieldKey,
  }) {
    if (mode == AutoWidgetMode.view) {
      return _buildFieldRow(label: label, child: Text(value?.toString() ?? '—'));
    }

    return _buildFieldRow(
      label: label,
      child: TextFormField(
        key: fieldKey,
        initialValue: value?.toString(),
        decoration: InputDecoration(
          hintText: hint,
          border: const OutlineInputBorder(),
          contentPadding: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
        ),
        keyboardType: TextInputType.number,
        inputFormatters: [FilteringTextInputFormatter.digitsOnly],
        onChanged: (text) => onChanged(text.isEmpty ? null : int.tryParse(text)),
        validator: (v) {
          if (required && (v == null || v.isEmpty)) {
            return validationError ?? '$label is required';
          }
          if (v != null && v.isNotEmpty && int.tryParse(v) == null) {
            return 'Must be a valid integer';
          }
          return null;
        },
      ),
    );
  }

  static Widget _buildDoubleFieldSimple({
    required String label,
    required double? value,
    required AutoWidgetMode mode,
    required ValueChanged<double?> onChanged,
    String? hint,
    bool required = true,
    String? validationError,
    GlobalKey<FormFieldState>? fieldKey,
  }) {
    if (mode == AutoWidgetMode.view) {
      return _buildFieldRow(label: label, child: Text(value?.toString() ?? '—'));
    }

    return _buildFieldRow(
      label: label,
      child: TextFormField(
        key: fieldKey,
        initialValue: value?.toString(),
        decoration: InputDecoration(
          hintText: hint,
          border: const OutlineInputBorder(),
          contentPadding: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
        ),
        keyboardType: const TextInputType.numberWithOptions(decimal: true),
        inputFormatters: [FilteringTextInputFormatter.allow(RegExp(r'^\d*\.?\d*'))],
        onChanged: (text) => onChanged(text.isEmpty ? null : double.tryParse(text)),
        validator: (v) {
          if (required && (v == null || v.isEmpty)) {
            return validationError ?? '$label is required';
          }
          if (v != null && v.isNotEmpty && double.tryParse(v) == null) {
            return 'Must be a valid number';
          }
          return null;
        },
      ),
    );
  }

  static Widget _buildBoolFieldSimple({
    required String label,
    required bool? value,
    required AutoWidgetMode mode,
    required ValueChanged<bool?> onChanged,
  }) {
    if (mode == AutoWidgetMode.view) {
      return _buildFieldRow(
        label: label,
        child: Icon(value == true ? Icons.check_circle : Icons.cancel, color: value == true ? Colors.green : Colors.grey),
      );
    }

    return _buildFieldRow(
      label: label,
      child: Switch(value: value ?? false, onChanged: onChanged),
    );
  }

  static Widget _buildDateTimeFieldSimple({
    required String label,
    required DateTime? value,
    required AutoWidgetMode mode,
    required ValueChanged<DateTime?> onChanged,
    String? hint,
    String? validationError,
  }) {
    final dateFormat = DateFormat('yyyy-MM-dd HH:mm');

    if (mode == AutoWidgetMode.view) {
      return _buildFieldRow(label: label, child: Text(value != null ? dateFormat.format(value) : '—'));
    }

    return _buildFieldRow(
      label: label,
      child: InkWell(
        onTap: () async {
          final context = _scaffoldMessengerKey.currentContext;
          if (context == null) return;

          final date = await showDatePicker(
            context: context,
            initialDate: value ?? DateTime.now(),
            firstDate: DateTime(1900),
            lastDate: DateTime(2100),
          );

          if (date != null) {
            final time = await showTimePicker(context: context, initialTime: TimeOfDay.fromDateTime(value ?? DateTime.now()));

            if (time != null) {
              onChanged(DateTime(date.year, date.month, date.day, time.hour, time.minute));
            }
          }
        },
        child: InputDecorator(
          decoration: InputDecoration(
            errorText: validationError,
            border: const OutlineInputBorder(),
            contentPadding: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
          ),
          child: Row(
            mainAxisAlignment: MainAxisAlignment.spaceBetween,
            children: [Text(value != null ? dateFormat.format(value) : hint ?? 'Select date'), const Icon(Icons.calendar_today)],
          ),
        ),
      ),
    );
  }

  static Widget _buildEnumFieldSimple({
    required String label,
    required dynamic value,
    required AutoWidgetMode mode,
    required ValueChanged<dynamic> onChanged,
    List<dynamic>? enumValues,
    bool required = true,
    String? validationError,
    GlobalKey<FormFieldState>? fieldKey,
  }) {
    if (mode == AutoWidgetMode.view) {
      return _buildFieldRow(label: label, child: Text(value?.toString().split('.').last ?? '—'));
    }

    return _buildFieldRow(
      label: label,
      child: DropdownButtonFormField<dynamic>(
        key: fieldKey,
        initialValue: value,
        decoration: InputDecoration(
          border: const OutlineInputBorder(),
          contentPadding: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
        ),
        items: enumValues?.map((ev) {
          return DropdownMenuItem<dynamic>(value: ev, child: Text(ev.toString().split('.').last));
        }).toList(),
        onChanged: onChanged,
        validator: (v) {
          if (required && v == null) {
            return validationError ?? '$label is required';
          }
          return null;
        },
      ),
    );
  }

  static Widget _buildListFieldSimple({
    required String label,
    required List? value,
    required AutoWidgetMode mode,
    required ValueChanged<List?> onChanged,
    String? validationError,
  }) {
    final items = value ?? [];

    if (items.isEmpty) {
      return _buildFieldRow(
        label: label,
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            const Text('—'),
            if (mode == AutoWidgetMode.edit)
              TextButton.icon(icon: const Icon(Icons.add), label: const Text('Add item'), onPressed: () => onChanged(['New item'])),
          ],
        ),
      );
    }

    return _buildFieldRow(
      label: label,
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          ...List.generate(items.length, (index) {
            return _buildListItem(
              index: index,
              item: items[index],
              mode: mode,
              onChanged: (newValue) {
                final newList = List.from(items);
                newList[index] = newValue;
                onChanged(newList);
              },
              onRemove: mode == AutoWidgetMode.edit
                  ? () {
                      final newList = List.from(items)..removeAt(index);
                      onChanged(newList);
                    }
                  : null,
            );
          }),
          if (mode == AutoWidgetMode.edit)
            TextButton.icon(
              icon: const Icon(Icons.add),
              label: const Text('Add item'),
              onPressed: () {
                // Use first item as template for complex objects
                final template = items.isNotEmpty && items.first is Map
                    ? Map<String, dynamic>.from((items.first as Map).map((k, v) => MapEntry(k, _defaultValueForType(v))))
                    : 'New item';
                onChanged(List.from(items)..add(template));
              },
            ),
          if (validationError != null) Text(validationError, style: const TextStyle(color: Colors.red, fontSize: 12)),
        ],
      ),
    );
  }

  /// Builds a single list item based on its type.
  static Widget _buildListItem({
    required int index,
    required dynamic item,
    required AutoWidgetMode mode,
    required ValueChanged<dynamic> onChanged,
    VoidCallback? onRemove,
  }) {
    final itemType = FieldConfig.inferType(item);

    // Complex object (Map) - render as expandable card
    if (itemType == FieldType.nested && item is Map<String, dynamic>) {
      return Card(
        margin: const EdgeInsets.only(bottom: 8.0),
        child: Padding(
          padding: const EdgeInsets.all(8.0),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Row(
                children: [
                  Text('[$index]', style: const TextStyle(fontWeight: FontWeight.bold)),
                  const Spacer(),
                  if (onRemove != null)
                    IconButton(
                      icon: const Icon(Icons.remove_circle, size: 20),
                      onPressed: onRemove,
                      padding: EdgeInsets.zero,
                      constraints: const BoxConstraints(),
                    ),
                ],
              ),
              const SizedBox(height: 8),
              ...item.keys.map((key) {
                return _buildNestedField(
                  key: key,
                  value: item[key],
                  config: null,
                  mode: mode,
                  onChanged: (newValue) {
                    final updated = Map<String, dynamic>.from(item);
                    updated[key] = newValue;
                    onChanged(updated);
                  },
                );
              }),
            ],
          ),
        ),
      );
    }

    // Nested list - recursive rendering
    if (itemType == FieldType.list && item is List) {
      return Card(
        margin: const EdgeInsets.only(bottom: 8.0),
        child: Padding(
          padding: const EdgeInsets.all(8.0),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Row(
                children: [
                  Text('[$index]', style: const TextStyle(fontWeight: FontWeight.bold)),
                  const Spacer(),
                  if (onRemove != null) IconButton(icon: const Icon(Icons.remove_circle, size: 20), onPressed: onRemove),
                ],
              ),
              _buildListFieldSimple(label: '', value: item, mode: mode, onChanged: onChanged),
            ],
          ),
        ),
      );
    }

    // Simple types - bullet point display
    if (mode == AutoWidgetMode.view) {
      return Padding(padding: const EdgeInsets.only(bottom: 4.0), child: Text('• ${item.toString()}'));
    }

    // Edit mode for simple types
    return Padding(
      padding: const EdgeInsets.only(bottom: 8.0),
      child: Row(
        children: [
          Expanded(
            child: TextFormField(
              initialValue: item?.toString() ?? '',
              decoration: const InputDecoration(
                border: OutlineInputBorder(),
                contentPadding: EdgeInsets.symmetric(horizontal: 12, vertical: 8),
                isDense: true,
              ),
              onChanged: (text) => onChanged(text),
            ),
          ),
          if (onRemove != null) IconButton(icon: const Icon(Icons.remove_circle), onPressed: onRemove),
        ],
      ),
    );
  }

  /// Returns a default value for a given type (for creating new list items).
  static dynamic _defaultValueForType(dynamic existingValue) {
    if (existingValue is String) return '';
    if (existingValue is int) return 0;
    if (existingValue is double) return 0.0;
    if (existingValue is bool) return false;
    if (existingValue is List) return <dynamic>[];
    if (existingValue is Map) {
      return Map<String, dynamic>.from(existingValue.map((k, v) => MapEntry(k, _defaultValueForType(v))));
    }
    return null;
  }

  static Widget _buildNestedFieldSimple({
    required String label,
    required Map<String, dynamic>? value,
    required AutoWidgetMode mode,
    required ValueChanged<Map<String, dynamic>?> onChanged,
    Map<String, FieldConfig>? nestedFields,
  }) {
    // Use value keys for auto-discovery (consistent with v2.2.0 pattern)
    final fieldsToRender = value?.keys ?? <String>[];

    if (fieldsToRender.isEmpty) {
      return _buildFieldRow(
        label: label,
        child: const Card(
          child: Padding(padding: EdgeInsets.all(8.0), child: Text('Empty object')),
        ),
      );
    }

    return _buildFieldRow(
      label: label,
      child: Card(
        child: Padding(
          padding: const EdgeInsets.all(8.0),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              for (final key in fieldsToRender)
                _buildNestedField(
                  key: key,
                  value: value![key],
                  config: nestedFields?[key],
                  mode: mode,
                  onChanged: (newValue) {
                    final updated = Map<String, dynamic>.from(value);
                    updated[key] = newValue;
                    onChanged(updated);
                  },
                ),
            ],
          ),
        ),
      ),
    );
  }

  /// Builds a single field within a nested object.
  /// Uses type inference and optional config overrides.
  static Widget _buildNestedField({
    required String key,
    required dynamic value,
    FieldConfig? config,
    required AutoWidgetMode mode,
    required ValueChanged<dynamic> onChanged,
  }) {
    // Infer type from value, with optional config override
    final effectiveType = config?.getEffectiveType(value) ?? FieldConfig.inferType(value);
    final label = config?.label ?? humanizeFieldName(toSnakeCase(key));

    if (effectiveType == null) {
      return Padding(
        padding: const EdgeInsets.only(bottom: 8.0),
        child: Text('Unknown type for "$key"', style: const TextStyle(color: Colors.red)),
      );
    }

    return Padding(
      padding: const EdgeInsets.only(bottom: 8.0),
      child: buildWidgetByType(
        type: effectiveType,
        label: label,
        value: value,
        mode: mode,
        onChanged: onChanged,
        required: config?.required ?? true,
        hint: config?.hint,
        validationError: config?.validationError,
        enumValues: config?.enumValues,
        nestedFields: config?.nestedFields,
      ),
    );
  }
}

// Global key for accessing BuildContext in date picker
final GlobalKey<ScaffoldMessengerState> _scaffoldMessengerKey = GlobalKey<ScaffoldMessengerState>();
