package server

import (
	"encoding/json"
	"io"
	"net/http"

	"hai/api/internal/contacts"
)

// HandleUploadVCards handles POST /api/contacts/upload
func (s *APIServer) HandleUploadVCards(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, &ErrMethodNotAllowed{Method: r.Method})
		return
	}

	// Parse multipart form
	if err := r.ParseMultipartForm(10 << 20); err != nil { // 10MB max
		writeError(w, http.StatusBadRequest, &ErrBadRequest{Message: "failed to parse multipart form: " + err.Error()})
		return
	}

	// Get vCard file
	file, header, err := r.FormFile("vcf")
	if err != nil {
		writeError(w, http.StatusBadRequest, &ErrBadRequest{Message: "vcf file is required: " + err.Error()})
		return
	}
	defer file.Close()

	// Read file content
	vcardData, err := io.ReadAll(file)
	if err != nil {
		writeError(w, http.StatusInternalServerError, &ErrBadRequest{Message: "failed to read file: " + err.Error()})
		return
	}

	// Validate file extension
	if header.Filename != "" {
		ext := header.Filename[len(header.Filename)-4:]
		if ext != ".vcf" && ext != ".vcard" {
			writeError(w, http.StatusBadRequest, &ErrBadRequest{Message: "file must be a .vcf or .vcard file"})
			return
		}
	}

	// Append to default vCard file
	if err := contacts.AppendVCards(contacts.VCardFilePath, vcardData); err != nil {
		writeError(w, http.StatusInternalServerError, &ErrBadRequest{Message: "failed to save vCard file: " + err.Error()})
		return
	}

	// Import contacts from the file
	imported, err := s.contacts.ImportVCardsFromDefault(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, &ErrBadRequest{Message: "failed to import contacts: " + err.Error()})
		return
	}

	// Return imported contacts
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"imported": len(imported),
		"contacts": imported,
	})
}








