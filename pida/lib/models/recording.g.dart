// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'recording.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

Segment _$SegmentFromJson(Map<String, dynamic> json) => Segment(
      id: (json['id'] as num).toInt(),
      speakerId: json['speaker_id'] as String?,
      recordingId: json['recording_id'] as String,
      startTime: (json['start_time'] as num).toDouble(),
      endTime: (json['end_time'] as num).toDouble(),
      duration: (json['duration'] as num).toDouble(),
      speakerName: json['speaker_name'] as String?,
      transcript: json['transcript'] as String?,
      time: json['time'] as String,
      blockquoteTime: json['blockquote_time'] as String?,
      blockquoteDuration: (json['blockquote_duration'] as num?)?.toDouble(),
    );

Map<String, dynamic> _$SegmentToJson(Segment instance) => <String, dynamic>{
      'id': instance.id,
      'speaker_id': instance.speakerId,
      'recording_id': instance.recordingId,
      'start_time': instance.startTime,
      'end_time': instance.endTime,
      'duration': instance.duration,
      'speaker_name': instance.speakerName,
      'transcript': instance.transcript,
      'time': instance.time,
      'blockquote_time': instance.blockquoteTime,
      'blockquote_duration': instance.blockquoteDuration,
    };

RecordingAudioInfo _$RecordingAudioInfoFromJson(Map<String, dynamic> json) =>
    RecordingAudioInfo(
      apiUrl: json['api_url'] as String,
      queryParams: json['query_params'] as Map<String, dynamic>,
      headers: Map<String, String>.from(json['headers'] as Map),
      absoluteStartTime: json['absolute_start_time'] as String,
      absoluteEndTime: json['absolute_end_time'] as String,
      contentType: json['content_type'] as String,
    );

Map<String, dynamic> _$RecordingAudioInfoToJson(RecordingAudioInfo instance) =>
    <String, dynamic>{
      'api_url': instance.apiUrl,
      'query_params': instance.queryParams,
      'headers': instance.headers,
      'absolute_start_time': instance.absoluteStartTime,
      'absolute_end_time': instance.absoluteEndTime,
      'content_type': instance.contentType,
    };
