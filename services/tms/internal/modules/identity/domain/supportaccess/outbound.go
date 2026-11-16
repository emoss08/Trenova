package supportaccess

import (
	"context"
	"slices"

	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
)

type OutboundGuard struct {
	Table      string
	Columns    []string
	OnInsert   bool
	OnUpdate   bool
	WhenColumn string
}

var outboundGuards = []OutboundGuard{
	{Table: "email_messages", OnInsert: true},
	{
		Table:      "edi_messages",
		OnInsert:   true,
		OnUpdate:   true,
		WhenColumn: "direction",
	},
	{Table: "factoring_submissions", OnInsert: true, OnUpdate: true},
	{Table: "factoring_transports", OnInsert: true, OnUpdate: true},
	{Table: "detention_notices", OnInsert: true, OnUpdate: true},
	{Table: "inbound_replies", OnInsert: true, OnUpdate: true},
	{Table: "invoice_document_share_tokens", OnInsert: true, OnUpdate: true},
	{Table: "customer_tracking_links", OnInsert: true, OnUpdate: true},
	{Table: "carrier_tracking_links", OnInsert: true, OnUpdate: true},
	{Table: "tender_offer_tokens", OnInsert: true, OnUpdate: true},
	{Table: "rate_confirmation_tokens", OnInsert: true, OnUpdate: true},
	{Table: "report_schedules", OnInsert: true, OnUpdate: true},
	{Table: "document_template_versions", OnInsert: true, OnUpdate: true},
	{
		Table: "invoices",
		Columns: []string{
			"memo", "email_subject_snapshot", "email_body_snapshot", "email_to_snapshot",
			"email_cc_snapshot", "email_bcc_snapshot",
		},
		OnUpdate: true,
	},
	{Table: "invoice_lines", Columns: []string{"description"}, OnUpdate: true},
	{Table: "invoice_lines", OnInsert: true, WhenColumn: "typed_description"},
	{Table: "order_charges", Columns: []string{"description"}, OnInsert: true, OnUpdate: true},
	{Table: "billing_controls", Columns: []string{"default_invoice_footer"}, OnUpdate: true},
	{
		Table:    "customers",
		Columns:  []string{"status_update_recipients"},
		OnInsert: true,
		OnUpdate: true,
	},
	{
		Table: "customer_email_profiles",
		Columns: []string{
			"subject",
			"comment",
			"to_recipients",
			"cc_recipients",
			"bcc_recipients",
		},
		OnInsert: true,
		OnUpdate: true,
	},
}

func OutboundGuards() []OutboundGuard {
	out := make([]OutboundGuard, len(outboundGuards))
	for i, g := range outboundGuards {
		g.Columns = slices.Clone(g.Columns)
		out[i] = g
	}
	return out
}

func ErrOutboundRefused() *errortypes.AuthorizationError {
	return errortypes.NewAuthorizationError(
		"Trenova support cannot send anything outside your organization, create share links or scheduled reports, or change what your customers read on an invoice or in an email",
	)
}

func InitiatedBySupport(ctx context.Context, userID pulid.ID) bool {
	if IsPrincipalID(userID) {
		return true
	}
	tenant, ok := dbscope.TenantFrom(ctx)
	return ok && IsPrincipalID(tenant.UserID)
}
