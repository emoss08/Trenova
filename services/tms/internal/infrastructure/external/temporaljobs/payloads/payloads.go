/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package payloads

import "github.com/emoss08/trenova/shared/pulid"

// BasePayload contains common fields for all workflow payloads
type BasePayload struct {
	JobID          string         `json:"jobId"`
	OrganizationID pulid.ID       `json:"organizationId"`
	BusinessUnitID pulid.ID       `json:"businessUnitId"`
	UserID         pulid.ID       `json:"userId,omitempty"`
	Timestamp      int64          `json:"timestamp"`
	Metadata       map[string]any `json:"metadata,omitempty"`
}

// EmailPayload represents the payload for email workflows
type EmailPayload struct {
	BasePayload
	To       []string `json:"to"`
	Cc       []string `json:"cc,omitempty"`
	Bcc      []string `json:"bcc,omitempty"`
	Subject  string   `json:"subject"`
	Body     string   `json:"body"`
	BodyHTML string   `json:"bodyHtml,omitempty"`
	Priority string   `json:"priority,omitempty"`
}

// PatternAnalysisPayload represents the payload for pattern analysis
type PatternAnalysisPayload struct {
	BasePayload
	MinFrequency   int    `json:"minFrequency"`
	TriggerReason  string `json:"triggerReason"`
	LaneID         string `json:"laneId,omitempty"`
}

// ExpireSuggestionsPayload represents the payload for expiring suggestions
type ExpireSuggestionsPayload struct {
	BasePayload
	BatchSize int `json:"batchSize"`
}

// ComplianceCheckPayload represents the payload for compliance checks
type ComplianceCheckPayload struct {
	BasePayload
	CheckType string `json:"checkType"` // "all", "dot", "hazmat"
}

// DuplicateShipmentPayload for duplicating shipments
type DuplicateShipmentPayload struct {
	BasePayload
	ShipmentID               pulid.ID `json:"shipmentId"`
	Count                    int      `json:"count"`
	OverrideDates            bool     `json:"overrideDates"`
	IncludeCommodities       bool     `json:"includeCommodities"`
	IncludeAdditionalCharges bool     `json:"includeAdditionalCharges"`
}

// JobCompletionNotificationPayload for job completion notifications
type JobCompletionNotificationPayload struct {
	BasePayload
	JobID          string         `json:"jobId"`
	JobType        string         `json:"jobType"`
	Success        bool           `json:"success"`
	Result         string         `json:"result"`
	Error          string         `json:"error,omitempty"`
	OriginalEntity string         `json:"originalEntity,omitempty"`
	Data           map[string]any `json:"data,omitempty"`
}