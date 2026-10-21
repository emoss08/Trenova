package contentscanports

import (
	"context"
	"errors"
	"fmt"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/fileutils"
)

var (
	ErrScannerUnavailable = errors.New("the content scanner could not check the file")
	ErrInfected           = errors.New("the file contains malware")
	ErrTooLarge           = errors.New("the file is larger than the content scanner can check")
)

type Verdict string

const (
	VerdictClean     = Verdict("Clean")
	VerdictInfected  = Verdict("Infected")
	VerdictUnscanned = Verdict("Unscanned")
)

type Source string

const (
	SourceUpload            = Source("upload")
	SourceInboundAttachment = Source("inbound_attachment")
	SourceCapturePage       = Source("capture_page")
	SourceEDIInboundFile    = Source("edi_inbound_file")
	SourceGeneratedDocument = Source("generated_document")
)

type ScanRequest struct {
	TenantInfo pagination.TenantInfo
	Source     Source
	FileName   string
	Content    []byte
}

type ScanResult struct {
	Verdict   Verdict
	Signature string
}

type InfectedError struct {
	Signature string
}

func (e *InfectedError) Error() string {
	return fmt.Sprintf("%s: %s", ErrInfected, e.Signature)
}

func (e *InfectedError) Unwrap() error {
	return ErrInfected
}

type TooLargeError struct {
	Size  int64
	Limit int64
}

func (e *TooLargeError) Error() string {
	if e.Limit <= 0 {
		return fmt.Sprintf("%s: %d bytes", ErrTooLarge, e.Size)
	}
	return fmt.Sprintf("%s: %d bytes, at most %d", ErrTooLarge, e.Size, e.Limit)
}

func (e *TooLargeError) Unwrap() error {
	return ErrTooLarge
}

type Scanner interface {
	Scan(ctx context.Context, content []byte) (*ScanResult, error)
	Ping(ctx context.Context) error
}

type Gate interface {
	Check(ctx context.Context, req *ScanRequest) (*ScanResult, error)
}

func Check(ctx context.Context, gate Gate, req *ScanRequest) (*ScanResult, error) {
	if gate == nil {
		return &ScanResult{Verdict: VerdictUnscanned}, nil
	}
	return gate.Check(ctx, req)
}

func RefusalError(field string, err error) error {
	var infected *InfectedError
	if errors.As(err, &infected) {
		return errortypes.NewValidationError(
			field,
			errortypes.ErrInvalid,
			"This file was refused: the malware scanner found {0} in it.",
			infected.Signature,
		)
	}
	var tooLarge *TooLargeError
	if errors.As(err, &tooLarge) {
		if tooLarge.Limit <= 0 {
			return errortypes.NewValidationError(
				field,
				errortypes.ErrInvalid,
				"This file is too large to be checked for malware, so it was not accepted.",
			)
		}
		return errortypes.NewValidationError(
			field,
			errortypes.ErrInvalid,
			"This file is too large to be checked for malware. Files up to {0} can be accepted.",
			fileutils.HumanizeBytes(tooLarge.Limit),
		)
	}
	if errors.Is(err, ErrScannerUnavailable) {
		return errortypes.NewBusinessError(
			"This file could not be checked for malware right now, so it was not accepted. Try again in a few minutes.",
		).WithInternal(err)
	}
	return err
}
