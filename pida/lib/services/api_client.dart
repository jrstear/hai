import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:pida/models/contact.dart';
import 'package:pida/models/lifelog.dart';
import 'package:pida/models/recording.dart';
import 'package:pida/models/speaker.dart';
import 'package:pida/providers/api_key_provider.dart';
import 'package:pida/providers/config_provider.dart';
import 'package:pida/utils/error_handler.dart';

/// API client service for communicating with the Hai API server
class ApiClient {
  final Dio _dio;
  final String baseUrl;
  final String? apiKey;
  final String? deviceId;

  ApiClient({
    required this.baseUrl,
    this.apiKey,
    this.deviceId,
    Dio? dio,
  }) : _dio = dio ?? Dio() {
    _dio.options.baseUrl = baseUrl;
    _dio.options.connectTimeout = const Duration(seconds: 30);
    _dio.options.receiveTimeout = const Duration(seconds: 30);
    _dio.options.headers['Content-Type'] = 'application/json';
    _dio.options.headers['Accept'] = 'application/json';
    
    // For Flutter web, CORS is handled by the API server
    // No need to set withCredentials unless server requires it

    // Set API key if provided (for Limitless API audio streaming)
    if (apiKey != null && apiKey!.isNotEmpty) {
      _dio.options.headers['X-API-Key'] = apiKey;
    }

    // Set device ID if provided (for future multi-device support)
    // NOTE: This relates to future schema changes where recordings will have
    // a recording_device_id field. Currently not used by API but prepared for future.
    if (deviceId != null && deviceId!.isNotEmpty) {
      _dio.options.headers['X-Device-ID'] = deviceId;
    }

    // Add interceptors for logging and error handling
    // NOTE: Log interceptor will NOT log API keys (they're in headers, not body)
    // Custom interceptor to handle upload endpoints with minimal logging
    _dio.interceptors.add(InterceptorsWrapper(
      onRequest: (options, handler) {
        // Mark upload requests so we can filter them in logging
        if (options.path.contains('/upload')) {
          options.extra['skipBodyLogging'] = true;
        }
        handler.next(options);
      },
      onResponse: (response, handler) {
        // For upload endpoints, log a simple success message
        if (response.requestOptions.path.contains('/upload')) {
          print('✓ API returned success in response to vCard upload request');
        }
        handler.next(response);
      },
    ));
    
    // Add standard log interceptor with custom logPrint to filter upload endpoints
    _dio.interceptors.add(LogInterceptor(
      requestBody: true,
      responseBody: true,
      error: true,
      // Don't log request headers to avoid exposing API keys
      requestHeader: false,
      // Custom log function to skip verbose logging for upload endpoints
      logPrint: (object) {
        final message = object.toString();
        // Skip request/response body logging for upload endpoints (contains entire vCard file)
        final isUpload = message.contains('/upload') || 
                        message.contains('/api/contacts/upload');
        final isBodyLog = message.contains('Response Body:') || 
                         message.contains('Request Body:');
        
        if (isUpload && isBodyLog) {
          return; // Skip verbose body logging for uploads
        }
        print(object);
      },
    ));
  }

  /// Update API key dynamically
  void setApiKey(String? key) {
    if (key != null && key.isNotEmpty) {
      _dio.options.headers['X-API-Key'] = key;
    } else {
      _dio.options.headers.remove('X-API-Key');
    }
  }

  /// Handle Dio errors and convert to ApiException
  ApiException _handleError(DioException e) {
    return handleDioError(e);
  }

  // Health endpoints
  Future<Map<String, dynamic>> health() async {
    try {
      final response = await _dio.get('/api/health');
      return response.data as Map<String, dynamic>;
    } on DioException catch (e) {
      throw _handleError(e);
    }
  }

  // Contacts endpoints
  Future<ContactListResponse> getContacts({
    bool? known,
    String? search,
  }) async {
    try {
      final queryParams = <String, dynamic>{};
      if (known != null) queryParams['known'] = known.toString();
      if (search != null && search.isNotEmpty) queryParams['search'] = search;

      final response = await _dio.get(
        '/api/contacts',
        queryParameters: queryParams,
      );
      return ContactListResponse.fromJson(response.data);
    } on DioException catch (e) {
      throw _handleError(e);
    }
  }

  Future<Contact> getContact(String id) async {
    try {
      final response = await _dio.get('/api/contacts/$id');
      return Contact.fromJson(response.data);
    } on DioException catch (e) {
      throw _handleError(e);
    }
  }

  Future<Contact> createContact(Contact contact) async {
    try {
      final response = await _dio.post(
        '/api/contacts',
        data: contact.toJson(),
      );
      return Contact.fromJson(response.data);
    } on DioException catch (e) {
      throw _handleError(e);
    }
  }

  Future<Contact> updateContact(String id, Contact contact) async {
    try {
      final response = await _dio.put(
        '/api/contacts/$id',
        data: contact.toJson(),
      );
      return Contact.fromJson(response.data);
    } on DioException catch (e) {
      throw _handleError(e);
    }
  }

  Future<void> associateSpeaker(String contactId, String speakerId) async {
    try {
      await _dio.post(
        '/api/contacts/$contactId/associate-speaker',
        data: AssociateSpeakerRequest(speakerId: speakerId).toJson(),
      );
    } on DioException catch (e) {
      throw _handleError(e);
    }
  }

  // Speakers endpoints
  Future<List<Speaker>> getUnassociatedSpeakers() async {
    try {
      final response = await _dio.get('/api/speakers/unassociated');
      final data = response.data as Map<String, dynamic>;
      final speakers = data['speakers'] as List<dynamic>;
      return speakers.map((json) => Speaker.fromJson(json as Map<String, dynamic>)).toList();
    } on DioException catch (e) {
      throw _handleError(e);
    }
  }

  Future<List<Segment>> getSpeakerRecordings(String speakerId) async {
    try {
      final response = await _dio.get('/api/speakers/$speakerId/recordings');
      final data = response.data as Map<String, dynamic>;
      final segments = data['segments'] as List<dynamic>;
      return segments.map((json) => Segment.fromJson(json as Map<String, dynamic>)).toList();
    } on DioException catch (e) {
      throw _handleError(e);
    }
  }

  // Recordings endpoints
  Future<RecordingAudioInfo> getRecordingAudio(
    String recordingId, {
    double? start,
    double? end,
  }) async {
    try {
      final queryParams = <String, dynamic>{};
      if (start != null) queryParams['start'] = start.toString();
      if (end != null) queryParams['end'] = end.toString();

      final response = await _dio.get(
        '/api/recordings/$recordingId/audio',
        queryParameters: queryParams,
      );
      return RecordingAudioInfo.fromJson(response.data);
    } on DioException catch (e) {
      throw _handleError(e);
    }
  }

  // Lifelogs endpoints
  Future<LifelogResponse> getLifelogs(String date) async {
    try {
      final response = await _dio.get(
        '/api/lifelogs',
        queryParameters: {'date': date},
      );
      return LifelogResponse.fromJson(response.data);
    } on DioException catch (e) {
      throw _handleError(e);
    }
  }

  // Blockquotes endpoints
  Future<void> updateBlockquoteContact(String blockquoteId, String? contactId) async {
    try {
      await _dio.put(
        '/api/blockquotes/$blockquoteId/contact',
        data: {'contact_id': contactId},
      );
    } on DioException catch (e) {
      throw _handleError(e);
    }
  }

  // Get conversation participants (optimized endpoint)
  Future<List<String>> getLifelogParticipants(String lifelogId) async {
    try {
      final response = await _dio.get('/api/lifelogs/$lifelogId/participants');
      final data = response.data as Map<String, dynamic>;
      final participants = data['participants'] as List<dynamic>;
      return participants.map((id) => id as String).toList();
    } on DioException catch (e) {
      throw _handleError(e);
    }
  }

  // Upload vCard file
  Future<Map<String, dynamic>> uploadVCard(String filePath) async {
    try {
      final formData = FormData.fromMap({
        'vcf': await MultipartFile.fromFile(filePath, filename: filePath.split('/').last),
      });
      
      // Remove Content-Type header to let Dio set it for multipart/form-data
      final options = Options(
        headers: {
          'Content-Type': 'multipart/form-data',
        },
      );
      
      final response = await _dio.post(
        '/api/contacts/upload',
        data: formData,
        options: options,
      );
      
      return response.data as Map<String, dynamic>;
    } on DioException catch (e) {
      throw _handleError(e);
    }
  }

  // Upload vCard file from bytes (for web drag and drop)
  Future<Map<String, dynamic>> uploadVCardFromBytes(List<int> bytes, String filename) async {
    try {
      final formData = FormData.fromMap({
        'vcf': MultipartFile.fromBytes(
          bytes,
          filename: filename,
        ),
      });
      
      final options = Options(
        headers: {
          'Content-Type': 'multipart/form-data',
        },
      );
      
      final response = await _dio.post(
        '/api/contacts/upload',
        data: formData,
        options: options,
      );
      
      return response.data as Map<String, dynamic>;
    } on DioException catch (e) {
      throw _handleError(e);
    }
  }

  /// Get a setting value by key
  Future<String> getSetting(String key) async {
    try {
      final response = await _dio.get('/api/settings/$key');
      return response.data['value'] as String;
    } on DioException catch (e) {
      throw _handleError(e);
    }
  }

  /// Set a setting value by key
  Future<void> setSetting(String key, String value) async {
    try {
      await _dio.put(
        '/api/settings/$key',
        data: {'value': value},
      );
    } on DioException catch (e) {
      throw _handleError(e);
    }
  }
}

/// Provider for API client
/// 
/// Automatically includes API key and device ID from environment variables.
/// In the future, these will come from user settings/secure storage.
final apiClientProvider = Provider<ApiClient>((ref) {
  final baseUrl = ref.watch(apiBaseUrlProvider);
  final apiKey = ref.watch(apiKeyProvider);
  final deviceId = ref.watch(recordingDeviceIdProvider);
  
  return ApiClient(
    baseUrl: baseUrl,
    apiKey: apiKey,
    deviceId: deviceId,
  );
});

