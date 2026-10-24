package telematicsrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/telematics"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/telematics/telematicsdb"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/telematicsports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

var formMappingErrors = dberror.Mapping{Entity: "FormMapping"}

func (r *repository) UpsertFormSubmission(
	ctx context.Context,
	submission *telematics.FormSubmission,
) (bool, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (bool, error) {
		fs := telematicsdb.FormSubmissions
		stored, err := fs.Upsert(
			ctx,
			r.db.Keel(ctx),
			[]*telematics.FormSubmission{submission},
			keel.OnConflict(fs.OrganizationID, fs.BusinessUnitID, fs.Provider, fs.ProviderSubmissionID).
				Update(
					fs.TemplateName,
					fs.WorkerID,
					fs.ShipmentID,
					fs.ShipmentMoveID,
					fs.StopID,
					fs.SubmittedAt,
					fs.Fields,
					fs.Applied,
					fs.AppliedFields,
					fs.AppliedAt,
				),
		)
		if err != nil {
			return false, fmt.Errorf("upsert form submission: %w", err)
		}

		return len(stored) > 0, nil
	})
}

func formSubmissionsQuery(
	req *telematicsports.ListFormSubmissionsRequest,
) *keel.Query[telematics.FormSubmission] {
	fs := telematicsdb.FormSubmissions
	q := fs.Where(
		keel.When(req.ShipmentID.IsNotNil(), fs.ShipmentID.Eq(req.ShipmentID)),
		keel.When(req.WorkerID.IsNotNil(), fs.WorkerID.Eq(req.WorkerID)),
		keel.When(req.Since > 0, fs.SubmittedAt.Gte(req.Since)),
	).Order(fs.SubmittedAt.Desc())
	if req.Limit > 0 {
		q = q.Limit(req.Limit)
	}

	return q
}

func (r *repository) ListFormSubmissions(
	ctx context.Context,
	req *telematicsports.ListFormSubmissionsRequest,
) ([]*telematics.FormSubmission, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*telematics.FormSubmission, error) {
		entities, err := formSubmissionsQuery(req).All(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, fmt.Errorf("list form submissions: %w", err)
		}

		return entities, nil
	})
}

func (r *repository) ListFormMappings(
	ctx context.Context,
	req *telematicsports.ListFormMappingsRequest,
) ([]*telematics.FormMapping, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*telematics.FormMapping, error) {
		fm := telematicsdb.FormMappings
		var enabled keel.Pred
		if req.Enabled != nil {
			enabled = fm.Enabled.Eq(*req.Enabled)
		}

		entities, err := fm.Where(
			keel.When(req.Provider != "", fm.Provider.Eq(req.Provider)),
			enabled,
		).
			Load(fm.Items()).
			Order(fm.CreatedAt.Desc()).
			All(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, fmt.Errorf("list form mappings: %w", err)
		}

		return entities, nil
	})
}

func (r *repository) GetFormMapping(
	ctx context.Context,
	_ pagination.TenantInfo,
	id pulid.ID,
) (*telematics.FormMapping, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*telematics.FormMapping, error) {
		fm := telematicsdb.FormMappings
		entity, err := fm.Where(fm.ID.Eq(id)).Load(fm.Items()).One(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, formMappingErrors.Map(err, id.String())
		}

		return entity, nil
	})
}

func (r *repository) SaveFormMapping(
	ctx context.Context,
	mapping *telematics.FormMapping,
	items []*telematics.FormMappingItem,
) (*telematics.FormMapping, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*telematics.FormMapping, error) {
		if err := r.saveFormMapping(ctx, mapping, items); err != nil {
			return nil, fmt.Errorf(
				"save form mapping: %w",
				formMappingErrors.Map(err, mapping.ID.String()),
			)
		}

		mapping.Items = items
		return mapping, nil
	})
}

func (r *repository) saveFormMapping(
	ctx context.Context,
	mapping *telematics.FormMapping,
	items []*telematics.FormMappingItem,
) error {
	db := r.db.Keel(ctx)
	fm := telematicsdb.FormMappings
	fi := telematicsdb.FormMappingItems

	if mapping.ID.IsNil() {
		mapping.ID = telematics.NewFormMappingID()
		mapping.Version = 0
		if err := fm.Create(ctx, db, mapping); err != nil {
			return err
		}
	} else {
		if err := fm.Save(ctx, db, mapping); err != nil {
			return err
		}
		if _, err := fi.Where(fi.MappingID.Eq(mapping.ID)).Delete(ctx, db); err != nil {
			return err
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

	return fi.CreateMany(ctx, db, items)
}

func (r *repository) DeleteFormMapping(
	ctx context.Context,
	_ pagination.TenantInfo,
	id pulid.ID,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		db := r.db.Keel(ctx)
		fi := telematicsdb.FormMappingItems
		if _, err := fi.Where(fi.MappingID.Eq(id)).Delete(ctx, db); err != nil {
			return fmt.Errorf("delete form mapping: %w", err)
		}

		fm := telematicsdb.FormMappings
		if _, err := fm.Where(fm.ID.Eq(id)).Delete(ctx, db); err != nil {
			return fmt.Errorf("delete form mapping: %w", err)
		}

		return nil
	})
}
