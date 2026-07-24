package telematicsrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/telematics"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
)

func (r *repository) UpsertFormSubmission(
	ctx context.Context,
	submission *telematics.FormSubmission,
) (bool, error) {
	cols := buncolgen.FormSubmissionColumns
	result, err := r.db.DB().NewInsert().
		Model(submission).
		On("CONFLICT (organization_id, business_unit_id, provider, provider_submission_id) DO UPDATE").
		Set(cols.TemplateName.SetExcluded()).
		Set(cols.WorkerID.SetExcluded()).
		Set(cols.ShipmentID.SetExcluded()).
		Set(cols.ShipmentMoveID.SetExcluded()).
		Set(cols.StopID.SetExcluded()).
		Set(cols.SubmittedAt.SetExcluded()).
		Set(cols.Fields.SetExcluded()).
		Set(cols.Applied.SetExcluded()).
		Set(cols.AppliedFields.SetExcluded()).
		Set(cols.AppliedAt.SetExcluded()).
		Exec(ctx)
	if err != nil {
		return false, fmt.Errorf("upsert form submission: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("upsert form submission rows affected: %w", err)
	}
	return rows > 0, nil
}

func (r *repository) ListFormSubmissions(
	ctx context.Context,
	req *repositories.ListFormSubmissionsRequest,
) ([]*telematics.FormSubmission, error) {
	cols := buncolgen.FormSubmissionColumns

	entities := make([]*telematics.FormSubmission, 0)
	q := r.db.DB().NewSelect().
		Model(&entities).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			sq = buncolgen.FormSubmissionScopeTenant(sq, req.TenantInfo)
			if !req.ShipmentID.IsNil() {
				sq = sq.Where(cols.ShipmentID.Eq(), req.ShipmentID)
			}
			if !req.WorkerID.IsNil() {
				sq = sq.Where(cols.WorkerID.Eq(), req.WorkerID)
			}
			if req.Since > 0 {
				sq = sq.Where(cols.SubmittedAt.Gte(), req.Since)
			}
			return sq
		}).
		Order(cols.SubmittedAt.OrderDesc())

	if req.Limit > 0 {
		q = q.Limit(req.Limit)
	}

	if err := q.Scan(ctx); err != nil {
		return nil, fmt.Errorf("list form submissions: %w", err)
	}
	return entities, nil
}

func (r *repository) ListFormMappings(
	ctx context.Context,
	req *repositories.ListFormMappingsRequest,
) ([]*telematics.FormMapping, error) {
	cols := buncolgen.FormMappingColumns

	entities := make([]*telematics.FormMapping, 0)
	q := r.db.DB().NewSelect().
		Model(&entities).
		Relation(buncolgen.FormMappingRelations.Items).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			sq = buncolgen.FormMappingScopeTenant(sq, req.TenantInfo)
			if req.Provider != "" {
				sq = sq.Where(cols.Provider.Eq(), req.Provider)
			}
			if req.Enabled != nil {
				sq = sq.Where(cols.Enabled.Eq(), *req.Enabled)
			}
			return sq
		}).
		Order(cols.CreatedAt.OrderDesc())

	if err := q.Scan(ctx); err != nil {
		return nil, fmt.Errorf("list form mappings: %w", err)
	}
	return entities, nil
}

func (r *repository) GetFormMapping(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	id pulid.ID,
) (*telematics.FormMapping, error) {
	entity := new(telematics.FormMapping)
	err := r.db.DB().NewSelect().
		Model(entity).
		Relation(buncolgen.FormMappingRelations.Items).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.FormMappingScopeTenant(sq, tenantInfo).
				Where(buncolgen.FormMappingColumns.ID.Eq(), id)
		}).
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "FormMapping")
	}
	return entity, nil
}

func (r *repository) SaveFormMapping(
	ctx context.Context,
	mapping *telematics.FormMapping,
	items []*telematics.FormMappingItem,
) (*telematics.FormMapping, error) {
	err := r.db.DB().RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		if mapping.ID.IsNil() {
			mapping.ID = telematics.NewFormMappingID()
			mapping.Version = 0
			if _, insErr := tx.NewInsert().Model(mapping).Exec(ctx); insErr != nil {
				return insErr
			}
		} else {
			ov := mapping.Version
			mapping.Version++
			result, updErr := tx.NewUpdate().
				Model(mapping).
				WherePK().
				Where(buncolgen.FormMappingColumns.Version.Eq(), ov).
				OmitZero().
				Returning("*").
				Exec(ctx)
			if updErr != nil {
				return updErr
			}
			if raErr := dberror.CheckRowsAffected(result, "FormMapping", mapping.ID.String()); raErr != nil {
				return raErr
			}
			if _, delErr := tx.NewDelete().
				Model((*telematics.FormMappingItem)(nil)).
				Where(buncolgen.FormMappingItemColumns.MappingID.Eq(), mapping.ID).
				Where(buncolgen.FormMappingItemColumns.OrganizationID.Eq(), mapping.OrganizationID).
				Where(buncolgen.FormMappingItemColumns.BusinessUnitID.Eq(), mapping.BusinessUnitID).
				Exec(ctx); delErr != nil {
				return delErr
			}
		}

		if len(items) == 0 {
			return nil
		}
		for _, item := range items {
			item.ID = telematics.NewFormMappingItemID()
			item.MappingID = mapping.ID
			item.OrganizationID = mapping.OrganizationID
			item.BusinessUnitID = mapping.BusinessUnitID
		}
		if _, insErr := tx.NewInsert().Model(&items).Exec(ctx); insErr != nil {
			return insErr
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("save form mapping: %w", err)
	}
	mapping.Items = items
	return mapping, nil
}

func (r *repository) DeleteFormMapping(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	id pulid.ID,
) error {
	err := r.db.DB().RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		if _, delErr := tx.NewDelete().
			Model((*telematics.FormMappingItem)(nil)).
			Where(buncolgen.FormMappingItemColumns.MappingID.Eq(), id).
			Where(buncolgen.FormMappingItemColumns.OrganizationID.Eq(), tenantInfo.OrgID).
			Where(buncolgen.FormMappingItemColumns.BusinessUnitID.Eq(), tenantInfo.BuID).
			Exec(ctx); delErr != nil {
			return delErr
		}
		_, delErr := tx.NewDelete().
			Model((*telematics.FormMapping)(nil)).
			WhereGroup(" AND ", func(dq *bun.DeleteQuery) *bun.DeleteQuery {
				return buncolgen.FormMappingScopeTenantDelete(dq, tenantInfo).
					Where(buncolgen.FormMappingColumns.ID.Eq(), id)
			}).
			Exec(ctx)
		return delErr
	})
	if err != nil {
		return fmt.Errorf("delete form mapping: %w", err)
	}
	return nil
}
