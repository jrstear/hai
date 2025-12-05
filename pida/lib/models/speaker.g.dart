// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'speaker.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

Speaker _$SpeakerFromJson(Map<String, dynamic> json) => Speaker(
      id: json['id'] as String,
      contactId: json['contact_id'] as String?,
      firstSeen: DateTime.parse(json['first_seen'] as String),
      lastSeen: DateTime.parse(json['last_seen'] as String),
      duration: (json['duration'] as num).toInt(),
      durationDisplay: json['duration_display'] as String,
      createdAt: DateTime.parse(json['created_at'] as String),
      updatedAt: DateTime.parse(json['updated_at'] as String),
    );

Map<String, dynamic> _$SpeakerToJson(Speaker instance) => <String, dynamic>{
      'id': instance.id,
      'contact_id': instance.contactId,
      'first_seen': instance.firstSeen.toIso8601String(),
      'last_seen': instance.lastSeen.toIso8601String(),
      'duration': instance.duration,
      'duration_display': instance.durationDisplay,
      'created_at': instance.createdAt.toIso8601String(),
      'updated_at': instance.updatedAt.toIso8601String(),
    };
