import 'dart:typed_data';
import 'package:flutter/material.dart';
import 'storage.dart';

/// A widget that accepts drag-and-drop file uploads using Flutter's built-in DragTarget.
///
/// Files can be uploaded programmatically via the [StorageDropZoneState.uploadFile] method,
/// which is accessible through a [GlobalKey].
///
/// Example usage:
/// ```dart
/// final dropZoneKey = GlobalKey<StorageDropZoneState>();
///
/// StorageDropZone(
///   key: dropZoneKey,
///   storage: storage,
///   pathBuilder: (fileName) => 'uploads/$fileName',
///   onUploadComplete: (path) => print('Uploaded to: $path'),
///   onError: (error) => print('Upload failed: $error'),
///   child: Container(
///     width: 200,
///     height: 200,
///     decoration: BoxDecoration(
///       border: Border.all(color: Colors.grey),
///       borderRadius: BorderRadius.circular(8),
///     ),
///     child: const Center(child: Text('Drop files here')),
///   ),
/// )
///
/// // To upload programmatically (e.g., from file picker):
/// dropZoneKey.currentState?.uploadFile('photo.jpg', imageBytes);
/// ```
class StorageDropZone extends StatefulWidget {
  /// The storage instance to use for uploads.
  final Storage storage;

  /// Builds the storage path from the file name.
  ///
  /// Example: `(fileName) => 'uploads/${DateTime.now().millisecondsSinceEpoch}/$fileName'`
  final String Function(String fileName) pathBuilder;

  /// Called when a file is successfully uploaded.
  final void Function(String path)? onUploadComplete;

  /// Called when a file upload fails.
  final void Function(Object error)? onError;

  /// Called when upload starts.
  final void Function()? onUploadStart;

  /// The child widget (drop zone content).
  final Widget child;

  /// Optional widget to display when dragging over the zone.
  final Widget? draggingChild;

  const StorageDropZone({
    super.key,
    required this.storage,
    required this.pathBuilder,
    required this.child,
    this.onUploadComplete,
    this.onError,
    this.onUploadStart,
    this.draggingChild,
  });

  @override
  State<StorageDropZone> createState() => StorageDropZoneState();
}

class StorageDropZoneState extends State<StorageDropZone> {
  bool _isDragging = false;
  bool _isUploading = false;

  /// Whether an upload is currently in progress.
  bool get isUploading => _isUploading;

  /// Upload a file programmatically.
  ///
  /// This is useful for integrating with file pickers or other file sources.
  /// The [fileName] is used with [pathBuilder] to generate the storage path.
  ///
  /// Returns the storage path on success.
  /// Throws on failure (error is also passed to [onError] callback).
  Future<String> uploadFile(String fileName, Uint8List data) async {
    setState(() => _isUploading = true);
    widget.onUploadStart?.call();

    try {
      final path = widget.pathBuilder(fileName);
      await widget.storage.save(path, data);
      widget.onUploadComplete?.call(path);
      return path;
    } catch (e) {
      widget.onError?.call(e);
      rethrow;
    } finally {
      if (mounted) {
        setState(() => _isUploading = false);
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    return DragTarget<Object>(
      onWillAcceptWithDetails: (details) {
        setState(() => _isDragging = true);
        return true;
      },
      onLeave: (_) {
        setState(() => _isDragging = false);
      },
      onAcceptWithDetails: (details) {
        setState(() => _isDragging = false);
        // Note: Flutter's DragTarget receives data from Draggable widgets.
        // For platform file drops (desktop/web), integrate with platform-specific
        // APIs (e.g., file_picker package) and call uploadFile() directly.
      },
      builder: (context, candidateData, rejectedData) {
        // Show uploading state
        if (_isUploading) {
          return Stack(
            children: [
              widget.child,
              Positioned.fill(
                child: ColoredBox(
                  color: Colors.black26,
                  child: Center(
                    child: Column(
                      mainAxisSize: MainAxisSize.min,
                      children: [
                        const CircularProgressIndicator(),
                        const SizedBox(height: 8),
                        Text(
                          'Uploading...',
                          style: Theme.of(context).textTheme.bodyMedium?.copyWith(
                                color: Colors.white,
                              ),
                        ),
                      ],
                    ),
                  ),
                ),
              ),
            ],
          );
        }

        // Show dragging state
        if (_isDragging && widget.draggingChild != null) {
          return widget.draggingChild!;
        }

        // Normal state with visual feedback when dragging
        return AnimatedContainer(
          duration: const Duration(milliseconds: 150),
          decoration: _isDragging
              ? BoxDecoration(
                  border: Border.all(
                    color: Theme.of(context).primaryColor,
                    width: 2,
                  ),
                  borderRadius: BorderRadius.circular(8),
                )
              : null,
          child: widget.child,
        );
      },
    );
  }
}
