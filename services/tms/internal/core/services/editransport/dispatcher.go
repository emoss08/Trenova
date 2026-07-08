package editransport

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"go.uber.org/fx"
)

type DispatcherParams struct {
	fx.In

	Transports []services.EDITransport `group:"edi_transports"`
}

type Dispatcher struct {
	transports map[edi.ConnectionMethod]services.EDITransport
}

func NewDispatcher(p DispatcherParams) *Dispatcher {
	transports := make(map[edi.ConnectionMethod]services.EDITransport, len(p.Transports))
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
	req *services.EDITransportRequest,
) ([]services.EDIConnectionCheck, error) {
	transport, ok := d.transports[method]
	if !ok {
		return nil, fmt.Errorf(
			"EDI connection testing is not supported for connection method %s",
			method,
		)
	}
	tester, ok := transport.(services.EDIConnectionTester)
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
	req *services.EDITransportRequest,
) (*services.EDITransportResult, error) {
	transport, ok := d.transports[method]
	if !ok {
		return nil, fmt.Errorf("EDI delivery is not supported for connection method %s", method)
	}

	return transport.Deliver(ctx, req)
}

func (d *Dispatcher) FetchInbound(
	ctx context.Context,
	method edi.ConnectionMethod,
	req *services.EDIInboundFetchRequest,
) ([]*services.EDIInboundRemoteFile, error) {
	fetcher, err := d.fetcherFor(method)
	if err != nil {
		return nil, err
	}

	return fetcher.FetchInboundFiles(ctx, req)
}

func (d *Dispatcher) ArchiveInbound(
	ctx context.Context,
	method edi.ConnectionMethod,
	req *services.EDIInboundFetchRequest,
	remotePath string,
) error {
	fetcher, err := d.fetcherFor(method)
	if err != nil {
		return err
	}

	return fetcher.ArchiveInboundFile(ctx, req, remotePath)
}

func (d *Dispatcher) fetcherFor(method edi.ConnectionMethod) (services.EDIMailboxFetcher, error) {
	transport, ok := d.transports[method]
	if !ok {
		return nil, fmt.Errorf(
			"EDI inbound polling is not supported for connection method %s",
			method,
		)
	}

	fetcher, ok := transport.(services.EDIMailboxFetcher)
	if !ok {
		return nil, fmt.Errorf(
			"EDI inbound polling is not supported for connection method %s",
			method,
		)
	}

	return fetcher, nil
}
