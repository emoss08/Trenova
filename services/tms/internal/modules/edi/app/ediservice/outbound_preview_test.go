package ediservice

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/edi/app/edix12"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi/templates"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func resolved214(t *testing.T, bol string) *resolvedDocumentContext {
	t.Helper()

	tenant := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	versionID := pulid.MustNew("editplv_")
	envelope := edi.DefaultX12EnvelopeSettings()
	envelope.InterchangeReceiverID = "ACMERETAIL"
	profile := &edi.EDIPartnerDocumentProfile{
		ID:                pulid.MustNew("edipdp_"),
		EDIPartnerID:      pulid.MustNew("edip_"),
		Direction:         edi.DocumentDirectionOutbound,
		Standard:          edi.EDIStandardX12,
		TransactionSet:    edi.TransactionSet214,
		FunctionalGroupID: "QM",
		Envelope:          envelope,
		ValidationMode:    edi.ValidationModeStrict,
		PartnerSettings:   map[string]any{"carrier": map[string]any{"scac": "TRNV"}},
	}
	version := &edi.EDITemplateVersion{
		ID:                versionID,
		X12Version:        edi.DefaultX12204Version,
		FunctionalGroupID: "QM",
		Status:            edi.TemplateStatusActive,
		Segments:          templates.Base214Segments(tenant, versionID),
	}
	payload := edi.DocumentPayload{
		TransactionSet: edi.TransactionSet214,
		ShipmentStatus: &edi.ShipmentStatusPayload{
			ShipmentID:       pulid.MustNew("shp_"),
			BOL:              bol,
			ShipperReference: bol,
			ProNumber:        "PRO-88",
			StatusCode:       "X3",
			EventDate:        1_790_000_000,
			EventTime:        1_790_000_000,
			StopSequence:     1,
			LocationName:     "Acme DC",
			City:             "Dallas",
			StateCode:        "TX",
			PostalCode:       "75201",
			CountryCode:      "US",
		},
	}
	payload.Normalize()
	for _, segment := range version.Segments {
		for idx := range segment.Elements {
			if segment.Elements[idx].FieldPath == "shipmentStatus.shipperReference" {
				segment.Elements[idx].Validation.Required = true
			}
		}
	}

	return &resolvedDocumentContext{
		ctx:             t.Context(),
		profile:         profile,
		templateVersion: version,
		payload:         payload,
		x12Version:      edi.DefaultX12204Version,
		runtime:         edix12.RuntimeValues(profile, edi.DefaultX12204Version),
	}
}

func TestOutboundPreview_IsWhatTheSendPlanRendersWhenItWouldGo(t *testing.T) {
	t.Parallel()

	resolved := resolved214(t, "BOL-88")
	runtimeBefore := len(resolved.runtime)

	preview, err := resolved.outboundPreview()
	require.NoError(t, err)
	assert.False(t, preview.Blocking)
	assert.Contains(t, preview.Document.RawX12, "ST*214*")
	assert.Contains(t, preview.Document.RawX12, "BOL-88")
	assert.Len(t, resolved.runtime, runtimeBefore, "a preview leaves the send's runtime alone")

	planned, err := resolved.provisionalRender()
	require.NoError(t, err)
	assert.Equal(t, preview.Document.RawX12, planned.RawX12,
		"the preview and the send's plan render the same document")
	assert.Equal(t, preview.Document.Diagnostics, planned.Diagnostics)
}

func TestOutboundPreview_ShowsWhatWouldStopTheSendInsteadOfFailing(t *testing.T) {
	t.Parallel()

	resolved := resolved214(t, "")

	preview, err := resolved.outboundPreview()
	require.NoError(t, err)
	assert.True(t, preview.Blocking, "a strict profile stops a document with an error")
	assert.True(t, edix12.HasBlockingDiagnostics(preview.Document.Diagnostics, edi.ValidationModeStrict))
	assert.True(t, strings.Contains(preview.Document.RawX12, "B10*"),
		"the document is still rendered for the person to read")

	_, err = resolved.provisionalRender()
	require.Error(t, err, "the send's plan refuses what the preview marks as blocking")
}
