package notification

import (
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	PreferenceIDPrefix        = "unp_"
	TCAPreferenceKey          = "tca.*"
	tcaEventPrefix            = "tca."
	maxDisabledEventTypes     = 300
	DefaultQuietHoursStart    = "19:00"
	DefaultQuietHoursEnd      = "07:00"
	MentionEventType          = "shipment_comment_mention"
	AIAuditChainMismatchEvent = "ai_audit_chain_mismatch"
)

var (
	eventTypePattern = regexp.MustCompile(`^[a-z0-9_.]+(\.\*)?$`)
	clockPattern     = regexp.MustCompile(`^([01][0-9]|2[0-3]):[0-5][0-9]$`)
)

type DeliveryMode string

const (
	DeliveryModeOff    = DeliveryMode("off")
	DeliveryModeUrgent = DeliveryMode("urgent")
	DeliveryModeAll    = DeliveryMode("all")
)

func DeliveryModes() []DeliveryMode {
	return []DeliveryMode{DeliveryModeOff, DeliveryModeUrgent, DeliveryModeAll}
}

func (m DeliveryMode) IsValid() bool {
	return slices.Contains(DeliveryModes(), m)
}

func (m DeliveryMode) String() string {
	return string(m)
}

func requiredEventTypes() []string {
	return []string{AIAuditChainMismatchEvent}
}

func IsRequiredEventType(eventType string) bool {
	return slices.Contains(requiredEventTypes(), eventType)
}

func urgentEventTypes() []string {
	return []string{
		AIAuditChainMismatchEvent,
		"ai_audit_export_failed",
		"credential_expired",
		"report_run_failed",
		"edi.message.dead_lettered",
		"edi.inbound_file.quarantined",
		"carrier_intel_block",
		"carrier_intel_provider_paused",
		"carrier_equipment_mismatch",
		"carrier_intel_spend_cap",
		"tender_waterfall_exhausted",
		"rate_confirmation_issue_failed",
		"tender_delivery_failed",
	}
}

func IsUrgent(eventType string, priority Priority) bool {
	return priority == PriorityCritical || slices.Contains(urgentEventTypes(), eventType)
}

type Preference struct {
	keel.Table `keel:"user_notification_preferences"`

	ID                    pulid.ID     `json:"id" keel:"id,pk,autoid:unp_"`
	OrganizationID        pulid.ID     `json:"organizationId" keel:"organization_id,pk,tenant"`
	BusinessUnitID        pulid.ID     `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	UserID                pulid.ID     `json:"userId" keel:"user_id"`
	DisabledEventTypes    []string     `json:"disabledEventTypes" keel:"disabled_event_types"`
	DesktopMode           DeliveryMode `json:"desktopMode" keel:"desktop_mode"`
	EmailMode             DeliveryMode `json:"emailMode" keel:"email_mode"`
	EmailDigest           bool         `json:"emailDigest" keel:"email_digest"`
	Sound                 bool         `json:"sound" keel:"sound"`
	QuietHoursEnabled     bool         `json:"quietHoursEnabled" keel:"quiet_hours_enabled"`
	QuietHoursStart       string       `json:"quietHoursStart" keel:"quiet_hours_start"`
	QuietHoursEnd         string       `json:"quietHoursEnd" keel:"quiet_hours_end"`
	QuietHoursDays        []int        `json:"quietHoursDays" keel:"quiet_hours_days"`
	QuietHoursAllowUrgent bool         `json:"quietHoursAllowUrgent" keel:"quiet_hours_allow_urgent"`
	PausedUntil           *int64       `json:"pausedUntil" keel:"paused_until"`
	LastDigestAt          *int64       `json:"lastDigestAt" keel:"last_digest_at"`
	Version               int64        `json:"version" keel:"version,version"`
	CreatedAt             int64        `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt             int64        `json:"updatedAt" keel:"updated_at,autoupdate"`
}

func DefaultPreference(orgID, buID, userID pulid.ID) *Preference {
	return &Preference{
		OrganizationID:        orgID,
		BusinessUnitID:        buID,
		UserID:                userID,
		DisabledEventTypes:    []string{},
		DesktopMode:           DeliveryModeUrgent,
		EmailMode:             DeliveryModeUrgent,
		EmailDigest:           true,
		Sound:                 true,
		QuietHoursEnabled:     false,
		QuietHoursStart:       DefaultQuietHoursStart,
		QuietHoursEnd:         DefaultQuietHoursEnd,
		QuietHoursDays:        []int{1, 2, 3, 4, 5},
		QuietHoursAllowUrgent: true,
	}
}

func (p *Preference) Normalize() {
	seen := make(map[string]struct{}, len(p.DisabledEventTypes))
	types := make([]string, 0, len(p.DisabledEventTypes))
	for _, raw := range p.DisabledEventTypes {
		eventType := strings.TrimSpace(raw)
		if eventType == "" {
			continue
		}
		if _, dup := seen[eventType]; dup {
			continue
		}
		seen[eventType] = struct{}{}
		types = append(types, eventType)
	}
	slices.Sort(types)
	p.DisabledEventTypes = types

	days := slices.Clone(p.QuietHoursDays)
	slices.Sort(days)
	p.QuietHoursDays = slices.Compact(days)
	if p.QuietHoursDays == nil {
		p.QuietHoursDays = []int{}
	}
}

func (p *Preference) Validate(multiErr *errortypes.MultiError) {
	if p.UserID.IsNil() {
		multiErr.Add("userId", errortypes.ErrRequired, "User is required")
	}
	if !p.DesktopMode.IsValid() {
		multiErr.Add("desktopMode", errortypes.ErrInvalid, "Desktop must be off, urgent or all")
	}
	if !p.EmailMode.IsValid() {
		multiErr.Add("emailMode", errortypes.ErrInvalid, "Email must be off, urgent or all")
	}
	if !clockPattern.MatchString(p.QuietHoursStart) {
		multiErr.Add("quietHoursStart", errortypes.ErrInvalidFormat, "Start must be a time like 19:00")
	}
	if !clockPattern.MatchString(p.QuietHoursEnd) {
		multiErr.Add("quietHoursEnd", errortypes.ErrInvalidFormat, "End must be a time like 07:00")
	}
	if p.QuietHoursEnabled && p.QuietHoursStart == p.QuietHoursEnd {
		multiErr.Add("quietHoursEnd", errortypes.ErrInvalid, "Quiet hours must end at a different time than they start")
	}
	if p.QuietHoursEnabled && len(p.QuietHoursDays) == 0 {
		multiErr.Add("quietHoursDays", errortypes.ErrRequired, "Pick at least one day for quiet hours")
	}
	for index, day := range p.QuietHoursDays {
		if day < 0 || day > 6 {
			multiErr.Add(
				fmt.Sprintf("quietHoursDays[%d]", index),
				errortypes.ErrInvalid,
				"Days run from 0 (Sunday) to 6 (Saturday)",
			)
		}
	}
	if len(p.DisabledEventTypes) > maxDisabledEventTypes {
		multiErr.Add("disabledEventTypes", errortypes.ErrInvalid, "Too many notification types")
	}
	for index, eventType := range p.DisabledEventTypes {
		field := fmt.Sprintf("disabledEventTypes[%d]", index)
		switch {
		case len(eventType) > maxEventTypeLength || !eventTypePattern.MatchString(eventType):
			multiErr.Add(field, errortypes.ErrInvalidFormat, "Not a notification type Trenova sends")
		case IsRequiredEventType(eventType):
			multiErr.Add(field, errortypes.ErrInvalid, "Your organization requires this notification")
		}
	}
}

func (p *Preference) IsPaused(now int64) bool {
	return p.PausedUntil != nil && *p.PausedUntil > now
}

func (p *Preference) IsEventTypeEnabled(eventType string) bool {
	if IsRequiredEventType(eventType) {
		return true
	}
	if slices.Contains(p.DisabledEventTypes, eventType) {
		return false
	}
	if strings.HasPrefix(eventType, tcaEventPrefix) &&
		slices.Contains(p.DisabledEventTypes, TCAPreferenceKey) {
		return false
	}
	return true
}

func parseClock(value string) (int, bool) {
	if !clockPattern.MatchString(value) {
		return 0, false
	}
	hours, _ := strconv.Atoi(value[:2])
	minutes, _ := strconv.Atoi(value[3:])
	return hours*60 + minutes, true
}
