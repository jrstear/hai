import 'package:json_annotation/json_annotation.dart';

part 'speaker.g.dart';

@JsonSerializable()
class Speaker {
  final String id;
  @JsonKey(name: 'contact_id')
  final String? contactId;
  @JsonKey(name: 'first_seen')
  final DateTime firstSeen;
  @JsonKey(name: 'last_seen')
  final DateTime lastSeen;
  final int duration; // Total speaking time in seconds
  @JsonKey(name: 'duration_display')
  final String durationDisplay; // Formatted duration string
  @JsonKey(name: 'created_at')
  final DateTime createdAt;
  @JsonKey(name: 'updated_at')
  final DateTime updatedAt;

  Speaker({
    required this.id,
    this.contactId,
    required this.firstSeen,
    required this.lastSeen,
    required this.duration,
    required this.durationDisplay,
    required this.createdAt,
    required this.updatedAt,
  });

  factory Speaker.fromJson(Map<String, dynamic> json) => _$SpeakerFromJson(json);
  Map<String, dynamic> toJson() => _$SpeakerToJson(this);
}

