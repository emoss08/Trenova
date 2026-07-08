package services

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/edi"
)

type EDITransportRequest struct {
	Profile  *edi.EDICommunicationProfile
	Secrets  map[string]string
	FileName string
	Contents string
}

type EDITransportResult struct {
	RemotePath string
	MessageID  string
	MIC        string
	Pending    bool
}

type EDITransport interface {
	Method() edi.ConnectionMethod
	Deliver(ctx context.Context, req *EDITransportRequest) (*EDITransportResult, error)
}

type EDIInboundFetchRequest struct {
	Profile *edi.EDICommunicationProfile
	Secrets map[string]string
}

type EDIInboundRemoteFile struct {
	Path     string
	Name     string
	Contents string
	Size     int64
}

type EDIMailboxFetcher interface {
	FetchInboundFiles(
		ctx context.Context,
		req *EDIInboundFetchRequest,
	) ([]*EDIInboundRemoteFile, error)
	ArchiveInboundFile(
		ctx context.Context,
		req *EDIInboundFetchRequest,
		remotePath string,
	) error
}

type EDITransportDispatcher interface {
	Supports(method edi.ConnectionMethod) bool
	Deliver(
		ctx context.Context,
		method edi.ConnectionMethod,
		req *EDITransportRequest,
	) (*EDITransportResult, error)
	FetchInbound(
		ctx context.Context,
		method edi.ConnectionMethod,
		req *EDIInboundFetchRequest,
	) ([]*EDIInboundRemoteFile, error)
	ArchiveInbound(
		ctx context.Context,
		method edi.ConnectionMethod,
		req *EDIInboundFetchRequest,
		remotePath string,
	) error
}
