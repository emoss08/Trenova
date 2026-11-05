package mocks

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/documents/ports/inboundmessageports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

var _ inboundmessageports.TriageEventSink = (*NoopTriageEventSink)(nil)

type NoopTriageEventSink struct{}

func (*NoopTriageEventSink) ShipmentDelivered(
	context.Context,
	pagination.TenantInfo,
	pulid.ID,
) {
}

func (*NoopTriageEventSink) DocumentAttached(
	context.Context,
	pagination.TenantInfo,
	pulid.ID,
	pulid.ID,
) {
}

func (*NoopTriageEventSink) InvoicePaid(context.Context, pagination.TenantInfo, pulid.ID) {}
