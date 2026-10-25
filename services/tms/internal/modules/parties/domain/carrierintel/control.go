package carrierintel

import (
	"context"
	"strconv"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/integration"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/shopspring/decimal"
)

const (
	DefaultRecentUsageDays        = 90
	DefaultPollIntervalMinutes    = 120
	DefaultSnapshotTTLHours       = 24
	DefaultFullProfileTTLDays     = 30
	DefaultPreTenderMaxAgeHours   = 24
	DefaultHardMaxAgeHours        = 168
	DefaultSoftCapPercent         = 80
	DefaultRawRetentionDays       = 90
	DefaultSnapshotHistoryLimit   = 12
	MaxOverrideDurationSeconds    = int64(90 * 86400)
	DefaultOverrideDurationSecond = int64(14 * 86400)
)

type CarrierIntelControl struct {
	keel.Table `keel:"carrier_intel_controls"`

	ID                      pulid.ID          `json:"id"                      keel:"id,pk,autoid:cictl_"`
	BusinessUnitID          pulid.ID          `json:"businessUnitId"          keel:"business_unit_id,pk,tenant"`
	OrganizationID          pulid.ID          `json:"organizationId"          keel:"organization_id,pk,tenant"`
	PrimaryProvider         *integration.Type `json:"primaryProvider"         keel:"primary_provider"`
	FallbackProvider        *integration.Type `json:"fallbackProvider"        keel:"fallback_provider"`
	EnrollmentPolicy        EnrollmentPolicy  `json:"enrollmentPolicy"        keel:"enrollment_policy"`
	RecentUsageDays         int               `json:"recentUsageDays"         keel:"recent_usage_days"`
	IncludeOpenTenders      bool              `json:"includeOpenTenders"      keel:"include_open_tenders"`
	AutoEnrollOnCreate      bool              `json:"autoEnrollOnCreate"      keel:"auto_enroll_on_create"`
	AutoUnenrollOnInactive  bool              `json:"autoUnenrollOnInactive"  keel:"auto_unenroll_on_inactive"`
	ExclusiveWatchlist      bool              `json:"exclusiveWatchlist"      keel:"exclusive_watchlist"`
	PollIntervalMinutes     int               `json:"pollIntervalMinutes"     keel:"poll_interval_minutes"`
	SnapshotTTLHours        int               `json:"snapshotTtlHours"        keel:"snapshot_ttl_hours"`
	FullProfileTTLDays      int               `json:"fullProfileTtlDays"      keel:"full_profile_ttl_days"`
	PreTenderRefreshEnabled bool              `json:"preTenderRefreshEnabled" keel:"pretender_refresh_enabled"`
	PreTenderMaxAgeHours    int               `json:"preTenderMaxAgeHours"    keel:"pretender_max_age_hours"`
	HardMaxAgeHours         int               `json:"hardMaxAgeHours"         keel:"hard_max_age_hours"`
	ConfirmBlockingChanges  bool              `json:"confirmBlockingChanges"  keel:"confirm_blocking_changes"`
	OutagePolicy            OutagePolicy      `json:"outagePolicy"            keel:"outage_policy"`
	AutoDisqualifyOnBlock   bool              `json:"autoDisqualifyOnBlock"   keel:"auto_disqualify_on_block"`
	AutoApplySafetyRating   bool              `json:"autoApplySafetyRating"   keel:"auto_apply_safety_rating"`
	Rules                   RuleSettings      `json:"rules"                   keel:"rules,jsonb"`
	AutoSyncFields          []SyncField       `json:"autoSyncFields"          keel:"auto_sync_fields,jsonb"`
	MonthlySpendCap         *decimal.Decimal  `json:"monthlySpendCap"         keel:"monthly_spend_cap"`
	SoftCapPercent          int               `json:"softCapPercent"          keel:"soft_cap_percent"`
	DailyFullProfileCap     *int              `json:"dailyFullProfileCap"     keel:"daily_full_profile_cap"`
	RawRetentionDays        int               `json:"rawRetentionDays"        keel:"raw_retention_days"`
	SnapshotHistoryLimit    int               `json:"snapshotHistoryLimit"    keel:"snapshot_history_limit"`
	SelfMonitoringEnabled   bool              `json:"selfMonitoringEnabled"   keel:"self_monitoring_enabled"`
	PolicyVersion           int64             `json:"policyVersion"           keel:"policy_version"`
	Version                 int64             `json:"version"                 keel:"version,version"`
	CreatedAt               int64             `json:"createdAt"               keel:"created_at,autocreate"`
	UpdatedAt               int64             `json:"updatedAt"               keel:"updated_at,autoupdate"`
}

func (c *CarrierIntelControl) BeforeCreate(_ context.Context) error {
	if c.Rules == nil {
		c.Rules = RuleSettings{}
	}
	if c.AutoSyncFields == nil {
		c.AutoSyncFields = []SyncField{}
	}

	return nil
}

func NewDefaultControl(orgID, buID pulid.ID) *CarrierIntelControl {
	return &CarrierIntelControl{
		OrganizationID:         orgID,
		BusinessUnitID:         buID,
		EnrollmentPolicy:       EnrollmentPolicyManual,
		RecentUsageDays:        DefaultRecentUsageDays,
		AutoEnrollOnCreate:     true,
		AutoUnenrollOnInactive: true,
		PollIntervalMinutes:    DefaultPollIntervalMinutes,
		SnapshotTTLHours:       DefaultSnapshotTTLHours,
		FullProfileTTLDays:     DefaultFullProfileTTLDays,
		PreTenderMaxAgeHours:   DefaultPreTenderMaxAgeHours,
		HardMaxAgeHours:        DefaultHardMaxAgeHours,
		ConfirmBlockingChanges: true,
		OutagePolicy:           OutagePolicyFailOpen,
		AutoApplySafetyRating:  true,
		Rules:                  RuleSettings{},
		AutoSyncFields:         []SyncField{},
		SoftCapPercent:         DefaultSoftCapPercent,
		RawRetentionDays:       DefaultRawRetentionDays,
		SnapshotHistoryLimit:   DefaultSnapshotHistoryLimit,
		SelfMonitoringEnabled:  true,
		PolicyVersion:          1,
	}
}

func (c *CarrierIntelControl) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(c,
		validation.Field(&c.EnrollmentPolicy,
			validation.Required.Error("Enrollment policy is required"),
			domainvalidation.ValidEnum[EnrollmentPolicy]("Enrollment policy is invalid"),
		),
		validation.Field(&c.OutagePolicy,
			validation.Required.Error("Outage policy is required"),
			domainvalidation.ValidEnum[OutagePolicy]("Outage policy is invalid"),
		),
		validation.Field(&c.RecentUsageDays,
			validation.Min(7).Error("Recent usage window must be at least 7 days"),
			validation.Max(365).Error("Recent usage window cannot exceed 365 days"),
		),
		validation.Field(&c.PollIntervalMinutes,
			validation.Min(60).Error("Poll interval must be at least 60 minutes"),
			validation.Max(1440).Error("Poll interval cannot exceed 1440 minutes"),
		),
		validation.Field(&c.SnapshotTTLHours,
			validation.Min(1).Error("Snapshot freshness must be at least 1 hour"),
			validation.Max(720).Error("Snapshot freshness cannot exceed 720 hours"),
		),
		validation.Field(&c.FullProfileTTLDays,
			validation.Min(1).Error("Full profile freshness must be at least 1 day"),
			validation.Max(365).Error("Full profile freshness cannot exceed 365 days"),
		),
		validation.Field(&c.PreTenderMaxAgeHours,
			validation.Min(1).Error("Pre-tender freshness must be at least 1 hour"),
			validation.Max(720).Error("Pre-tender freshness cannot exceed 720 hours"),
		),
		validation.Field(&c.HardMaxAgeHours,
			validation.Min(1).Error("Maximum intelligence age must be at least 1 hour"),
			validation.Max(8760).Error("Maximum intelligence age cannot exceed 8760 hours"),
		),
		validation.Field(&c.SoftCapPercent,
			validation.Min(1).Error("Soft cap percent must be at least 1"),
			validation.Max(100).Error("Soft cap percent cannot exceed 100"),
		),
		validation.Field(&c.RawRetentionDays,
			validation.Min(7).Error("Raw payload retention must be at least 7 days"),
			validation.Max(730).Error("Raw payload retention cannot exceed 730 days"),
		),
		validation.Field(&c.SnapshotHistoryLimit,
			validation.Min(1).Error("Snapshot history must keep at least 1 snapshot"),
			validation.Max(100).Error("Snapshot history cannot keep more than 100 snapshots"),
		),
	))

	if c.HardMaxAgeHours < c.PreTenderMaxAgeHours {
		multiErr.Add("hardMaxAgeHours", errortypes.ErrInvalid,
			"Maximum intelligence age cannot be shorter than the pre-tender freshness window")
	}
	if c.MonthlySpendCap != nil && c.MonthlySpendCap.IsNegative() {
		multiErr.Add(
			"monthlySpendCap",
			errortypes.ErrInvalid,
			"Monthly spend cap cannot be negative",
		)
	}
	if c.DailyFullProfileCap != nil && *c.DailyFullProfileCap < 0 {
		multiErr.Add("dailyFullProfileCap", errortypes.ErrInvalid,
			"Daily full profile cap cannot be negative")
	}
	if c.FallbackProvider != nil && *c.FallbackProvider != integration.TypeFMCSAQCMobile {
		multiErr.Add("fallbackProvider", errortypes.ErrInvalid,
			"Only FMCSA QCMobile can be used as a fallback provider")
	}
	if c.PrimaryProvider != nil && c.FallbackProvider != nil &&
		*c.PrimaryProvider == *c.FallbackProvider {
		multiErr.Add("fallbackProvider", errortypes.ErrInvalid,
			"The fallback provider must differ from the primary provider")
	}

	for idx, field := range c.AutoSyncFields {
		if !field.IsValid() {
			multiErr.WithIndex("autoSyncFields", idx).Add("", errortypes.ErrInvalid,
				"Automatic sync field is invalid")
		}
	}

	ValidateRuleSettings(c.Rules, multiErr.WithPrefix("rules"))
}

func ValidateRuleSettings(settings RuleSettings, multiErr *errortypes.MultiError) {
	for code, setting := range settings {
		def, ok := RuleByCode(code)
		if !ok {
			multiErr.Add(code.String(), errortypes.ErrInvalid, "Unknown rule")
			continue
		}
		if !setting.Action.IsValid() {
			multiErr.Add(code.String()+".action", errortypes.ErrInvalid, "Rule action is invalid")
		}
		for key, raw := range setting.Params {
			spec := def.paramSpec(key)
			if spec == nil {
				multiErr.Add(code.String()+".params."+key, errortypes.ErrInvalid,
					"Unknown rule parameter")
				continue
			}
			if msg := validateParam(spec, raw); msg != "" {
				multiErr.Add(code.String()+".params."+key, errortypes.ErrInvalid, msg)
			}
		}
	}
}

func validateParam(spec *RuleParamSpec, raw string) string {
	if raw == "" {
		return ""
	}
	var value float64
	switch spec.Type {
	case RuleParamTypeInteger:
		v, err := strconv.Atoi(raw)
		if err != nil {
			return spec.Label + " must be a whole number"
		}
		value = float64(v)
	case RuleParamTypeDecimal:
		v, err := decimal.NewFromString(raw)
		if err != nil {
			return spec.Label + " must be a number"
		}
		value = v.InexactFloat64()
	case RuleParamTypeNumber:
		v, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			return spec.Label + " must be a number"
		}
		value = v
	case RuleParamTypeSelect:
		for _, option := range spec.Options {
			if option == raw {
				return ""
			}
		}
		return spec.Label + " has an invalid option"
	case RuleParamTypeMultiSelect:
		ctx := &RuleContext{params: map[string]string{spec.Key: raw}, spec: &RuleDefinition{}}
		for _, item := range ctx.paramList(spec.Key) {
			found := false
			for _, option := range spec.Options {
				if option == item {
					found = true
					break
				}
			}
			if !found {
				return spec.Label + " has an invalid option"
			}
		}
		return ""
	default:
		return ""
	}
	if spec.Min != nil && value < *spec.Min {
		return spec.Label + " is below the minimum"
	}
	if spec.Max != nil && value > *spec.Max {
		return spec.Label + " is above the maximum"
	}
	return ""
}

func (c *CarrierIntelControl) SyncSettings() SyncSettings {
	return SyncSettings{
		AutoApplySafetyRating: c.AutoApplySafetyRating,
		AutoFillFields:        c.AutoSyncFields,
	}
}

func (c *CarrierIntelControl) PrimaryType() (integration.Type, bool) {
	if c == nil || c.PrimaryProvider == nil || *c.PrimaryProvider == "" {
		return "", false
	}
	return *c.PrimaryProvider, true
}

func (c *CarrierIntelControl) FallbackType() (integration.Type, bool) {
	if c == nil || c.FallbackProvider == nil || *c.FallbackProvider == "" {
		return "", false
	}
	return *c.FallbackProvider, true
}

func (c *CarrierIntelControl) SnapshotTTLSeconds() int64 {
	return int64(c.SnapshotTTLHours) * 3600
}

func (c *CarrierIntelControl) FullProfileTTLSeconds() int64 {
	return int64(c.FullProfileTTLDays) * 86400
}

func (c *CarrierIntelControl) TTLSecondsForDepth(depth LookupDepth) int64 {
	if depth == LookupDepthFull {
		return c.FullProfileTTLSeconds()
	}
	return c.SnapshotTTLSeconds()
}

func (c *CarrierIntelControl) PreTenderMaxAgeSeconds() int64 {
	return int64(c.PreTenderMaxAgeHours) * 3600
}

func (c *CarrierIntelControl) HardMaxAgeSeconds() int64 {
	return int64(c.HardMaxAgeHours) * 3600
}

func (c *CarrierIntelControl) GetID() pulid.ID { return c.ID }

func (c *CarrierIntelControl) GetOrganizationID() pulid.ID { return c.OrganizationID }

func (c *CarrierIntelControl) GetBusinessUnitID() pulid.ID { return c.BusinessUnitID }

func (c *CarrierIntelControl) GetTableName() string { return "carrier_intel_controls" }
