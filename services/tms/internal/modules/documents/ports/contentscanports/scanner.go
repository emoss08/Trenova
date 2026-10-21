package contentscanports

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"

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
	SourceFuelCardFile      = Source("fuel_card_file")
)

type ScanRequest struct {
	TenantInfo pagination.TenantInfo
	Source     Source
	FileName   string
	Content    []byte
}

type StreamRequest struct {
	TenantInfo pagination.TenantInfo
	Source     Source
	FileName   string
	Size       int64
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
	Scan(ctx context.Context, content io.Reader, size int64) (*ScanResult, error)
	Ping(ctx context.Context) error
}

type Stream interface {
	io.Writer
	Finish() (*ScanResult, error)
	Abort()
}

type Gate interface {
	Check(ctx context.Context, req *ScanRequest) (*ScanResult, error)
	Stream(ctx context.Context, req *StreamRequest) (Stream, error)
}

func Check(ctx context.Context, gate Gate, req *ScanRequest) (*ScanResult, error) {
	if gate == nil {
		return &ScanResult{Verdict: VerdictUnscanned}, nil
	}
	return gate.Check(ctx, req)
}

func Open(ctx context.Context, gate Gate, req *StreamRequest) (Stream, error) {
	if gate == nil {
		return unscanned{}, nil
	}
	return gate.Stream(ctx, req)
}

func CheckWith(ctx context.Context, gate Gate, req *ScanRequest) (*ScanResult, error) {
	stream, err := gate.Stream(ctx, &StreamRequest{
		TenantInfo: req.TenantInfo,
		Source:     req.Source,
		FileName:   req.FileName,
		Size:       int64(len(req.Content)),
	})
	if err != nil {
		return nil, err
	}
	if _, err = io.Copy(stream, bytes.NewReader(req.Content)); err != nil {
		stream.Abort()
		return nil, err
	}
	return stream.Finish()
}

type unscanned struct{}

func (unscanned) Write(p []byte) (int, error) { return len(p), nil }

func (unscanned) Finish() (*ScanResult, error) { return &ScanResult{Verdict: VerdictUnscanned}, nil }

func (unscanned) Abort() {}

type Gated struct {
	src    io.Reader
	stream Stream
	result *ScanResult
	err    error
}

func NewGated(src io.Reader, stream Stream) *Gated {
	return &Gated{src: src, stream: stream}
}

func (g *Gated) Read(p []byte) (int, error) {
	if g.err != nil {
		return 0, g.err
	}
	n, err := g.src.Read(p)
	if n > 0 {
		if _, writeErr := g.stream.Write(p[:n]); writeErr != nil {
			g.err = writeErr
			return 0, writeErr
		}
	}
	if errors.Is(err, io.EOF) {
		result, finishErr := g.stream.Finish()
		if finishErr != nil {
			g.err = finishErr
			return n, finishErr
		}
		g.result = result
		g.err = io.EOF
		return n, io.EOF
	}
	if err != nil {
		g.stream.Abort()
		g.err = err
	}
	return n, err
}

func (g *Gated) Result() *ScanResult {
	return g.result
}

func (g *Gated) Abort() {
	if g.err == nil {
		g.stream.Abort()
	}
}

func (g *Gated) Err() error {
	if errors.Is(g.err, io.EOF) {
		return nil
	}
	return g.err
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
