package airetrieval

import (
	"errors"
	"fmt"
	"slices"
)

const (
	Dimensions768  = 768
	Dimensions1024 = 1024
	Dimensions1536 = 1536

	MaxModelKeyChars    = 300
	ContentHashChars    = 64
	MaxCatalogItemChars = 200
)

var allowedDimensions = []int{Dimensions768, Dimensions1024, Dimensions1536}

func AllowedDimensions() []int { return slices.Clone(allowedDimensions) }

func IsAllowedDimension(dimensions int) bool {
	return slices.Contains(allowedDimensions, dimensions)
}

var (
	ErrVectorUnavailable     = errors.New("semantic retrieval storage is unavailable")
	ErrInvalidStorageRequest = errors.New("invalid semantic retrieval storage request")
	ErrUnsupportedDimensions = errors.New("embedding dimensions must be 768, 1024 or 1536")
	ErrEmbeddingLength       = errors.New("embedding length does not match its dimensions")
	ErrChunkEmbeddingMissing = errors.New(
		"a chunk without an embedding must already be stored with the same content hash",
	)
	ErrModelInUse = errors.New(
		"the embedding model is active or pending and cannot be purged",
	)
	ErrNoPendingModel    = errors.New("no embedding model change is pending")
	ErrPendingModelMoved = errors.New("the pending embedding model changed before the swap")
)

type UnavailableError struct {
	Reason           UnavailableReason
	ExtensionVersion string
}

func (e *UnavailableError) Error() string {
	if e.ExtensionVersion != "" {
		return fmt.Sprintf(
			"%s: %s (pgvector %s)",
			ErrVectorUnavailable.Error(),
			e.Reason,
			e.ExtensionVersion,
		)
	}

	return fmt.Sprintf("%s: %s", ErrVectorUnavailable.Error(), e.Reason)
}

func (e *UnavailableError) Is(target error) bool {
	return target == ErrVectorUnavailable
}

type Availability struct {
	Available          bool              `json:"available"`
	Reason             UnavailableReason `json:"reason,omitempty"`
	ExtensionInstalled bool              `json:"extensionInstalled"`
	ExtensionVersion   string            `json:"extensionVersion,omitempty"`
}

func (a Availability) Err() error {
	if a.Available {
		return nil
	}

	return &UnavailableError{Reason: a.Reason, ExtensionVersion: a.ExtensionVersion}
}
