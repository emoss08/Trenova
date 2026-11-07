package audit

import (
	"regexp"
	"strings"

	"github.com/emoss08/trenova/pkg/auditctx"
)

type Category string

const (
	CategorySystem   = Category("System")
	CategoryUser     = Category("User")
	CategorySecurity = Category("Security")
)

func (c Category) Valid() bool {
	switch c {
	case CategorySystem, CategoryUser, CategorySecurity:
		return true
	}
	return false
}

type RetentionClass string

const (
	RetentionAccess      = RetentionClass("Access")
	RetentionOperational = RetentionClass("Operational")
	RetentionFinancial   = RetentionClass("Financial")
	RetentionSecurity    = RetentionClass("Security")
)

const (
	AccessRetentionFloorDays      = 90
	OperationalRetentionFloorDays = 730
	FinancialRetentionFloorDays   = 2555
	SecurityRetentionFloorDays    = 730
	MaxRetentionDays              = 3650
)

var RetentionClasses = []RetentionClass{
	RetentionAccess,
	RetentionOperational,
	RetentionFinancial,
	RetentionSecurity,
}

func (c RetentionClass) Valid() bool {
	_, ok := retentionFloors[c]
	return ok
}

var retentionFloors = map[RetentionClass]int{
	RetentionAccess:      AccessRetentionFloorDays,
	RetentionOperational: OperationalRetentionFloorDays,
	RetentionFinancial:   FinancialRetentionFloorDays,
	RetentionSecurity:    SecurityRetentionFloorDays,
}

func (c RetentionClass) FloorDays() int {
	if days, ok := retentionFloors[c]; ok {
		return days
	}
	return MaxRetentionDays
}

type Channel string

const (
	ChannelWeb               = Channel(auditctx.ChannelWeb)
	ChannelAPI               = Channel(auditctx.ChannelAPI)
	ChannelPortal            = Channel(auditctx.ChannelPortal)
	ChannelAgentConversation = Channel(auditctx.ChannelAgentConversation)
	ChannelAgentRun          = Channel(auditctx.ChannelAgentRun)
	ChannelSupport           = Channel(auditctx.ChannelSupport)
	ChannelSCIM              = Channel(auditctx.ChannelSCIM)
	ChannelJob               = Channel(auditctx.ChannelJob)
	ChannelCLI               = Channel(auditctx.ChannelCLI)
	ChannelSystem            = Channel(auditctx.ChannelSystem)
)

func (c Channel) Valid() bool {
	switch c {
	case ChannelWeb, ChannelAPI, ChannelPortal, ChannelAgentConversation, ChannelAgentRun,
		ChannelSupport, ChannelSCIM, ChannelJob, ChannelCLI, ChannelSystem:
		return true
	}
	return false
}

var (
	actionPattern  = regexp.MustCompile(`^[a-z][a-z0-9_]*(\.[a-z][a-z0-9_]*)+$`)
	actionSegments = regexp.MustCompile(`[^a-z0-9_]+`)
)

func ValidAction(action string) bool {
	return actionPattern.MatchString(action)
}

func DefaultAction(resource, operation string) string {
	return actionSegment(resource) + "." + actionSegment(operation)
}

func actionSegment(value string) string {
	segment := actionSegments.ReplaceAllString(strings.ToLower(strings.TrimSpace(value)), "_")
	segment = strings.Trim(segment, "_")
	if segment == "" {
		return "unknown"
	}
	if segment[0] < 'a' || segment[0] > 'z' {
		segment = "x" + segment
	}
	return segment
}

const (
	ActionLegalHoldPlaced    = "audit_legal_hold.placed"
	ActionLegalHoldReleased  = "audit_legal_hold.released"
	ActionExportRequested    = "audit_log.export_requested"
	ActionExportDownloaded   = "audit_log.export_downloaded"
	ActionAlertPolicyChanged = "audit_alert_policy.updated"
	ActionAlertRaised        = "audit_alert.raised"
	ActionCrossTenantRefused = "organization.cross_tenant_refused"
	ActionMalwareRefused     = "document.malware_refused"
)
