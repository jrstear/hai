// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'lifelog.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

Blockquote _$BlockquoteFromJson(Map<String, dynamic> json) => Blockquote(
      id: json['id'] as String,
      lifelogId: json['lifelog_id'] as String,
      lifelogTitle: json['lifelog_title'] as String?,
      speakerName: json['speaker_name'] as String,
      speakerId: json['speaker_id'] as String?,
      content: json['content'] as String,
      startTime: json['start_time'] as String,
      endTime: json['end_time'] as String,
      duration: (json['duration'] as num).toDouble(),
      startOffsetMs: (json['start_offset_ms'] as num).toInt(),
      endOffsetMs: (json['end_offset_ms'] as num).toInt(),
    );

Map<String, dynamic> _$BlockquoteToJson(Blockquote instance) =>
    <String, dynamic>{
      'id': instance.id,
      'lifelog_id': instance.lifelogId,
      'lifelog_title': instance.lifelogTitle,
      'speaker_name': instance.speakerName,
      'speaker_id': instance.speakerId,
      'content': instance.content,
      'start_time': instance.startTime,
      'end_time': instance.endTime,
      'duration': instance.duration,
      'start_offset_ms': instance.startOffsetMs,
      'end_offset_ms': instance.endOffsetMs,
    };

ConversationTiming _$ConversationTimingFromJson(Map<String, dynamic> json) =>
    ConversationTiming(
      startMs: (json['start_ms'] as num).toInt(),
      endMs: (json['end_ms'] as num).toInt(),
    );

Map<String, dynamic> _$ConversationTimingToJson(ConversationTiming instance) =>
    <String, dynamic>{
      'start_ms': instance.startMs,
      'end_ms': instance.endMs,
    };

LifelogResponse _$LifelogResponseFromJson(Map<String, dynamic> json) =>
    LifelogResponse(
      date: json['date'] as String,
      blockquotes: (json['blockquotes'] as List<dynamic>)
          .map((e) => Blockquote.fromJson(e as Map<String, dynamic>))
          .toList(),
      grouped: (json['grouped'] as Map<String, dynamic>).map(
        (k, e) => MapEntry(
            k,
            (e as List<dynamic>)
                .map((e) => Blockquote.fromJson(e as Map<String, dynamic>))
                .toList()),
      ),
      conversationTimings: LifelogResponse._conversationTimingsFromJson(
          json['conversationTimings']),
      total: (json['total'] as num).toInt(),
    );

Map<String, dynamic> _$LifelogResponseToJson(LifelogResponse instance) =>
    <String, dynamic>{
      'date': instance.date,
      'blockquotes': instance.blockquotes,
      'grouped': instance.grouped,
      'conversationTimings': instance.conversationTimings,
      'total': instance.total,
    };
