// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'contact.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

Contact _$ContactFromJson(Map<String, dynamic> json) => Contact(
      id: json['id'] as String,
      externalId: json['externalId'] as String,
      name: json['name'] as String,
      givenName: json['givenName'] as String,
      familyName: json['familyName'] as String,
      email: json['email'] as String?,
      phone: json['phone'] as String?,
      pictureUrl: json['picture_url'] as String?,
      favoriteColor: json['favorite_color'] as String?,
      known: json['known'] as bool,
      createdAt: DateTime.parse(json['created_at'] as String),
      updatedAt: DateTime.parse(json['updated_at'] as String),
      source: json['source'] as String,
    );

Map<String, dynamic> _$ContactToJson(Contact instance) => <String, dynamic>{
      'id': instance.id,
      'externalId': instance.externalId,
      'name': instance.name,
      'givenName': instance.givenName,
      'familyName': instance.familyName,
      'email': instance.email,
      'phone': instance.phone,
      'picture_url': instance.pictureUrl,
      'favorite_color': instance.favoriteColor,
      'known': instance.known,
      'created_at': instance.createdAt.toIso8601String(),
      'updated_at': instance.updatedAt.toIso8601String(),
      'source': instance.source,
    };

ContactListResponse _$ContactListResponseFromJson(Map<String, dynamic> json) =>
    ContactListResponse(
      contacts: (json['contacts'] as List<dynamic>)
          .map((e) => Contact.fromJson(e as Map<String, dynamic>))
          .toList(),
      total: (json['total'] as num).toInt(),
    );

Map<String, dynamic> _$ContactListResponseToJson(
        ContactListResponse instance) =>
    <String, dynamic>{
      'contacts': instance.contacts,
      'total': instance.total,
    };

AssociateSpeakerRequest _$AssociateSpeakerRequestFromJson(
        Map<String, dynamic> json) =>
    AssociateSpeakerRequest(
      speakerId: json['speaker_id'] as String,
    );

Map<String, dynamic> _$AssociateSpeakerRequestToJson(
        AssociateSpeakerRequest instance) =>
    <String, dynamic>{
      'speaker_id': instance.speakerId,
    };
