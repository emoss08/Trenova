package agentquerytoolservice

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/rating/ports/rateimportports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/typeutils"
)

const (
	paramFuelIndexID    = "fuelIndexId"
	fieldTractorID      = "tractorId"
	fieldJurisdictionID = "jurisdictionId"
	rateImportsKey      = "imports"
)

func fuelRateReadProviders() []any {
	return []any{
		newListRateImportsTool,
	}
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
) agenttoolports.AgentQueryTool {
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

func (t *listRateImportsTool) Policy() agenttoolports.ToolPolicy {
	return readPolicy(t.Name(), readSpec{resource: permission.ResourceRateAgreement})
}

func (t *listRateImportsTool) Query(
	ctx context.Context,
	params *agenttoolports.QueryToolParams,
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

	return map[string]any{
		rateImportsKey:   rows,
		"total":          result.Total,
		"totalEstimated": !result.TotalExact,
	}, nil
}
