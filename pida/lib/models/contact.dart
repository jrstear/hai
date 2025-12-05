import 'package:json_annotation/json_annotation.dart';

part 'contact.g.dart';

@JsonSerializable()
class Contact {
  final String id;
  @JsonKey(name: 'externalId')
  final String externalId;
  final String name;
  @JsonKey(name: 'givenName')
  final String givenName;
  @JsonKey(name: 'familyName')
  final String familyName;
  final String? email;
  final String? phone;
  @JsonKey(name: 'picture_url')
  final String? pictureUrl;
  @JsonKey(name: 'favorite_color')
  final String? favoriteColor;
  final bool known;
  @JsonKey(name: 'created_at')
  final DateTime createdAt;
  @JsonKey(name: 'updated_at')
  final DateTime updatedAt;
  final String source;

  Contact({
    required this.id,
    required this.externalId,
    required this.name,
    required this.givenName,
    required this.familyName,
    this.email,
    this.phone,
    this.pictureUrl,
    this.favoriteColor,
    required this.known,
    required this.createdAt,
    required this.updatedAt,
    required this.source,
  });

  factory Contact.fromJson(Map<String, dynamic> json) => _$ContactFromJson(json);
  Map<String, dynamic> toJson() => _$ContactToJson(this);
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

