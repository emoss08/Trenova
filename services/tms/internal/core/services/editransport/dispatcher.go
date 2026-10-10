package editransport

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/internal/modules/edi/ports/ediports"
	"go.uber.org/fx"
)

type DispatcherParams struct {
	fx.In

	Transports []ediports.EDITransport `group:"edi_transports"`
}

type Dispatcher struct {
	transports map[edi.ConnectionMethod]ediports.EDITransport
}

func NewDispatcher(p DispatcherParams) *Dispatcher {
	transports := make(map[edi.ConnectionMethod]ediports.EDITransport, len(p.Transports))
	for _, transport := range p.Transports {
		if transport == nil {
			continue
		}
		transports[transport.Method()] = transport
	}

	return &Dispatcher{transports: transports}
}

func (d *Dispatcher) Supports(method edi.ConnectionMethod) bool {
	_, ok := d.transports[method]
	return ok
}

func (d *Dispatcher) TestConnection(
	ctx context.Context,
	method edi.ConnectionMethod,
	req *ediports.EDITransportRequest,
) ([]ediports.EDIConnectionCheck, error) {
	transport, ok := d.transports[method]
	if !ok {
		return nil, fmt.Errorf(
			"EDI connection testing is not supported for connection method %s",
			method,
		)
	}
	tester, ok := transport.(ediports.EDIConnectionTester)
	if !ok {
		return nil, fmt.Errorf(
			"EDI connection testing is not supported for connection method %s",
			method,
		)
	}
	return tester.TestConnection(ctx, req), nil
}

func (d *Dispatcher) Deliver(
	ctx context.Context,
	method edi.ConnectionMethod,
	req *ediports.EDITransportRequest,
) (*ediports.EDITransportResult, error) {
	transport, ok := d.transports[method]
	if !ok {
		return nil, fmt.Errorf("EDI delivery is not supported for connection method %s", method)
	}

	return transport.Deliver(ctx, req)
}

func (d *Dispatcher) FetchInbound(
	ctx context.Context,
	method edi.ConnectionMethod,
	req *ediports.EDIInboundFetchRequest,
) ([]*ediports.EDIInboundRemoteFile, error) {
	fetcher, err := d.fetcherFor(method)
	if err != nil {
		return nil, err
	}

	return fetcher.FetchInboundFiles(ctx, req)
}

func (d *Dispatcher) ArchiveInbound(
	ctx context.Context,
	method edi.ConnectionMethod,
	req *ediports.EDIInboundFetchRequest,
	remotePath string,
) error {
	fetcher, err := d.fetcherFor(method)
	if err != nil {
		return err
	}

	return fetcher.ArchiveInboundFile(ctx, req, remotePath)
}

func (d *Dispatcher) fetcherFor(method edi.ConnectionMethod) (ediports.EDIMailboxFetcher, error) {
	transport, ok := d.transports[method]
	if !ok {
		return nil, fmt.Errorf(
			"EDI inbound polling is not supported for connection method %s",
			method,
		)
	}

	fetcher, ok := transport.(ediports.EDIMailboxFetcher)
	if !ok {
		return nil, fmt.Errorf(
			"EDI inbound polling is not supported for connection method %s",
			method,
		)
	}

	return fetcher, nil
}
