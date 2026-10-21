package contentscanports

import (
	"context"
	"errors"
	"fmt"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
)

var (
	ErrScannerUnavailable = errors.New("the content scanner could not check the file")
	ErrInfected           = errors.New("the file contains malware")
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
	if errors.Is(err, ErrScannerUnavailable) {
		return errortypes.NewBusinessError(
			"This file could not be checked for malware right now, so it was not accepted. Try again in a few minutes.",
		).WithInternal(err)
	}
	return err
}
