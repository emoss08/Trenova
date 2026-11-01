package rateimportresolver

import (
	"context"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlid"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlscalar"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvercore"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/rating/app/rateimportservice"
	"github.com/emoss08/trenova/internal/modules/rating/domain/rateimport"
	"github.com/emoss08/trenova/pkg/pagination"
)

const defaultRowPage = 100

func boundedFirst(first *int) int {
	if first == nil {
		return defaultRowPage
	}

	return min(max(*first, 1), pagination.MaxLimit)
}

type transitionFunc func(
	ctx context.Context,
	req *rateimportservice.CommitRequest,
) (*rateimport.RateImportBatch, error)

func (d *Deps) transition(
	ctx context.Context,
	id string,
	fn transitionFunc,
) (*gqlmodel.RateImportBatch, error) {
	authCtx, err := d.RequirePermission(ctx, permission.ResourceRateAgreement, permission.OpUpdate)
	if err != nil {
		return nil, err
	}

	importID, err := gqlid.ParseIDField("id", id)
	if err != nil {
		return nil, err
	}

	batch, err := fn(ctx, &rateimportservice.CommitRequest{
		TenantInfo:        resolvercore.TenantInfo(authCtx),
		RateImportBatchID: importID,
	})
	if err != nil {
		return nil, err
	}

	return rateImportBatchToModel(batch), nil
}

func rateImportBatchToModel(entity *rateimport.RateImportBatch) *gqlmodel.RateImportBatch {
	var mapping map[string]any
	if len(entity.Mapping) > 0 {
		mapping = make(map[string]any, len(entity.Mapping))
		for field, column := range entity.Mapping {
			mapping[field] = column
		}
	}
	changes := make([]*gqlmodel.RateImportChange, 0, len(entity.Changes))
	for _, change := range entity.Changes {
		fields := make([]*gqlmodel.RateImportFieldChange, 0, len(change.Fields))
		for _, field := range change.Fields {
			fields = append(fields, &gqlmodel.RateImportFieldChange{
				Field:  field.Field,
				Before: field.Before,
				After:  field.After,
			})
		}
		changes = append(changes, &gqlmodel.RateImportChange{
			Kind:       change.Kind,
			LaneKey:    change.LaneKey,
			Label:      change.Label,
			ExistingID: gqlscalar.EmptyToNil(change.ExistingID),
			Fields:     fields,
		})
	}
	var summary *gqlmodel.RateImportSummary
	if entity.Summary != nil {
		summary = &gqlmodel.RateImportSummary{
			Added:     entity.Summary.Added,
			Changed:   entity.Summary.Changed,
			Removed:   entity.Summary.Removed,
			Duplicate: entity.Summary.Duplicate,
			Unchanged: entity.Summary.Unchanged,
		}
	}
	unmapped := entity.UnmappedHeaders
	if unmapped == nil {
		unmapped = []string{}
	}

	return &gqlmodel.RateImportBatch{
		ID:              entity.ID.String(),
		BusinessUnitID:  entity.BusinessUnitID.String(),
		OrganizationID:  entity.OrganizationID.String(),
		RateAgreementID: entity.RateAgreementID.String(),
		FileName:        entity.FileName,
		SourceFormat:    entity.SourceFormat,
		Status:          entity.Status,
		EffectiveFrom:   int(entity.EffectiveFrom),
		Mapping:         mapping,
		UnmappedHeaders: unmapped,
		Changes:         changes,
		Summary:         summary,
		RowCount:        entity.RowCount,
		ErrorCount:      entity.ErrorCount,
		Error:           entity.Error,
		UploadedByID:    gqlid.IDPtrFromPulidPtr(entity.UploadedByID),
		CommittedAt:     gqlscalar.Int64PtrToIntPtr(entity.CommittedAt),
		CommittedBy:     gqlid.IDPtrFromPulidPtr(entity.CommittedBy),
		Version:         int(entity.Version),
		CreatedAt:       int(entity.CreatedAt),
		UpdatedAt:       int(entity.UpdatedAt),
	}
}

func rateImportRowsToModel(rows []*rateimport.RateImportRow) []*gqlmodel.RateImportRow {
	out := make([]*gqlmodel.RateImportRow, 0, len(rows))
	for _, row := range rows {
		if row == nil {
			continue
		}
		cells := row.Cells
		if cells == nil {
			cells = []string{}
		}
		out = append(out, &gqlmodel.RateImportRow{
			ID:                row.ID.String(),
			RateImportBatchID: row.RateImportBatchID.String(),
			RowNumber:         row.RowNumber,
			Cells:             cells,
			LaneKey:           row.LaneKey,
			Error:             row.Error,
			CreatedAt:         int(row.CreatedAt),
		})
	}

	return out
}
