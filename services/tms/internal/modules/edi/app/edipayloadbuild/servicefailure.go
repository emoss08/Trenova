package edipayloadbuild

import (
	"strings"

	"github.com/emoss08/trenova/internal/modules/edi/domain/edidocument"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edipayload"
)

func NormalizedX12Code(value string) string {
	return strings.ToUpper(strings.TrimSpace(value))
}

func serviceFailure214Diagnostics(
	payload *edipayload.ShipmentStatusPayload,
) []edidocument.Diagnostic {
	if payload == nil {
		return []edidocument.Diagnostic{requiredServiceFailure214Diagnostic(
			"shipmentStatus",
			"EDI 214 shipment status payload is required",
			"Build the shipment status payload before generating the 214.",
		)}
	}
	diagnostics := make([]edidocument.Diagnostic, 0, 6)
	statusCode := NormalizedX12Code(payload.StatusCode)
	reasonCode := NormalizedX12Code(payload.StatusReasonCode)
	if payload.EventDate == 0 {
		diagnostics = append(diagnostics, requiredServiceFailure214Diagnostic(
			"shipmentStatus.eventDate",
			"Service failure 214 event date is required",
			"Set the service failure detection, creation, or actual arrival timestamp before generating the 214.",
		))
	}
	if payload.EventTime == 0 {
		diagnostics = append(diagnostics, requiredServiceFailure214Diagnostic(
			"shipmentStatus.eventTime",
			"Service failure 214 event time is required",
			"Set the service failure detection, creation, or actual arrival timestamp before generating the 214.",
		))
	}
	if statusCode == "SD" && reasonCode == "" {
		diagnostics = append(diagnostics, requiredServiceFailure214Diagnostic(
			"shipmentStatus.statusReasonCode",
			"X12 214 service failure status code SD requires a status reason code",
			"Set an override reason code or configure a default reason code on the service failure reason code.",
		))
	}
	return diagnostics
}

func requiredServiceFailure214Diagnostic(
	path, message, suggestedFix string,
) edidocument.Diagnostic {
	diagnostic := edidocument.Diagnostic{
		Severity:     edidocument.ValidationSeverityError,
		Code:         "required",
		SegmentID:    "AT7",
		Path:         path,
		Message:      message,
		SuggestedFix: suggestedFix,
	}
	if path == "shipmentStatus.statusReasonCode" {
		diagnostic.ElementPosition = 2
	}
	return diagnostic
}
