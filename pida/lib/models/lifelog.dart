import 'package:json_annotation/json_annotation.dart';

part 'lifelog.g.dart';

@JsonSerializable()
class Blockquote {
  final String id;
  @JsonKey(name: 'lifelog_id')
  final String lifelogId;
  @JsonKey(name: 'lifelog_title')
  final String? lifelogTitle;
  @JsonKey(name: 'speaker_name')
  final String speakerName;
  @JsonKey(name: 'speaker_id')
  final String? speakerId; // Optional: Global speaker ID (populated after matching)
  @JsonKey(name: 'contact_id')
  final String? contactId; // Optional: Associated contact ID (user-assigned)
  final String content;
  @JsonKey(name: 'start_time')
  final String startTime; // Formatted time (HH:MM:SS)
  @JsonKey(name: 'end_time')
  final String endTime; // Formatted time (HH:MM:SS)
  final double duration; // Duration in seconds
  @JsonKey(name: 'start_offset_ms')
  final int startOffsetMs; // Absolute Unix milliseconds
  @JsonKey(name: 'end_offset_ms')
  final int endOffsetMs; // Absolute Unix milliseconds

  Blockquote({
    required this.id,
    required this.lifelogId,
    this.lifelogTitle,
    required this.speakerName,
    this.speakerId,
    this.contactId,
    required this.content,
    required this.startTime,
    required this.endTime,
    required this.duration,
    required this.startOffsetMs,
    required this.endOffsetMs,
  });

  factory Blockquote.fromJson(Map<String, dynamic> json) => _$BlockquoteFromJson(json);
  Map<String, dynamic> toJson() => _$BlockquoteToJson(this);
}

@JsonSerializable()
class ConversationTiming {
  @JsonKey(name: 'start_ms')
  final int startMs;
  @JsonKey(name: 'end_ms')
  final int endMs;

  ConversationTiming({
    required this.startMs,
    required this.endMs,
  });

  factory ConversationTiming.fromJson(Map<String, dynamic> json) =>
      _$ConversationTimingFromJson(json);
  Map<String, dynamic> toJson() => _$ConversationTimingToJson(this);
}

@JsonSerializable()
class LifelogResponse {
  final String date;
  final List<Blockquote> blockquotes;
  final Map<String, List<Blockquote>> grouped;
  @JsonKey(name: 'conversationTimings', fromJson: _conversationTimingsFromJson)
  final Map<String, ConversationTiming> conversationTimings;
  final int total;

  LifelogResponse({
    required this.date,
    required this.blockquotes,
    required this.grouped,
    required this.conversationTimings,
    required this.total,
  });
  
  // Handle both snake_case and camelCase from API
  static Map<String, ConversationTiming> _conversationTimingsFromJson(dynamic json) {
    if (json == null) return {};
    final Map<String, dynamic> map = json as Map<String, dynamic>;
    return map.map((k, v) {
      if (v is Map<String, dynamic>) {
        return MapEntry(k, ConversationTiming.fromJson(v));
      }
      return MapEntry(k, ConversationTiming.fromJson(v as Map<String, dynamic>));
    });
  }

  factory LifelogResponse.fromJson(Map<String, dynamic> json) =>
      _$LifelogResponseFromJson(json);
  Map<String, dynamic> toJson() => _$LifelogResponseToJson(this);
}

