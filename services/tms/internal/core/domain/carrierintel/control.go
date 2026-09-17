package carrierintel

import (
	"context"
	"strconv"

	"github.com/emoss08/trenova/internal/core/domain/integration"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
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

var _ bun.BeforeAppendModelHook = (*CarrierIntelControl)(nil)

type CarrierIntelControl struct {
	bun.BaseModel `bun:"table:carrier_intel_controls,alias:cictl" json:"-"`

	ID                      pulid.ID          `json:"id"                      bun:"id,type:VARCHAR(100),pk,notnull"`
	BusinessUnitID          pulid.ID          `json:"businessUnitId"          bun:"business_unit_id,type:VARCHAR(100),pk,notnull"`
	OrganizationID          pulid.ID          `json:"organizationId"          bun:"organization_id,type:VARCHAR(100),pk,notnull"`
	PrimaryProvider         *integration.Type `json:"primaryProvider"         bun:"primary_provider,type:integration_type,nullzero"`
	FallbackProvider        *integration.Type `json:"fallbackProvider"        bun:"fallback_provider,type:integration_type,nullzero"`
	EnrollmentPolicy        EnrollmentPolicy  `json:"enrollmentPolicy"        bun:"enrollment_policy,type:VARCHAR(20),notnull"`
	RecentUsageDays         int               `json:"recentUsageDays"         bun:"recent_usage_days,type:INTEGER,notnull"`
	IncludeOpenTenders      bool              `json:"includeOpenTenders"      bun:"include_open_tenders,type:BOOLEAN,notnull"`
	AutoEnrollOnCreate      bool              `json:"autoEnrollOnCreate"      bun:"auto_enroll_on_create,type:BOOLEAN,notnull"`
	AutoUnenrollOnInactive  bool              `json:"autoUnenrollOnInactive"  bun:"auto_unenroll_on_inactive,type:BOOLEAN,notnull"`
	ExclusiveWatchlist      bool              `json:"exclusiveWatchlist"      bun:"exclusive_watchlist,type:BOOLEAN,notnull"`
	PollIntervalMinutes     int               `json:"pollIntervalMinutes"     bun:"poll_interval_minutes,type:INTEGER,notnull"`
	SnapshotTTLHours        int               `json:"snapshotTtlHours"        bun:"snapshot_ttl_hours,type:INTEGER,notnull"`
	FullProfileTTLDays      int               `json:"fullProfileTtlDays"      bun:"full_profile_ttl_days,type:INTEGER,notnull"`
	PreTenderRefreshEnabled bool              `json:"preTenderRefreshEnabled" bun:"pretender_refresh_enabled,type:BOOLEAN,notnull"`
	PreTenderMaxAgeHours    int               `json:"preTenderMaxAgeHours"    bun:"pretender_max_age_hours,type:INTEGER,notnull"`
	HardMaxAgeHours         int               `json:"hardMaxAgeHours"         bun:"hard_max_age_hours,type:INTEGER,notnull"`
	ConfirmBlockingChanges  bool              `json:"confirmBlockingChanges"  bun:"confirm_blocking_changes,type:BOOLEAN,notnull"`
	OutagePolicy            OutagePolicy      `json:"outagePolicy"            bun:"outage_policy,type:VARCHAR(20),notnull"`
	AutoDisqualifyOnBlock   bool              `json:"autoDisqualifyOnBlock"   bun:"auto_disqualify_on_block,type:BOOLEAN,notnull"`
	AutoApplySafetyRating   bool              `json:"autoApplySafetyRating"   bun:"auto_apply_safety_rating,type:BOOLEAN,notnull"`
	Rules                   RuleSettings      `json:"rules"                   bun:"rules,type:JSONB,notnull"`
	AutoSyncFields          []SyncField       `json:"autoSyncFields"          bun:"auto_sync_fields,type:JSONB,notnull"`
	MonthlySpendCap         *decimal.Decimal  `json:"monthlySpendCap"         bun:"monthly_spend_cap,type:NUMERIC(19,4),nullzero"`
	SoftCapPercent          int               `json:"softCapPercent"          bun:"soft_cap_percent,type:INTEGER,notnull"`
	DailyFullProfileCap     *int              `json:"dailyFullProfileCap"     bun:"daily_full_profile_cap,type:INTEGER,nullzero"`
	RawRetentionDays        int               `json:"rawRetentionDays"        bun:"raw_retention_days,type:INTEGER,notnull"`
	SnapshotHistoryLimit    int               `json:"snapshotHistoryLimit"    bun:"snapshot_history_limit,type:INTEGER,notnull"`
	SelfMonitoringEnabled   bool              `json:"selfMonitoringEnabled"   bun:"self_monitoring_enabled,type:BOOLEAN,notnull"`
	PolicyVersion           int64             `json:"policyVersion"           bun:"policy_version,type:BIGINT,notnull"`
	Version                 int64             `json:"version"                 bun:"version,type:BIGINT"`
	CreatedAt               int64             `json:"createdAt"               bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt               int64             `json:"updatedAt"               bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
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

func (c *CarrierIntelControl) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if c.ID.IsNil() {
			c.ID = pulid.MustNew("cictl_")
		}
		if c.Rules == nil {
			c.Rules = RuleSettings{}
		}
		if c.AutoSyncFields == nil {
			c.AutoSyncFields = []SyncField{}
		}
		c.CreatedAt = now
		c.UpdatedAt = now
	case *bun.UpdateQuery:
		c.UpdatedAt = now
	}

	return nil
}
