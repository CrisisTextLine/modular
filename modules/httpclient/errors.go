package httpclient

import (
	"errors"
)

// Error definitions
var (
	// ErrNoSubjectForEventEmission is returned when trying to emit events without a subject
	ErrNoSubjectForEventEmission = errors.New("no subject available for event emission")

	// ErrUnsafeFilename is returned when a URL cannot be safely converted to a filename
	ErrUnsafeFilename = errors.New("URL unsuitable for safe filename")
)
