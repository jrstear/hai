package export2elastic

import "fmt"

// ErrSpeakerNotFound is returned when a local speaker ID cannot be mapped to a global speaker ID
type ErrSpeakerNotFound struct {
	LocalSpeakerID string
}

func (e *ErrSpeakerNotFound) Error() string {
	return fmt.Sprintf("speaker not found in mapping: %s", e.LocalSpeakerID)
}











