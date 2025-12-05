# Flutter Web CORS Connection Error Fix

## Problem

Flutter web app shows connection errors when trying to connect to API server:

```
*** DioException ***:
DioException [connection error]: The connection errored: The XMLHttpRequest onError callback was called.

extra: {withCredentials: true}
```

## Root Cause

When `withCredentials: true` is set in Dio's request configuration for Flutter web:
- The browser sends credentials with the request
- CORS requires **exact origin matching** (cannot use wildcard `*`)
- Our API server uses wildcard CORS (`AllowedOrigins: []string{"*"}`)
- Browser blocks the request because credentials + wildcard CORS are incompatible

## Solution

### 1. Remove `withCredentials` from Dio Configuration

In `pida/lib/services/api_client.dart`, **do NOT** set `withCredentials: true`:

```dart
ApiClient({
  required this.baseUrl,
  this.apiKey,
  this.deviceId,
  Dio? dio,
}) : _dio = dio ?? Dio() {
  _dio.options.baseUrl = baseUrl;
  // ... other configuration ...
  
  // ❌ WRONG - Do NOT do this:
  // if (kIsWeb) {
  //   _dio.options.extra['withCredentials'] = true;
  // }
  
  // ✅ CORRECT - CORS is handled by API server, no credentials needed:
  // For Flutter web, CORS is handled by the API server
  // No need to set withCredentials unless server requires it
}
```

### 2. Full App Restart Required

**Important**: After changing Dio configuration, you **must do a full restart** of the Flutter app, not just a hot reload:

- ❌ Hot reload (`r` key) - Won't work for configuration changes
- ✅ Full restart - Stop and restart the app completely

Dio configuration is initialized at app startup, so hot reload doesn't pick up these changes.

## API Server CORS Configuration

Our API server (Go/chi) is correctly configured for development:

```go
r.Use(cors.Handler(cors.Options{
    AllowedOrigins:   []string{"*"}, // Wildcard works when no credentials
    AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
    AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-API-Key", "X-Device-ID"},
    AllowCredentials: false, // Must be false when using wildcard origins
    MaxAge:           300,
}))
```

## Verification

After the fix, the error log should **not** show `withCredentials: true`:

```
extra: {}  // ✅ Empty or missing withCredentials
```

## Related Files

- `pida/lib/services/api_client.dart` - Dio client configuration
- `api/cmd/server/main.go` - CORS middleware configuration

## When This Error Appears

This error typically appears when:
1. Someone adds `withCredentials: true` to Dio configuration
2. The API server is using wildcard CORS origins (`*`)
3. Configuration changes were made but only hot reload was done (not full restart)

## Prevention

- Always do a **full restart** after changing Dio configuration
- Never set `withCredentials: true` unless the API server explicitly requires it
- If credentials are needed, API server must use specific origins (not `*`)

