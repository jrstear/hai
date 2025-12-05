import 'package:dio/dio.dart';
import 'package:pida/models/api_error.dart';

/// Custom API exception
class ApiException implements Exception {
  final String message;
  final int? statusCode;
  final dynamic originalError;

  ApiException({
    required this.message,
    this.statusCode,
    this.originalError,
  });

  @override
  String toString() => message;
}

/// Handle Dio errors and convert to ApiException
ApiException handleDioError(DioException error) {
  switch (error.type) {
    case DioExceptionType.connectionTimeout:
    case DioExceptionType.sendTimeout:
    case DioExceptionType.receiveTimeout:
      return ApiException(
        message: 'Connection timeout. Please check your internet connection.',
        statusCode: null,
        originalError: error,
      );

    case DioExceptionType.badResponse:
      final statusCode = error.response?.statusCode;
      final data = error.response?.data;

      // Try to parse API error response
      if (data is Map<String, dynamic>) {
        try {
          final apiError = ApiError.fromJson(data);
          return ApiException(
            message: apiError.message ?? apiError.error,
            statusCode: apiError.status ?? statusCode,
            originalError: error,
          );
        } catch (_) {
          // If parsing fails, use raw error
        }
      }

      return ApiException(
        message: data?['message']?.toString() ??
            data?['error']?.toString() ??
            'Server error (${statusCode ?? 'unknown'})',
        statusCode: statusCode,
        originalError: error,
      );

    case DioExceptionType.cancel:
      return ApiException(
        message: 'Request cancelled',
        statusCode: null,
        originalError: error,
      );

    case DioExceptionType.connectionError:
      return ApiException(
        message: 'Connection error. Please ensure the API server is running at http://localhost:8080 and check your browser console for details.',
        statusCode: null,
        originalError: error,
      );

    case DioExceptionType.badCertificate:
      return ApiException(
        message: 'Certificate error',
        statusCode: null,
        originalError: error,
      );

    case DioExceptionType.unknown:
    default:
      return ApiException(
        message: error.message ?? 'An unexpected error occurred',
        statusCode: null,
        originalError: error,
      );
  }
}

