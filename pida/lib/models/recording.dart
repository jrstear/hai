import 'package:json_annotation/json_annotation.dart';

part 'recording.g.dart';

@JsonSerializable()
class Segment {
  final int id;
  @JsonKey(name: 'speaker_id')
  final String? speakerId;
  @JsonKey(name: 'recording_id')
  final String recordingId;
  @JsonKey(name: 'start_time')
  final double startTime;
  @JsonKey(name: 'end_time')
  final double endTime;
  final double duration;
  @JsonKey(name: 'speaker_name')
  final String? speakerName;
  final String? transcript;
  final String time; // Formatted time
  @JsonKey(name: 'blockquote_time')
  final String? blockquoteTime;
  @JsonKey(name: 'blockquote_duration')
  final double? blockquoteDuration;

  Segment({
    required this.id,
    this.speakerId,
    required this.recordingId,
    required this.startTime,
    required this.endTime,
    required this.duration,
    this.speakerName,
    this.transcript,
    required this.time,
    this.blockquoteTime,
    this.blockquoteDuration,
  });

  factory Segment.fromJson(Map<String, dynamic> json) => _$SegmentFromJson(json);
  Map<String, dynamic> toJson() => _$SegmentToJson(this);
}

@JsonSerializable()
class RecordingAudioInfo {
  @JsonKey(name: 'api_url')
  final String apiUrl;
  @JsonKey(name: 'query_params')
  final Map<String, dynamic> queryParams;
  final Map<String, String> headers;
  @JsonKey(name: 'absolute_start_time')
  final String absoluteStartTime;
  @JsonKey(name: 'absolute_end_time')
  final String absoluteEndTime;
  @JsonKey(name: 'content_type')
  final String contentType;

  RecordingAudioInfo({
    required this.apiUrl,
    required this.queryParams,
    required this.headers,
    required this.absoluteStartTime,
    required this.absoluteEndTime,
    required this.contentType,
  });

  factory RecordingAudioInfo.fromJson(Map<String, dynamic> json) =>
      _$RecordingAudioInfoFromJson(json);
  Map<String, dynamic> toJson() => _$RecordingAudioInfoToJson(this);
}

