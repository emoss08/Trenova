package agentquerytoolservice

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/rating/ports/fuelsurchargeports"
	"github.com/emoss08/trenova/internal/modules/rating/ports/rateimportports"
	"github.com/emoss08/trenova/internal/modules/workspace/app/reporting"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/report"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/typeutils"
)

const tractorIDSource = "from equipment_list_tractors"

const (
	paramFuelIndexID       = "fuelIndexId"
	paramReportScheduleDef = "definitionId"
	fieldQuarter           = "quarter"
	fieldYear              = "year"
	fieldSource            = "source"
	fieldTractorID         = "tractorId"
	fieldJurisdictionID    = "jurisdictionId"
	maxFuelIndexPrices     = 104
	defaultFuelIndexPrices = 26
	maxReportSchedules     = 50
	priceDayLayout         = "2006-01-02"
)

func fuelRateReadProviders() []any {
	return []any{
		newListFuelIndexPricesTool,
		newListRateImportsTool,
		provideListReportSchedulesTool,
	}
}

type fuelIndexPriceRow struct {
	ID        string `json:"id"`
	PriceDate string `json:"priceDate"`
	Price     string `json:"price"`
	Currency  string `json:"currency"`
	IsManual  bool   `json:"isManual"`
}

type listFuelIndexPricesTool struct {
	prices fuelsurchargeports.FuelIndexPriceRepository
}

func newListFuelIndexPricesTool(
	prices fuelsurchargeports.FuelIndexPriceRepository,
) agentports.AgentQueryTool {
	return &listFuelIndexPricesTool{prices: prices}
}

func (t *listFuelIndexPricesTool) Name() string { return "rate_list_fuel_index_prices" }

func (t *listFuelIndexPricesTool) Description() string {
	return "List the prices on file for one fuel index, newest first, with the day each " +
		"applies to and whether a person entered it or it was fetched. Only a manual price " +
		"on a custom index can be corrected, with rate_correct_fuel_index_price; " +
		"rate_record_fuel_index_price adds the next one."
}

func (t *listFuelIndexPricesTool) ParamSchema() map[string]any {
	return objectSchema(map[string]any{
		paramFuelIndexID: agenttoolschema.KindID("The fuel index, from "+
			"rate_list_fuel_surcharge_programs. Never guess one.", permission.KindFuelIndex),
		"from":     agenttoolschema.Date("The earliest price day to include."),
		"to":       agenttoolschema.Date("The latest price day to include."),
		paramLimit: limitSchema(defaultFuelIndexPrices, maxFuelIndexPrices),
	}, paramFuelIndexID)
}

func (t *listFuelIndexPricesTool) Policy() agentports.ToolPolicy {
	return readPolicy(t.Name(), readSpec{resource: permission.ResourceFuelSurchargeProgram})
}

func optionalPriceDay(params map[string]any, key string) (string, error) {
	day := strings.TrimSpace(optionalString(params, key))
	if day == "" {
		return "", nil
	}
	if _, err := time.Parse(priceDayLayout, day); err != nil {
		return "", fmt.Errorf("%s must be YYYY-MM-DD, got %q", key, day)
	}

	return day, nil
}

func (t *listFuelIndexPricesTool) Query(
	ctx context.Context,
	params *agentports.QueryToolParams,
) (any, error) {
	if err := guardQuery(params); err != nil {
		return nil, err
	}

	indexID, err := requirePulid(params.Params, paramFuelIndexID)
	if err != nil {
		return nil, err
	}
	from, err := optionalPriceDay(params.Params, "from")
	if err != nil {
		return nil, err
	}
	to, err := optionalPriceDay(params.Params, "to")
	if err != nil {
		return nil, err
	}

	prices, err := t.prices.ListByIndex(ctx, &fuelsurchargeports.ListFuelIndexPricesRequest{
		FuelIndexID: indexID,
		TenantInfo:  tenantOf(params),
		From:        from,
		To:          to,
		Limit: min(
			max(optionalInt(params.Params, paramLimit, defaultFuelIndexPrices), 1),
			maxFuelIndexPrices,
		),
	})
	if err != nil {
		return nil, err
	}

	rows := make([]fuelIndexPriceRow, 0, len(prices))
	for _, price := range prices {
		rows = append(rows, fuelIndexPriceRow{
			ID:        price.ID.String(),
			PriceDate: price.PriceDate,
			Price:     price.Price.String(),
			Currency:  price.Currency,
			IsManual:  price.IsManual,
		})
	}

	return map[string]any{"fuelIndexId": indexID.String(), "prices": rows}, nil
}

type rateImportRow struct {
	ID              string       `json:"id"`
	RateAgreementID string       `json:"rateAgreementId"`
	FileName        string       `json:"fileName"`
	Status          string       `json:"status"`
	EffectiveFrom   optionalDate `json:"effectiveFrom"`
	RowCount        int          `json:"rowCount"`
	ErrorCount      int          `json:"errorCount"`
	Added           int          `json:"lanesAdded"`
	Changed         int          `json:"lanesChanged"`
	Removed         int          `json:"lanesRemoved"`
	Unchanged       int          `json:"lanesUnchanged"`
	CommittedAt     optionalDate `json:"committedAt"`
	Error           string       `json:"error,omitempty"`
}

type listRateImportsTool struct {
	imports rateimportports.RateImportRepository
}

func newListRateImportsTool(
	imports rateimportports.RateImportRepository,
) agentports.AgentQueryTool {
	return &listRateImportsTool{imports: imports}
}

func (t *listRateImportsTool) Name() string { return "rate_list_imports" }

func (t *listRateImportsTool) SearchTerms() []string {
	return []string{"rate sheet imports", "imports waiting review", "pending rate imports"}
}

func (t *listRateImportsTool) Description() string {
	return "List rate sheets uploaded against rate agreements, newest first, with the " +
		"agreement and the day the new rates take effect. Each says how many lanes the " +
		"sheet adds, changes and removes. A Parsed import waits for rate_commit_import or " +
		"rate_discard_import."
}

func (t *listRateImportsTool) ParamSchema() map[string]any {
	return objectSchema(map[string]any{
		"rateAgreementId": agenttoolschema.RecordIDText(permission.ResourceRateAgreement,
			"Only the imports against this agreement, from rate_list_agreements."),
		paramLimit: limitSchema(defaultListLimit, maxListLimit),
	})
}

func (t *listRateImportsTool) Policy() agentports.ToolPolicy {
	return readPolicy(t.Name(), readSpec{resource: permission.ResourceRateAgreement})
}

func (t *listRateImportsTool) Query(
	ctx context.Context,
	params *agentports.QueryToolParams,
) (any, error) {
	if err := guardQuery(params); err != nil {
		return nil, err
	}

	req := &rateimportports.ListRateImportBatchesRequest{
		Filter: &pagination.QueryOptions{
			TenantInfo: tenantOf(params),
			Pagination: pagination.Info{Limit: min(
				max(optionalInt(params.Params, paramLimit, defaultListLimit), 1),
				maxListLimit,
			)},
		},
	}
	if raw := strings.TrimSpace(optionalString(params.Params, "rateAgreementId")); raw != "" {
		agreementID, err := pulid.Parse(raw)
		if err != nil {
			return nil, fmt.Errorf("%q is not a rate agreement id; find it with "+
				"rate_list_agreements", raw)
		}
		req.RateAgreementID = &agreementID
	}

	result, err := t.imports.List(ctx, req)
	if err != nil {
		return nil, err
	}

	rows := make([]rateImportRow, 0, len(result.Items))
	for _, item := range result.Items {
		row := rateImportRow{
			ID:              item.ID.String(),
			RateAgreementID: item.RateAgreementID.String(),
			FileName:        item.FileName,
			Status:          string(item.Status),
			EffectiveFrom:   recordedDate(item.EffectiveFrom),
			RowCount:        item.RowCount,
			ErrorCount:      item.ErrorCount,
			CommittedAt:     expectedDate(typeutils.ValueOrZero(item.CommittedAt), "not committed"),
			Error:           item.Error,
		}
		if item.Summary != nil {
			row.Added = item.Summary.Added
			row.Changed = item.Summary.Changed
			row.Removed = item.Summary.Removed
			row.Unchanged = item.Summary.Unchanged
		}
		rows = append(rows, row)
	}

	return map[string]any{"imports": rows, "total": result.Total}, nil
}

type reportScheduleReader interface {
	ListSchedules(
		ctx context.Context,
		req *reporting.ListSchedulesRequest,
	) ([]*report.ReportSchedule, error)
}

type reportScheduleRow struct {
	ID             string       `json:"id"`
	DefinitionID   string       `json:"definitionId"`
	CronExpression string       `json:"cronExpression"`
	Timezone       string       `json:"timezone"`
	Formats        []string     `json:"formats"`
	Recipients     []string     `json:"emailRecipients"`
	Enabled        bool         `json:"enabled"`
	Yours          bool         `json:"yours"`
	NextRunAt      optionalDate `json:"nextRunAt"`
	Failures       int          `json:"consecutiveFailures,omitempty"`
}

type listReportSchedulesTool struct {
	schedules reportScheduleReader
}

func provideListReportSchedulesTool(schedules *reporting.Service) agentports.AgentQueryTool {
	return &listReportSchedulesTool{schedules: schedules}
}

func (t *listReportSchedulesTool) Name() string { return "report_list_schedules" }

func (t *listReportSchedulesTool) SearchTerms() []string {
	return []string{"scheduled", "emailed", "recurring", "subscription"}
}

func (t *listReportSchedulesTool) Description() string {
	return "List reports on a schedule: when each runs, in what formats, who it is emailed " +
		"to, whether it is on and when it runs next. Only the person who set a schedule up " +
		"can change it with report_update_schedule or remove it with " +
		"report_delete_schedule; yours says which are theirs."
}

func (t *listReportSchedulesTool) ParamSchema() map[string]any {
	return objectSchema(map[string]any{
		paramReportScheduleDef: agenttoolschema.RecordIDText(permission.ResourceReport,
			"Only the schedules of this saved report, from report_list."),
		"enabledOnly": boolParam("Only schedules that are switched on."),
		paramLimit:    limitSchema(maxReportSchedules, maxReportSchedules),
	})
}

func (t *listReportSchedulesTool) Policy() agentports.ToolPolicy {
	return readPolicy(t.Name(), readSpec{resource: permission.ResourceReport})
}

func (t *listReportSchedulesTool) Query(
	ctx context.Context,
	params *agentports.QueryToolParams,
) (any, error) {
	if err := guardQuery(params); err != nil {
		return nil, err
	}

	definitionID := pulid.Nil
	if raw := strings.TrimSpace(optionalString(params.Params, paramReportScheduleDef)); raw != "" {
		parsed, err := pulid.Parse(raw)
		if err != nil {
			return nil, fmt.Errorf("%q is not a report id; find it with report_list", raw)
		}
		definitionID = parsed
	}

	tenant := tenantOf(params)
	schedules, err := t.schedules.ListSchedules(ctx, &reporting.ListSchedulesRequest{
		Request:      reporting.Request{TenantInfo: tenant},
		DefinitionID: definitionID,
		EnabledOnly:  optionalBool(params.Params, "enabledOnly"),
		Limit: min(
			max(optionalInt(params.Params, paramLimit, maxReportSchedules), 1),
			maxReportSchedules,
		),
	})
	if err != nil {
		return nil, err
	}

	rows := make([]reportScheduleRow, 0, len(schedules))
	for _, schedule := range schedules {
		row := reportScheduleRow{
			ID:             schedule.ID.String(),
			DefinitionID:   schedule.DefinitionID.String(),
			CronExpression: schedule.CronExpression,
			Timezone:       schedule.Timezone,
			Formats:        schedule.Formats,
			Enabled:        schedule.Enabled,
			Yours:          schedule.RunAsID == tenant.UserID,
			NextRunAt:      expectedDate(schedule.NextRunAt, "not scheduled"),
			Failures:       schedule.ConsecutiveFailures,
		}
		if schedule.Delivery != nil {
			row.Recipients = schedule.Delivery.EmailRecipients
		}
		rows = append(rows, row)
	}

	return map[string]any{"schedules": rows}, nil
}
