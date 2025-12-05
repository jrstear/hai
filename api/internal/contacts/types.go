package contacts

import "time"

// Contact represents a contact in the system
type Contact struct {
	ID            string    `json:"id"`              // Internal ID (contact_<uuid>)
	ExternalID    string    `json:"external_id"`     // External ID (google:xxx, apple:yyy, vcf:zzz)
	Name          string    `json:"name"`            // Full name
	GivenName     string    `json:"given_name"`      // First name
	FamilyName    string    `json:"family_name"`     // Last name
	Email         string    `json:"email,omitempty"` // Primary email
	Phone         string    `json:"phone,omitempty"` // Primary phone
	PictureURL    string    `json:"picture_url,omitempty"` // Path to picture
	FavoriteColor string    `json:"favorite_color,omitempty"` // Hex color code
	Known         bool      `json:"known"`           // Whether speaker voice is known (computed)
	CreatedAt     time.Time `json:"created_at"`      // When contact was created
	UpdatedAt     time.Time `json:"updated_at"`      // Last update time
	Source        string    `json:"source"`          // Source: "vcf", "google", "apple", "manual"
}

// ContactListResponse represents a paginated list of contacts
type ContactListResponse struct {
	Contacts []Contact `json:"contacts"`
	Total    int       `json:"total"`
}

// AssociateSpeakerRequest represents a request to associate a speaker with a contact
type AssociateSpeakerRequest struct {
	SpeakerID string `json:"speaker_id"`
}







