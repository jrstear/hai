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
