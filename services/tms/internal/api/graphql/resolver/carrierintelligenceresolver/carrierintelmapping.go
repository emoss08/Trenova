package carrierintelligenceresolver

import (
	"context"
	"sort"
	"strings"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/loaders"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlconnection"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqldecimal"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlid"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlscalar"
	"github.com/emoss08/trenova/internal/modules/parties/app/carrierintelservice"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrierintel"
	"github.com/emoss08/trenova/internal/modules/parties/ports/carrierintelports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

func controlPatchFromInput(
	input *gqlmodel.CarrierIntelControlPatchInput,
) (*carrierintelservice.ControlPatch, error) {
	patch := &carrierintelservice.ControlPatch{
		ConfirmEstimatedCost: gqlscalar.BoolValue(input.ConfirmEstimatedCost),
		DailyFullProfileCap:  gqlscalar.NullableOmittable(input.DailyFullProfileCap),
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
	patch.EnrollmentPolicy, e = gqlscalar.RequiredOmittable(
		"enrollmentPolicy",
		"Enrollment policy",
		input.EnrollmentPolicy,
	)
	collect(e)
	patch.RecentUsageDays, e = gqlscalar.RequiredOmittable(
		"recentUsageDays",
		"Recent usage window",
		input.RecentUsageDays,
	)
	collect(e)
	patch.IncludeOpenTenders, e = gqlscalar.RequiredOmittable(
		"includeOpenTenders",
		"Include open tenders",
		input.IncludeOpenTenders,
	)
	collect(e)
	patch.AutoEnrollOnCreate, e = gqlscalar.RequiredOmittable(
		"autoEnrollOnCreate",
		"Automatic enrollment",
		input.AutoEnrollOnCreate,
	)
	collect(e)
	patch.AutoUnenrollOnInactive, e = gqlscalar.RequiredOmittable(
		"autoUnenrollOnInactive", "Automatic unenrollment", input.AutoUnenrollOnInactive,
	)
	collect(e)
	patch.ExclusiveWatchlist, e = gqlscalar.RequiredOmittable(
		"exclusiveWatchlist",
		"Exclusive watchlist",
		input.ExclusiveWatchlist,
	)
	collect(e)
	patch.PollIntervalMinutes, e = gqlscalar.RequiredOmittable(
		"pollIntervalMinutes",
		"Poll interval",
		input.PollIntervalMinutes,
	)
	collect(e)
	patch.SnapshotTTLHours, e = gqlscalar.RequiredOmittable(
		"snapshotTtlHours",
		"Snapshot freshness",
		input.SnapshotTTLHours,
	)
	collect(e)
	patch.FullProfileTTLDays, e = gqlscalar.RequiredOmittable(
		"fullProfileTtlDays",
		"Full profile freshness",
		input.FullProfileTTLDays,
	)
	collect(e)
	patch.PreTenderRefreshEnabled, e = gqlscalar.RequiredOmittable(
		"preTenderRefreshEnabled", "Pre-tender refresh", input.PreTenderRefreshEnabled,
	)
	collect(e)
	patch.PreTenderMaxAgeHours, e = gqlscalar.RequiredOmittable(
		"preTenderMaxAgeHours", "Pre-tender freshness", input.PreTenderMaxAgeHours,
	)
	collect(e)
	patch.HardMaxAgeHours, e = gqlscalar.RequiredOmittable(
		"hardMaxAgeHours",
		"Maximum intelligence age",
		input.HardMaxAgeHours,
	)
	collect(e)
	patch.ConfirmBlockingChanges, e = gqlscalar.RequiredOmittable(
		"confirmBlockingChanges", "Confirm blocking changes", input.ConfirmBlockingChanges,
	)
	collect(e)
	patch.OutagePolicy, e = gqlscalar.RequiredOmittable(
		"outagePolicy",
		"Outage policy",
		input.OutagePolicy,
	)
	collect(e)
	patch.AutoDisqualifyOnBlock, e = gqlscalar.RequiredOmittable(
		"autoDisqualifyOnBlock", "Automatic disqualification", input.AutoDisqualifyOnBlock,
	)
	collect(e)
	patch.AutoApplySafetyRating, e = gqlscalar.RequiredOmittable(
		"autoApplySafetyRating", "Automatic safety rating sync", input.AutoApplySafetyRating,
	)
	collect(e)
	patch.SoftCapPercent, e = gqlscalar.RequiredOmittable(
		"softCapPercent",
		"Soft cap percent",
		input.SoftCapPercent,
	)
	collect(e)
	patch.RawRetentionDays, e = gqlscalar.RequiredOmittable(
		"rawRetentionDays",
		"Raw payload retention",
		input.RawRetentionDays,
	)
	collect(e)
	patch.SnapshotHistoryLimit, e = gqlscalar.RequiredOmittable(
		"snapshotHistoryLimit", "Snapshot history", input.SnapshotHistoryLimit,
	)
	collect(e)
	patch.SelfMonitoringEnabled, e = gqlscalar.RequiredOmittable(
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
			amount, parseErr := gqldecimal.DecimalFromString(*raw, "monthlySpendCap")
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

func eventCountsToModel(
	counts *carrierintelports.CarrierIntelEventCounts,
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
	page, err := gqlconnection.EntityCursorConnection(
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
		Edges:           page.Edges,
		PageInfo:        page.PageInfo,
		TotalCount:      page.TotalCount,
		TotalCountExact: page.TotalCountExact,
	}, nil
}

func carrierMonitoringEnrollmentConnectionToModel(
	result *pagination.CursorListResult[*carrierintel.CarrierMonitoringEnrollment],
) (*gqlmodel.CarrierMonitoringEnrollmentConnection, error) {
	page, err := gqlconnection.EntityCursorConnection(
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
		Edges:           page.Edges,
		PageInfo:        page.PageInfo,
		TotalCount:      page.TotalCount,
		TotalCountExact: page.TotalCountExact,
	}, nil
}

func eventFilterFromInput(
	filter *gqlmodel.CarrierIntelEventFilterInput,
	req *carrierintelports.ListCarrierIntelEventsRequest,
) error {
	if filter == nil {
		return nil
	}
	req.Statuses = filter.Statuses
	req.Severities = filter.Severities
	req.Categories = filter.Categories
	req.OpenOnly = gqlscalar.BoolValue(filter.OpenOnly)
	if filter.SubjectType != nil {
		req.SubjectType = *filter.SubjectType
	}
	req.SubjectID = gqlscalar.StringValue(filter.SubjectID)
	carrierID, err := gqlid.OptionalID(filter.CarrierID)
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
	loader func() ([]*carrierintel.CarrierIntelSnapshot, error),
) (*carrierintel.CarrierIntelSnapshot, error) {
	var snapshots []*carrierintel.CarrierIntelSnapshot
	var err error
	if loaders.Available(ctx) {
		snapshots, err = loader()
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
		Text:                   gqlscalar.StringValue(input.Text),
		State:                  strings.ToUpper(gqlscalar.StringValue(input.State)),
		OriginState:            strings.ToUpper(gqlscalar.StringValue(input.OriginState)),
		DestinationState:       strings.ToUpper(gqlscalar.StringValue(input.DestinationState)),
		MinPowerUnits:          input.MinPowerUnits,
		MaxPowerUnits:          input.MaxPowerUnits,
		MinAuthorityAgeDays:    input.MinAuthorityAgeDays,
		MaxAuthorityAgeDays:    input.MaxAuthorityAgeDays,
		HazmatOnly:             gqlscalar.BoolValue(input.HazmatOnly),
		ExcludeBlocking:        gqlscalar.BoolValue(input.ExcludeBlocking),
		ExcludeExistingCarrier: gqlscalar.BoolValue(input.ExcludeExistingCarriers),
		Sort:                   sourcingSortValue(input.Sort),
		Limit:                  gqlscalar.IntValue(input.Limit),
		Offset:                 gqlscalar.IntValue(input.Offset),
	}
}

const carrierIntelReviewQueueCountLimit = 500

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
