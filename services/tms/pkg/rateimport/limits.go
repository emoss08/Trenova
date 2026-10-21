package rateimport

import (
	"errors"
	"fmt"

	"github.com/emoss08/trenova/pkg/errortypes"
)

const (
	DefaultMaxRows          = 100_000
	DefaultMaxColumns       = 256
	DefaultMaxCellBytes     = 32 << 10
	DefaultMaxSheetBytes    = 64 << 20
	DefaultMaxUnzippedBytes = 256 << 20
	DefaultMaxPartBytes     = 64 << 20
)

// Limits bounds what reading one uploaded sheet may cost. A zero field takes
// its default.
type Limits struct {
	MaxRows          int
	MaxColumns       int
	MaxCellBytes     int
	MaxSheetBytes    int
	MaxUnzippedBytes int64
	MaxPartBytes     int64
}

func (l Limits) withDefaults() Limits {
	if l.MaxRows <= 0 {
		l.MaxRows = DefaultMaxRows
	}
	if l.MaxColumns <= 0 {
		l.MaxColumns = DefaultMaxColumns
	}
	if l.MaxCellBytes <= 0 {
		l.MaxCellBytes = DefaultMaxCellBytes
	}
	if l.MaxSheetBytes <= 0 {
		l.MaxSheetBytes = DefaultMaxSheetBytes
	}
	if l.MaxUnzippedBytes <= 0 {
		l.MaxUnzippedBytes = DefaultMaxUnzippedBytes
	}
	if l.MaxPartBytes <= 0 {
		l.MaxPartBytes = DefaultMaxPartBytes
	}
	l.MaxPartBytes = min(l.MaxPartBytes, l.MaxUnzippedBytes)
	return l
}

type LimitKind string

const (
	LimitRows      LimitKind = "rows"
	LimitColumns   LimitKind = "columns"
	LimitCellBytes LimitKind = "cell"
	LimitSheet     LimitKind = "sheet"
)

var ErrLimitExceeded = errors.New("the file is larger than an import may be")

// LimitError says which limit a file went past, and at which line.
type LimitError struct {
	Kind LimitKind
	Max  int
	Line int
}

func (e *LimitError) Error() string {
	switch e.Kind {
	case LimitRows:
		return fmt.Sprintf("this file has more than %d rows; split it into smaller files", e.Max)
	case LimitColumns:
		return fmt.Sprintf("line %d has more than %d columns", e.Line, e.Max)
	case LimitCellBytes:
		return fmt.Sprintf("line %d has a cell longer than %d bytes", e.Line, e.Max)
	case LimitSheet:
		return fmt.Sprintf("this file holds more than %d bytes of text", e.Max)
	}
	return ErrLimitExceeded.Error()
}

func (e *LimitError) Unwrap() error {
	return ErrLimitExceeded
}

// ValidationError turns a limit a file went past into the field error the
// person uploading it is shown. Any other error is returned as it was.
func ValidationError(field string, err error) error {
	var limit *LimitError
	if !errors.As(err, &limit) {
		return err
	}
	switch limit.Kind {
	case LimitRows:
		return errortypes.NewValidationError(
			field,
			errortypes.ErrInvalid,
			"This file has more than {0} rows. Split it into smaller files and import each one.",
			limit.Max,
		)
	case LimitColumns:
		return errortypes.NewValidationError(
			field,
			errortypes.ErrInvalid,
			"Line {0} of this file has more than {1} columns.",
			limit.Line,
			limit.Max,
		)
	case LimitCellBytes:
		return errortypes.NewValidationError(
			field,
			errortypes.ErrInvalid,
			"Line {0} of this file has a cell longer than {1} bytes.",
			limit.Line,
			limit.Max,
		)
	case LimitSheet:
		return errortypes.NewValidationError(
			field,
			errortypes.ErrInvalid,
			"This file holds more than {0} bytes of text. Split it into smaller files and import each one.",
			limit.Max,
		)
	}
	return err
}

type recordCollector struct {
	limits  Limits
	records [][]string
	bytes   int
}

func newRecordCollector(limits Limits) *recordCollector {
	return &recordCollector{limits: limits, records: make([][]string, 0, 64)}
}

func (c *recordCollector) add(record []string) error {
	line := len(c.records) + 1
	if line > c.limits.MaxRows {
		return &LimitError{Kind: LimitRows, Max: c.limits.MaxRows, Line: line}
	}
	if len(record) > c.limits.MaxColumns {
		return &LimitError{Kind: LimitColumns, Max: c.limits.MaxColumns, Line: line}
	}
	for _, cell := range record {
		if len(cell) > c.limits.MaxCellBytes {
			return &LimitError{Kind: LimitCellBytes, Max: c.limits.MaxCellBytes, Line: line}
		}
		c.bytes += len(cell)
	}
	if c.bytes > c.limits.MaxSheetBytes {
		return &LimitError{Kind: LimitSheet, Max: c.limits.MaxSheetBytes, Line: line}
	}
	c.records = append(c.records, record)
	return nil
}
