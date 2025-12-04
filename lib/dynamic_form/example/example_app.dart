import 'package:flutter/material.dart';
import '../main.dart';
import '../field_widget.dart';
import '../schema.dart';
import '../theme.dart';
import 'models.dart';

/// Example Flutter app demonstrating DynamicFormWidget usage
///
/// This app shows how to:
/// 1. Display Map data in view mode
/// 2. Switch to edit mode
/// 3. Edit field values
/// 4. Save and retrieve the updated Map
/// 5. Use type inference and custom field configurations
/// 6. Use DynamicFormTheme for custom default widgets
/// 7. Use auto-labeling with labelResolver
///
/// Run this with: flutter run lib/util/builder/example/example_app.dart
void main() {
  runApp(const DynamicFormWidgetExampleApp());
}

class DynamicFormWidgetExampleApp extends StatelessWidget {
  const DynamicFormWidgetExampleApp({Key? key}) : super(key: key);

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: 'DynamicFormWidget Example',
      theme: ThemeData(primarySwatch: Colors.blue, useMaterial3: true),
      // Wrap app with DynamicFormTheme for project-wide customizations
      home: DynamicFormTheme(
        // Custom widget builders by type (Feature 1)
        builders: {
          FieldType.bool: _buildCustomBoolWidget,
        },
        // Project-wide label resolver (Feature 2)
        labelResolver: _projectLabelResolver,
        child: const ExampleHomePage(),
      ),
    );
  }

  /// Custom bool widget with a more descriptive display
  static Widget _buildCustomBoolWidget({
    required String label,
    required dynamic value,
    required AutoWidgetMode mode,
    required ValueChanged<dynamic> onChanged,
    bool required = true,
    String? hint,
    String? validationError,
    List<dynamic>? enumValues,
    dynamic nestedFields,
    GlobalKey<FormFieldState>? fieldKey,
  }) {
    final boolValue = value as bool?;

    if (mode == AutoWidgetMode.view) {
      return Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(label, style: const TextStyle(fontWeight: FontWeight.bold, fontSize: 14)),
          const SizedBox(height: 4),
          Container(
            padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 6),
            decoration: BoxDecoration(
              color: boolValue == true ? Colors.green.shade100 : Colors.grey.shade200,
              borderRadius: BorderRadius.circular(16),
            ),
            child: Text(
              boolValue == true ? 'Active' : 'Inactive',
              style: TextStyle(
                color: boolValue == true ? Colors.green.shade800 : Colors.grey.shade600,
                fontWeight: FontWeight.w500,
              ),
            ),
          ),
        ],
      );
    }

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(label, style: const TextStyle(fontWeight: FontWeight.bold, fontSize: 14)),
        const SizedBox(height: 4),
        SwitchListTile(
          value: boolValue ?? false,
          onChanged: onChanged,
          title: Text(boolValue == true ? 'Active' : 'Inactive'),
          contentPadding: EdgeInsets.zero,
        ),
      ],
    );
  }

  /// Simulated ARB-style label resolver
  /// In a real app, this would use AppLocalizations.of(context)
  static String? _projectLabelResolver(String fieldNameSnakeCase) {
    // Simulate ARB resource lookup
    const labels = {
      'id': 'User ID',
      'name': 'Full Name',
      'email': 'Email Address',
      'age': 'Age (years)',
      'is_active': 'Active Status',
      'account_type': 'Account Type',
      // Auto-labeling example fields
      'first_name': 'First Name',
      'last_name': 'Last Name',
      'phone_number': 'Phone Number',
    };
    return labels[fieldNameSnakeCase];
  }
}

class ExampleHomePage extends StatefulWidget {
  const ExampleHomePage({Key? key}) : super(key: key);

  @override
  State<ExampleHomePage> createState() => _ExampleHomePageState();
}

class _ExampleHomePageState extends State<ExampleHomePage> {
  // Initial user profile data (using Map instead of typed object)
  Map<String, dynamic> _userProfile = {
    'id': '123',
    'name': 'John Doe',
    'email': 'john.doe@example.com',
    'age': 30,
    'isActive': true,
    'accountType': AccountType.premium,
  };

  // Field configurations - demonstrating auto-labeling
  // Labels are resolved via: FieldConfig.label > widget.labelResolver > theme.labelResolver > humanize
  final Map<String, FieldConfig> _fieldConfigs = {
    'id': const FieldConfig(
      // No label - will use theme.labelResolver ('id' -> 'User ID')
      type: FieldType.string,
      required: false,
    ),
    'name': const FieldConfig(
      // No label - will use theme.labelResolver ('name' -> 'Full Name')
      hint: 'Enter your full name',
      // type is inferred from value (String)
    ),
    'email': const FieldConfig(
      // No label - will use theme.labelResolver ('email' -> 'Email Address')
      hint: 'Enter your email',
      // type is inferred from value (String)
    ),
    'age': const FieldConfig(
      // No label - will use theme.labelResolver ('age' -> 'Age (years)')
      hint: 'Enter your age',
      type: FieldType.int, // Explicitly specified
    ),
    'isActive': const FieldConfig(
      // No label - will use theme.labelResolver ('is_active' -> 'Active Status')
      // Uses custom bool widget from DynamicFormTheme.builders
    ),
    'accountType': FieldConfig(
      // No label - will use theme.labelResolver ('account_type' -> 'Account Type')
      type: FieldType.enumType,
      enumValues: AccountType.values,
    ),
  };

  // Controller for DynamicFormWidget
  final GlobalKey<DynamicFormWidgetState> _formKey = GlobalKey<DynamicFormWidgetState>();

  // Current mode
  AutoWidgetMode _mode = AutoWidgetMode.view;

  // Status message
  String? _statusMessage;

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('DynamicFormWidget Example'),
        actions: [
          if (_mode == AutoWidgetMode.view)
            IconButton(
              icon: const Icon(Icons.edit),
              onPressed: () {
                setState(() {
                  _mode = AutoWidgetMode.edit;
                  _statusMessage = null;
                });
              },
              tooltip: 'Edit',
            )
          else ...  [
            IconButton(
              icon: const Icon(Icons.cancel),
              onPressed: () {
                setState(() {
                  _mode = AutoWidgetMode.view;
                  _statusMessage = null;
                  _formKey.currentState?.reset();
                });
              },
              tooltip: 'Cancel',
            ),
            IconButton(icon: const Icon(Icons.save), onPressed: _saveProfile, tooltip: 'Save'),
          ],
        ],
      ),
      body: SingleChildScrollView(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            // Status message
            if (_statusMessage != null)
              Container(
                padding: const EdgeInsets.all(12),
                margin: const EdgeInsets.only(bottom: 16),
                decoration: BoxDecoration(color: Colors.green.shade100, borderRadius: BorderRadius.circular(8)),
                child: Row(
                  children: [
                    const Icon(Icons.check_circle, color: Colors.green),
                    const SizedBox(width: 8),
                    Expanded(
                      child: Text(_statusMessage!, style: const TextStyle(color: Colors.green)),
                    ),
                  ],
                ),
              ),

            // Mode indicator
            Container(
              padding: const EdgeInsets.all(12),
              margin: const EdgeInsets.only(bottom: 16),
              decoration: BoxDecoration(
                color: _mode == AutoWidgetMode.view ? Colors.blue.shade50 : Colors.orange.shade50,
                borderRadius: BorderRadius.circular(8),
                border: Border.all(color: _mode == AutoWidgetMode.view ? Colors.blue.shade200 : Colors.orange.shade200),
              ),
              child: Row(
                children: [
                  Icon(
                    _mode == AutoWidgetMode.view ? Icons.visibility : Icons.edit,
                    color: _mode == AutoWidgetMode.view ? Colors.blue : Colors.orange,
                  ),
                  const SizedBox(width: 8),
                  Text(
                    _mode == AutoWidgetMode.view ? 'View Mode' : 'Edit Mode',
                    style: TextStyle(fontWeight: FontWeight.bold, color: _mode == AutoWidgetMode.view ? Colors.blue : Colors.orange),
                  ),
                ],
              ),
            ),

            // DynamicFormWidget
            Card(
              elevation: 2,
              child: Padding(
                padding: const EdgeInsets.all(16),
                child: DynamicFormWidget(
                  key: _formKey,
                  value: _userProfile,
                  fieldConfigs: _fieldConfigs,
                  mode: _mode,
                ),
              ),
            ),

            const SizedBox(height: 24),

            // Info card
            Card(
              elevation: 1,
              color: Colors.grey.shade100,
              child: Padding(
                padding: const EdgeInsets.all(16),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: const [
                    Row(
                      children: [
                        Icon(Icons.info_outline, size: 20),
                        SizedBox(width: 8),
                        Text('How it works', style: TextStyle(fontWeight: FontWeight.bold)),
                      ],
                    ),
                    SizedBox(height: 8),
                    Text(
                      '1. Click Edit to switch to edit mode\n'
                      '2. Modify the field values\n'
                      '3. Click Save to apply changes\n'
                      '4. Click Cancel to discard changes\n\n'
                      'Features:\n'
                      '• Type inference (name, email, isActive)\n'
                      '• Explicit types (age: int, accountType: enum)\n'
                      '• Custom hints\n'
                      '• Map-based data (no code generation needed)\n\n'
                      'New Features:\n'
                      '• Auto-labeling via labelResolver\n'
                      '• Custom default widgets via DynamicFormTheme\n'
                      '• isActive uses custom bool widget from theme',
                      style: TextStyle(fontSize: 12),
                    ),
                  ],
                ),
              ),
            ),

            const SizedBox(height: 16),

            // Debug info
            ExpansionTile(
              title: const Text('Debug Info'),
              children: [
                Padding(
                  padding: const EdgeInsets.all(16),
                  child: Container(
                    padding: const EdgeInsets.all(12),
                    decoration: BoxDecoration(color: Colors.grey.shade200, borderRadius: BorderRadius.circular(4)),
                    child: Text(
                      'Current UserProfile Map:\n'
                      'ID: ${_userProfile['id']}\n'
                      'Name: ${_userProfile['name']}\n'
                      'Email: ${_userProfile['email']}\n'
                      'Age: ${_userProfile['age']}\n'
                      'Active: ${_userProfile['isActive']}\n'
                      'Account Type: ${(_userProfile['accountType'] as AccountType).name}',
                      style: const TextStyle(fontFamily: 'monospace', fontSize: 12),
                    ),
                  ),
                ),
              ],
            ),
          ],
        ),
      ),
    );
  }

  Future<void> _saveProfile() async {
    try {
      // Validate and save
      final updatedProfile = await _formKey.currentState!.save();

      setState(() {
        _userProfile = updatedProfile;
        _mode = AutoWidgetMode.view;
        _statusMessage = 'Profile saved successfully!';
      });

      // Clear status message after 3 seconds
      Future.delayed(const Duration(seconds: 3), () {
        if (mounted) {
          setState(() {
            _statusMessage = null;
          });
        }
      });

      // In a real app, you would send this to your backend:
      // await apiClient.updateUser(updatedProfile);
      print('Saved profile: $updatedProfile');
    } catch (e) {
      // Handle validation errors
      ScaffoldMessenger.of(
        context,
      ).showSnackBar(SnackBar(content: Text('Validation failed: ${e.toString()}'), backgroundColor: Colors.red));
    }
  }
}
