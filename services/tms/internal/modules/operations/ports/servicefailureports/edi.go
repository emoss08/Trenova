package servicefailureports

import (
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/shared/pulid"
)

type ServiceFailureEDITrigger string

func (t ServiceFailureEDITrigger) IsValid() bool {
	switch t {
	case ServiceFailureEDITriggerReviewed, ServiceFailureEDITriggerResolved:
		return true
	default:
		return false
	}
}

const ServiceFailureEDITriggerReviewed ServiceFailureEDITrigger = "Reviewed"

const ServiceFailureEDITriggerResolved ServiceFailureEDITrigger = "Resolved"

type ServiceFailureEDIAction string

type ServiceFailure214LifecycleResult struct {
	Trigger       ServiceFailureEDITrigger `json:"trigger"`
	Action        ServiceFailureEDIAction  `json:"action"`
	MessageID     pulid.ID                 `json:"messageId,omitempty"`
	SkippedReason string                   `json:"skippedReason,omitempty"`
	EDIPartnerID  pulid.ID                 `json:"ediPartnerId,omitempty"`
	Mandatory     bool                     `json:"mandatory"`
	Diagnostics   []edi.Diagnostic         `json:"diagnostics"`
}
