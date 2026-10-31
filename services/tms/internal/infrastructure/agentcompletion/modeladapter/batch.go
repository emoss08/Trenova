package modeladapter

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
)

type BatchRequest struct {
	CustomID string
	Request  *Request
}

type BatchSubmission struct {
	ID        string
	Status    string
	ExpiresAt int64
}

type BatchProgress struct {
	Ended      bool
	RawStatus  string
	Processing int
	Succeeded  int
	Errored    int
	Canceled   int
	Expired    int
}

type BatchItem struct {
	CustomID       string
	Response       *Response
	FailureCode    string
	FailureMessage string
	NotRun         bool
}

type BatchRunner interface {
	Batches(provider *aiprovider.Provider) bool
	SubmitBatch(ctx context.Context, call *Call, requests []BatchRequest) (*BatchSubmission, error)
	PollBatch(ctx context.Context, call *Call, id string) (*BatchProgress, error)
	// CancelBatch stops a batch's requests that have not run; the batch
	// then ends with what had already run.
	CancelBatch(ctx context.Context, call *Call, id string) error
	ReadBatch(ctx context.Context, call *Call, id string, each func(*BatchItem) error) error
}
