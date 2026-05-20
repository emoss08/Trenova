//nolint:gocritic // Repository request structs follow the existing value-parameter port contracts.
package edicontrolnumberrepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) repositories.EDIControlNumberRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.edi-control-number-repository"),
	}
}

func (r *repository) AllocateControlNumbers(
	ctx context.Context,
	req repositories.AllocateEDIControlNumbersRequest,
) (map[edi.ControlNumberKind]int64, error) {
	allocated := make(map[edi.ControlNumberKind]int64, len(req.Kinds))
	cols := buncolgen.EDIControlNumberSequenceColumns

	err := r.db.WithTx(ctx, ports.TxOptions{}, func(c context.Context, _ bun.Tx) error {
		for _, kind := range req.Kinds {
			sequence := &edi.EDIControlNumberSequence{
				BusinessUnitID: req.TenantInfo.BuID,
				OrganizationID: req.TenantInfo.OrgID,
				EDIPartnerID:   req.PartnerID,
				DocumentTypeID: req.DocumentTypeID,
				Kind:           kind,
			}
			_, err := r.db.DBForContext(c).
				NewInsert().
				Model(sequence).
				On(`CONFLICT ("edi_partner_id", "business_unit_id", "organization_id", "document_type_id", "kind") DO NOTHING`).
				Exec(c)
			if err != nil {
				return err
			}

			if err = r.db.DBForContext(c).
				NewSelect().
				Model(sequence).
				Where(cols.EDIPartnerID.Eq(), req.PartnerID).
				Where(cols.BusinessUnitID.Eq(), req.TenantInfo.BuID).
				Where(cols.OrganizationID.Eq(), req.TenantInfo.OrgID).
				Where(cols.DocumentTypeID.Eq(), req.DocumentTypeID).
				Where(cols.Kind.Eq(), kind).
				For("UPDATE").
				Scan(c); err != nil {
				return err
			}

			value := sequence.NextValue
			next := value + 1
			if next > sequence.MaxValue {
				next = sequence.MinValue
			}
			sequence.NextValue = next
			sequence.Version++
			if _, err = r.db.DBForContext(c).
				NewUpdate().
				Model(sequence).
				WherePK().
				Column(cols.NextValue.Bare(), cols.Version.Bare(), cols.UpdatedAt.Bare()).
				Exec(c); err != nil {
				return err
			}
			allocated[kind] = value
		}
		return nil
	})
	return allocated, err
}
