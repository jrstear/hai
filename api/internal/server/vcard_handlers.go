package server

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
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

	// Create a temporary file to import from (don't append to default file)
	tmpFile, err := os.CreateTemp("", "vcard-upload-*.vcf")
	if err != nil {
		writeError(w, http.StatusInternalServerError, &ErrBadRequest{Message: "failed to create temp file: " + err.Error()})
		return
	}
	defer os.Remove(tmpFile.Name()) // Clean up temp file
	defer tmpFile.Close()

	// Write uploaded data to temp file
	if _, err := tmpFile.Write(vcardData); err != nil {
		writeError(w, http.StatusInternalServerError, &ErrBadRequest{Message: "failed to write temp file: " + err.Error()})
		return
	}
	tmpFile.Close() // Close before importing

	// Import contacts directly from the uploaded file (not from default cumulative file)
	imported, err := s.contacts.ImportVCards(r.Context(), tmpFile.Name())
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
