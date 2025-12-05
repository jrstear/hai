import 'package:json_annotation/json_annotation.dart';

part 'contact.g.dart';

@JsonSerializable()
class Contact {
  final String id;
  @JsonKey(name: 'external_id')
  final String? externalId;
  final String name;
  @JsonKey(name: 'given_name')
  final String? givenName;
  @JsonKey(name: 'family_name')
  final String? familyName;
  final String? email;
  final String? phone;
  @JsonKey(name: 'picture_url')
  final String? pictureUrl;
  @JsonKey(name: 'favorite_color')
  final String? favoriteColor;
  @JsonKey(defaultValue: false)
  final bool known;
  @JsonKey(name: 'created_at', fromJson: _dateTimeFromJson, toJson: _dateTimeToJson)
  final DateTime? createdAt;
  @JsonKey(name: 'updated_at', fromJson: _dateTimeFromJson, toJson: _dateTimeToJson)
  final DateTime? updatedAt;
  final String? source;

  Contact({
    required this.id,
    this.externalId,
    required this.name,
    this.givenName,
    this.familyName,
    this.email,
    this.phone,
    this.pictureUrl,
    this.favoriteColor,
    this.known = false,
    this.createdAt,
    this.updatedAt,
    this.source,
  });

  factory Contact.fromJson(Map<String, dynamic> json) {
    // Custom fromJson to handle API response structure properly
    return Contact(
      id: json['id'] as String,
      externalId: json['external_id'] as String?,
      name: json['name'] as String? ?? '',
      givenName: json['given_name'] as String?,
      familyName: json['family_name'] as String?,
      email: json['email'] as String?,
      phone: json['phone'] as String?,
      pictureUrl: json['picture_url'] as String?,
      favoriteColor: json['favorite_color'] as String?,
      known: json['known'] as bool? ?? false,
      createdAt: json['created_at'] != null
          ? _dateTimeFromJson(json['created_at'] as String?)
          : null,
      updatedAt: json['updated_at'] != null
          ? _dateTimeFromJson(json['updated_at'] as String?)
          : null,
      source: json['source'] as String?,
    );
  }

  Map<String, dynamic> toJson() => _$ContactToJson(this);

  static DateTime? _dateTimeFromJson(String? json) {
    if (json == null || json.isEmpty) return null;
    try {
      return DateTime.parse(json);
    } catch (e) {
      return null;
    }
  }

  static String? _dateTimeToJson(DateTime? dateTime) {
    return dateTime?.toIso8601String();
  }
}

@JsonSerializable()
class ContactListResponse {
  final List<Contact> contacts;
  final int total;

  ContactListResponse({
    required this.contacts,
    required this.total,
  });

  factory ContactListResponse.fromJson(Map<String, dynamic> json) =>
      _$ContactListResponseFromJson(json);
  Map<String, dynamic> toJson() => _$ContactListResponseToJson(this);
}

@JsonSerializable()
class AssociateSpeakerRequest {
  @JsonKey(name: 'speaker_id')
  final String speakerId;

  AssociateSpeakerRequest({
    required this.speakerId,
  });

  factory AssociateSpeakerRequest.fromJson(Map<String, dynamic> json) =>
      _$AssociateSpeakerRequestFromJson(json);
  Map<String, dynamic> toJson() => _$AssociateSpeakerRequestToJson(this);
}
