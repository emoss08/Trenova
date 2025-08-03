/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package seqgen

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/sequencestore"
	"github.com/emoss08/trenova/pkg/types/pulid"
)

// Format represents the configuration for generating formatted codes
type Format struct {
	// * Type identifies the sequence type
	Type sequencestore.SequenceType

	// * Prefix is the letter prefix (e.g., "S" for shipments, "C" for consolidations)
	Prefix string

	// * IncludeYear determines whether to include year digits
	IncludeYear bool

	// * YearDigits is the number of digits to use for the year (e.g., 2 for "23")
	YearDigits int

	// * IncludeMonth determines whether to include month digits
	IncludeMonth bool

	// * SequenceDigits is the number of digits for the sequence number
	SequenceDigits int

	// * IncludeLocationCode determines whether to include location code
	IncludeLocationCode bool

	// * LocationCode is a code representing the location/region
	LocationCode string

	// * IncludeRandomDigits determines whether to include random digits
	IncludeRandomDigits bool

	// * RandomDigitsCount is the number of random digits to include
	RandomDigitsCount int

	// * IncludeCheckDigit adds a check digit for validation
	IncludeCheckDigit bool

	// * IncludeBusinessUnitCode adds the business unit code
	IncludeBusinessUnitCode bool

	// * BusinessUnitCode is the code representing the business unit
	BusinessUnitCode string

	// * UseSeparators determines whether to use separators
	UseSeparators bool

	// * SeparatorChar is the character to use as a separator
	SeparatorChar string

	// * AllowCustomFormat allows for a custom format string
	AllowCustomFormat bool

	// * CustomFormat is a string with placeholders
	CustomFormat string

	// * IncludeWeekNumber determines whether to include week number instead of month
	IncludeWeekNumber bool

	// * IncludeDay determines whether to include day of month
	IncludeDay bool
}

// SequenceStore interface for storing and retrieving sequences
type SequenceStore interface {
	// GetNextSequence gets the next sequence number, creating if necessary
	GetNextSequence(ctx context.Context, req *SequenceRequest) (int64, error)

	// GetNextSequenceBatch gets a batch of sequence numbers
	GetNextSequenceBatch(ctx context.Context, req *SequenceRequest) ([]int64, error)
}

// SequenceRequest contains the parameters for sequence generation
type SequenceRequest struct {
	Type           sequencestore.SequenceType
	OrganizationID pulid.ID
	BusinessUnitID pulid.ID
	Year           int
	Month          int
	Count          int // For batch requests
}

// FormatProvider interface for retrieving format configurations
type FormatProvider interface {
	// GetFormat retrieves the format configuration for a given type and organization
	GetFormat(
		ctx context.Context,
		sequenceType sequencestore.SequenceType,
		orgID, buID pulid.ID,
	) (*Format, error)
}

// Generator interface for generating formatted codes
type Generator interface {
	// Generate creates a single formatted code
	Generate(ctx context.Context, req *GenerateRequest) (string, error)

	// GenerateBatch creates multiple formatted codes
	GenerateBatch(ctx context.Context, req *GenerateRequest) ([]string, error)
}

// GenerateRequest contains parameters for code generation
type GenerateRequest struct {
	Type           sequencestore.SequenceType
	OrganizationID pulid.ID
	BusinessUnitID pulid.ID
	Count          int // For batch generation
}
