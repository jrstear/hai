import 'dart:typed_data';
import 'dart:async';

/// Stub implementation for non-web platforms
Future<Uint8List> readWebFileAsBytes(dynamic file) async {
  throw UnsupportedError('readWebFileAsBytes only available on web');
}

String getWebFileName(dynamic file) {
  throw UnsupportedError('getWebFileName only available on web');
}
