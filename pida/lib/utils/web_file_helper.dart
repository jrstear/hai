import 'dart:html' as html;
import 'dart:typed_data';
import 'dart:async';

/// Web-specific file helper for reading files
/// Only used on web platform
Future<Uint8List> readWebFileAsBytes(dynamic file) async {
  final reader = html.FileReader();
  final completer = Completer<Uint8List>();

  reader.onLoadEnd.listen((event) {
    completer.complete(reader.result as Uint8List);
  });

  reader.onError.listen((event) {
    completer.completeError(Exception('Failed to read file'));
  });

  reader.readAsArrayBuffer(file as html.File);
  return completer.future;
}

String getWebFileName(dynamic file) {
  return (file as html.File).name;
}

/// Prevent browser's default drag-and-drop behavior
/// This prevents files from being downloaded when dragged over the page
void preventDefaultDragBehavior() {
  final document = html.window.document;
  
  // Prevent default on dragover (allows drop)
  document.addEventListener('dragover', (e) {
    e.preventDefault();
    e.stopPropagation();
  });
  
  // Prevent default on drop (prevents navigation/download)
  document.addEventListener('drop', (e) {
    e.preventDefault();
    e.stopPropagation();
  });
}

/// Get the HTML document (for use in widgets)
dynamic getDocument() {
  return html.window.document;
}
