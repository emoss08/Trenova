package resolver

import (
	"context"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/99designs/gqlgen/graphql"
	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/loaders"
	"github.com/emoss08/trenova/internal/core/domain/carrierintel"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/carrierintelservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

func requiredOmittable[T any](
	field, label string,
	value graphql.Omittable[*T],
) (carrierintelservice.Optional[T], error) {
	if !value.IsSet() {
		return carrierintelservice.Optional[T]{}, nil
	}
	ptr := value.Value()
	if ptr == nil {
		return carrierintelservice.Optional[T]{}, errortypes.NewValidationError(
			field, errortypes.ErrRequired, "{0} cannot be cleared", label,
		)
	}
	return carrierintelservice.Some(*ptr), nil
}

func nullableIntOmittable(value graphql.Omittable[*int]) carrierintelservice.Optional[*int] {
	if !value.IsSet() {
		return carrierintelservice.Optional[*int]{}
	}
	return carrierintelservice.Some(value.Value())
}

func controlPatchFromInput(
	input *gqlmodel.CarrierIntelControlPatchInput,
) (*carrierintelservice.ControlPatch, error) {
	patch := &carrierintelservice.ControlPatch{
		ConfirmEstimatedCost: boolValue(input.ConfirmEstimatedCost),
		DailyFullProfileCap:  nullableIntOmittable(input.DailyFullProfileCap),
	}
	if input.Version != nil {
		version := int64(*input.Version)
		patch.ExpectedVersion = &version
	}

	var err error
	multiErr := errortypes.NewMultiError()
	collect := func(e error) {
		if e == nil {
			return
		}
		if single, ok := e.(*errortypes.Error); ok {
			multiErr.AddError(single)
			return
		}
		err = e
	}

	var e error
	patch.EnrollmentPolicy, e = requiredOmittable(
		"enrollmentPolicy",
		"Enrollment policy",
		input.EnrollmentPolicy,
	)
	collect(e)
	patch.RecentUsageDays, e = requiredOmittable(
		"recentUsageDays",
		"Recent usage window",
		input.RecentUsageDays,
	)
	collect(e)
	patch.IncludeOpenTenders, e = requiredOmittable(
		"includeOpenTenders",
		"Include open tenders",
		input.IncludeOpenTenders,
	)
	collect(e)
	patch.AutoEnrollOnCreate, e = requiredOmittable(
		"autoEnrollOnCreate",
		"Automatic enrollment",
		input.AutoEnrollOnCreate,
	)
	collect(e)
	patch.AutoUnenrollOnInactive, e = requiredOmittable(
		"autoUnenrollOnInactive", "Automatic unenrollment", input.AutoUnenrollOnInactive,
	)
	collect(e)
	patch.ExclusiveWatchlist, e = requiredOmittable(
		"exclusiveWatchlist",
		"Exclusive watchlist",
		input.ExclusiveWatchlist,
	)
	collect(e)
	patch.PollIntervalMinutes, e = requiredOmittable(
		"pollIntervalMinutes",
		"Poll interval",
		input.PollIntervalMinutes,
	)
	collect(e)
	patch.SnapshotTTLHours, e = requiredOmittable(
		"snapshotTtlHours",
		"Snapshot freshness",
		input.SnapshotTTLHours,
	)
	collect(e)
	patch.FullProfileTTLDays, e = requiredOmittable(
		"fullProfileTtlDays",
		"Full profile freshness",
		input.FullProfileTTLDays,
	)
	collect(e)
	patch.PreTenderRefreshEnabled, e = requiredOmittable(
		"preTenderRefreshEnabled", "Pre-tender refresh", input.PreTenderRefreshEnabled,
	)
	collect(e)
	patch.PreTenderMaxAgeHours, e = requiredOmittable(
		"preTenderMaxAgeHours", "Pre-tender freshness", input.PreTenderMaxAgeHours,
	)
	collect(e)
	patch.HardMaxAgeHours, e = requiredOmittable(
		"hardMaxAgeHours",
		"Maximum intelligence age",
		input.HardMaxAgeHours,
	)
	collect(e)
	patch.ConfirmBlockingChanges, e = requiredOmittable(
		"confirmBlockingChanges", "Confirm blocking changes", input.ConfirmBlockingChanges,
	)
	collect(e)
	patch.OutagePolicy, e = requiredOmittable("outagePolicy", "Outage policy", input.OutagePolicy)
	collect(e)
	patch.AutoDisqualifyOnBlock, e = requiredOmittable(
		"autoDisqualifyOnBlock", "Automatic disqualification", input.AutoDisqualifyOnBlock,
	)
	collect(e)
	patch.AutoApplySafetyRating, e = requiredOmittable(
		"autoApplySafetyRating", "Automatic safety rating sync", input.AutoApplySafetyRating,
	)
	collect(e)
	patch.SoftCapPercent, e = requiredOmittable(
		"softCapPercent",
		"Soft cap percent",
		input.SoftCapPercent,
	)
	collect(e)
	patch.RawRetentionDays, e = requiredOmittable(
		"rawRetentionDays",
		"Raw payload retention",
		input.RawRetentionDays,
	)
	collect(e)
	patch.SnapshotHistoryLimit, e = requiredOmittable(
		"snapshotHistoryLimit", "Snapshot history", input.SnapshotHistoryLimit,
	)
	collect(e)
	patch.SelfMonitoringEnabled, e = requiredOmittable(
		"selfMonitoringEnabled", "Self-monitoring", input.SelfMonitoringEnabled,
	)
	collect(e)

	if input.AutoSyncFields.IsSet() {
		patch.AutoSyncFields = carrierintelservice.Some(input.AutoSyncFields.Value())
	}

	if input.MonthlySpendCap.IsSet() {
		raw := input.MonthlySpendCap.Value()
		if raw == nil || strings.TrimSpace(*raw) == "" {
			patch.MonthlySpendCap = carrierintelservice.Some[*decimal.Decimal](nil)
		} else {
			amount, parseErr := decimalFromString(*raw, "monthlySpendCap")
			collect(parseErr)
			if parseErr == nil {
				patch.MonthlySpendCap = carrierintelservice.Some(&amount)
			}
		}
	}

	if input.Rules.IsSet() {
		rules := make(carrierintel.RuleSettings, len(input.Rules.Value()))
		for _, rule := range input.Rules.Value() {
			if rule == nil {
				continue
			}
			params := make(map[string]string, len(rule.Params))
			for _, param := range rule.Params {
				if param == nil {
					continue
				}
				params[param.Key] = param.Value
			}
			rules[carrierintel.RuleCode(rule.Code)] = carrierintel.RuleSetting{
				Action: rule.Action,
				Params: params,
			}
		}
		patch.Rules = carrierintelservice.Some(rules)
	}

	if err != nil {
		return nil, err
	}
	if multiErr.HasErrors() {
		return nil, multiErr
	}
	return patch, nil
}

func ruleSettingsToModel(settings carrierintel.RuleSettings) []*gqlmodel.CarrierIntelRuleSetting {
	codes := make([]string, 0, len(settings))
	for code := range settings {
		codes = append(codes, code.String())
	}
	sort.Strings(codes)

	out := make([]*gqlmodel.CarrierIntelRuleSetting, 0, len(codes))
	for _, code := range codes {
		setting := settings[carrierintel.RuleCode(code)]
		keys := make([]string, 0, len(setting.Params))
		for key := range setting.Params {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		params := make([]*gqlmodel.CarrierIntelRuleSettingParam, 0, len(keys))
		for _, key := range keys {
			params = append(params, &gqlmodel.CarrierIntelRuleSettingParam{
				Key:   key,
				Value: setting.Params[key],
			})
		}
		out = append(out, &gqlmodel.CarrierIntelRuleSetting{
			Code:   code,
			Action: setting.Action,
			Params: params,
		})
	}
	return out
}

func fetchResultToModel(result *carrierintelservice.FetchResult) *gqlmodel.CarrierIntelFetchResult {
	return &gqlmodel.CarrierIntelFetchResult{
		Snapshot:     result.Snapshot,
		FromCache:    result.FromCache,
		UsedFallback: result.UsedFallback,
		RaisedCount:  len(result.Raised),
		ChangeCount:  len(result.Changes),
	}
}

func jsonValueString(value any) *string {
	if value == nil {
		return nil
	}
	switch typed := value.(type) {
	case string:
		return &typed
	default:
		raw, err := sonic.Marshal(typed)
		if err != nil {
			return nil
		}
		text := string(raw)
		return &text
	}
}

func providerInfoToModel(
	provider string,
	fallback *string,
	descriptor carrierintel.ProviderDescriptor,
	configured bool,
) *gqlmodel.CarrierIntelProviderInfo {
	info := &gqlmodel.CarrierIntelProviderInfo{
		Configured:       configured,
		FallbackProvider: fallback,
		Capabilities:     descriptor.Capabilities.Names(),
		Sections:         slices.Clone(descriptor.Sections),
	}
	if info.Sections == nil {
		info.Sections = []carrierintel.Section{}
	}
	if provider != "" {
		info.Provider = &provider
	}
	return info
}

func eventCountsToModel(
	counts *repositories.CarrierIntelEventCounts,
) *gqlmodel.CarrierIntelEventCounts {
	result := &gqlmodel.CarrierIntelEventCounts{
		Open:         counts.Open,
		Acknowledged: counts.Acknowledged,
		BySeverity:   make([]*gqlmodel.CarrierIntelSeverityCount, 0, len(counts.BySeverity)),
	}
	for _, severity := range []carrierintel.Severity{
		carrierintel.SeverityCritical,
		carrierintel.SeverityHigh,
		carrierintel.SeverityMedium,
		carrierintel.SeverityLow,
		carrierintel.SeverityInfo,
	} {
		result.BySeverity = append(result.BySeverity, &gqlmodel.CarrierIntelSeverityCount{
			Severity: severity,
			Count:    counts.BySeverity[severity],
		})
	}
	return result
}

func usageSummaryToModel(
	summary *carrierintelservice.UsageSummary,
) *gqlmodel.CarrierIntelUsageSummary {
	rows := make([]*gqlmodel.CarrierIntelUsageRow, 0, len(summary.ByEndpoint))
	for _, row := range summary.ByEndpoint {
		rows = append(rows, &gqlmodel.CarrierIntelUsageRow{
			Provider:      row.Provider.String(),
			Endpoint:      row.Endpoint.String(),
			Calls:         row.Calls,
			BillableUnits: row.BillableUnits,
			EstimatedCost: row.EstimatedCost.StringFixed(2),
		})
	}
	daily := make([]*gqlmodel.CarrierIntelUsageDay, 0, len(summary.Daily))
	for _, day := range summary.Daily {
		daily = append(daily, &gqlmodel.CarrierIntelUsageDay{
			Day:           day.Day,
			Endpoint:      day.Endpoint.String(),
			Calls:         day.Calls,
			BillableUnits: day.BillableUnits,
			EstimatedCost: day.EstimatedCost.StringFixed(2),
		})
	}
	var capValue *string
	if summary.Cap != nil {
		formatted := summary.Cap.StringFixed(2)
		capValue = &formatted
	}
	return &gqlmodel.CarrierIntelUsageSummary{
		MonthStart:     int(summary.MonthStart),
		MonthToDate:    summary.MonthToDate.StringFixed(2),
		Cap:            capValue,
		SoftCapPercent: summary.SoftCapPercent,
		ByEndpoint:     rows,
		Daily:          daily,
	}
}

func monthFromTimestamp(value *int) time.Time {
	if value == nil || *value <= 0 {
		return time.Time{}
	}
	return time.Unix(int64(*value), 0).UTC()
}

func carrierIntelEventConnectionToModel(
	result *pagination.CursorListResult[*carrierintel.CarrierIntelEvent],
) (*gqlmodel.CarrierIntelEventConnection, error) {
	page, err := entityCursorConnection(
		result,
		func(node *carrierintel.CarrierIntelEvent, cursor string) *gqlmodel.CarrierIntelEventEdge {
			return &gqlmodel.CarrierIntelEventEdge{Node: node, Cursor: cursor}
		},
		func(edge *gqlmodel.CarrierIntelEventEdge) string { return edge.Cursor },
	)
	if err != nil {
		return nil, err
	}
	return &gqlmodel.CarrierIntelEventConnection{
		Edges:      page.Edges,
		PageInfo:   page.PageInfo,
		TotalCount: page.TotalCount,
	}, nil
}

func carrierMonitoringEnrollmentConnectionToModel(
	result *pagination.CursorListResult[*carrierintel.CarrierMonitoringEnrollment],
) (*gqlmodel.CarrierMonitoringEnrollmentConnection, error) {
	page, err := entityCursorConnection(
		result,
		func(node *carrierintel.CarrierMonitoringEnrollment, cursor string) *gqlmodel.CarrierMonitoringEnrollmentEdge {
			return &gqlmodel.CarrierMonitoringEnrollmentEdge{Node: node, Cursor: cursor}
		},
		func(edge *gqlmodel.CarrierMonitoringEnrollmentEdge) string { return edge.Cursor },
	)
	if err != nil {
		return nil, err
	}
	return &gqlmodel.CarrierMonitoringEnrollmentConnection{
		Edges:      page.Edges,
		PageInfo:   page.PageInfo,
		TotalCount: page.TotalCount,
	}, nil
}

func eventFilterFromInput(
	filter *gqlmodel.CarrierIntelEventFilterInput,
	req *repositories.ListCarrierIntelEventsRequest,
) error {
	if filter == nil {
		return nil
	}
	req.Statuses = filter.Statuses
	req.Severities = filter.Severities
	req.Categories = filter.Categories
	req.OpenOnly = boolValue(filter.OpenOnly)
	if filter.SubjectType != nil {
		req.SubjectType = *filter.SubjectType
	}
	req.SubjectID = stringValue(filter.SubjectID)
	carrierID, err := optionalID(filter.CarrierID)
	if err != nil {
		return errortypes.NewValidationError(
			"carrierId",
			errortypes.ErrInvalid,
			"Carrier is invalid",
		)
	}
	req.CarrierID = carrierID
	return nil
}

func parseCarrierIntelID(raw, field, label string) (pulid.ID, error) {
	id, err := pulid.MustParse(raw)
	if err != nil {
		return pulid.Nil, errortypes.NewValidationError(
			field,
			errortypes.ErrInvalid,
			"{0} is invalid",
			label,
		)
	}
	return id, nil
}

func loadCarrierSnapshot(
	ctx context.Context,
	fetch func() ([]*carrierintel.CarrierIntelSnapshot, error),
	loader func(l *loaders.Loaders) ([]*carrierintel.CarrierIntelSnapshot, error),
) (*carrierintel.CarrierIntelSnapshot, error) {
	var snapshots []*carrierintel.CarrierIntelSnapshot
	var err error
	if l, ok := loaders.FromContext(ctx); ok && l != nil {
		snapshots, err = loader(l)
	} else {
		snapshots, err = fetch()
	}
	if err != nil || len(snapshots) == 0 {
		return nil, err
	}
	return snapshots[0], nil
}

func preferredEnrollment(
	rows []*carrierintel.CarrierMonitoringEnrollment,
) *carrierintel.CarrierMonitoringEnrollment {
	var best *carrierintel.CarrierMonitoringEnrollment
	for _, row := range rows {
		if row == nil {
			continue
		}
		if best == nil || (row.WantsEnrolled() && !best.WantsEnrolled()) ||
			(row.WantsEnrolled() == best.WantsEnrolled() && row.UpdatedAt > best.UpdatedAt) {
			best = row
		}
	}
	return best
}

func sourcingQueryFromInput(
	input *gqlmodel.CarrierSourcingSearchInput,
	tenant pagination.TenantInfo,
) *carrierintelservice.SourcingQuery {
	return &carrierintelservice.SourcingQuery{
		TenantInfo:             tenant,
		Text:                   stringValue(input.Text),
		State:                  strings.ToUpper(stringValue(input.State)),
		OriginState:            strings.ToUpper(stringValue(input.OriginState)),
		DestinationState:       strings.ToUpper(stringValue(input.DestinationState)),
		MinPowerUnits:          input.MinPowerUnits,
		MaxPowerUnits:          input.MaxPowerUnits,
		MinAuthorityAgeDays:    input.MinAuthorityAgeDays,
		MaxAuthorityAgeDays:    input.MaxAuthorityAgeDays,
		HazmatOnly:             boolValue(input.HazmatOnly),
		ExcludeBlocking:        boolValue(input.ExcludeBlocking),
		ExcludeExistingCarrier: boolValue(input.ExcludeExistingCarriers),
		Sort:                   sourcingSortValue(input.Sort),
		Limit:                  intValue(input.Limit),
		Offset:                 intValue(input.Offset),
	}
}

const carrierIntelReviewQueueCountLimit = 500

func (r *Resolver) carrierIntelNow() int64 {
	return time.Now().Unix()
}

func (r *Resolver) carrierIntelProviderInfo(
	ctx context.Context,
	tenant pagination.TenantInfo,
) (*gqlmodel.CarrierIntelProviderInfo, error) {
	control, err := r.carrierIntelService.Control(ctx, tenant)
	if err != nil {
		return nil, err
	}
	var fallback *string
	if provider, ok := control.FallbackType(); ok {
		value := provider.String()
		fallback = &value
	}
	provider, descriptor, configured, err := r.carrierIntelService.ProviderDescriptor(ctx, tenant)
	if err != nil {
		return nil, err
	}
	return providerInfoToModel(provider.String(), fallback, descriptor, configured), nil
}

func sourcingSortValue(sort *carrierintelservice.SourcingSort) carrierintelservice.SourcingSort {
	if sort == nil {
		return carrierintelservice.SourcingSortBestMatch
	}
	return *sort
}

func carrierIntelEventFieldLabel(event *carrierintel.CarrierIntelEvent) *string {
	if event.FieldPath == "" {
		return nil
	}
	label, _ := carrierintel.DescribeField(event.FieldPath)
	return &label
}

func carrierIntelEventRuleLabel(event *carrierintel.CarrierIntelEvent) *string {
	if event.RuleCode == "" {
		return nil
	}
	def, ok := carrierintel.RuleByCode(event.RuleCode)
	if !ok {
		return nil
	}
	label := def.Label
	return &label
}
