//nolint:gocritic // Repository request structs follow the existing value-parameter port contracts.
package edicontrolnumberrepository

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi/edidb"
	"github.com/emoss08/trenova/internal/modules/edi/ports/ediports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var sequenceErrors = dberror.Mapping{Entity: "EDIControlNumberSequence"}

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) ediports.EDIControlNumberRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.edi-control-number-repository"),
	}
}

type sequenceKey struct {
	partnerID      pulid.ID
	documentTypeID pulid.ID
	kind           edi.ControlNumberKind
}

func (k sequenceKey) where() []keel.Pred {
	es := edidb.EDIControlNumberSequences
	return []keel.Pred{
		es.EDIPartnerID.Eq(k.partnerID),
		es.DocumentTypeID.Eq(k.documentTypeID),
		es.Kind.Eq(k.kind),
	}
}

func (r *repository) AllocateControlNumbers(
	ctx context.Context,
	req ediports.AllocateEDIControlNumbersRequest,
) (map[edi.ControlNumberKind]int64, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (map[edi.ControlNumberKind]int64, error) {
			db := r.db.Keel(ctx)
			es := edidb.EDIControlNumberSequences
			allocated := make(map[edi.ControlNumberKind]int64, len(req.Kinds))
			for _, kind := range req.Kinds {
				key := sequenceKey{
					partnerID:      req.PartnerID,
					documentTypeID: req.DocumentTypeID,
					kind:           kind,
				}
				if err := createIfMissing(ctx, db, &edi.EDIControlNumberSequence{
					BusinessUnitID: req.TenantInfo.BuID,
					OrganizationID: req.TenantInfo.OrgID,
					EDIPartnerID:   req.PartnerID,
					DocumentTypeID: req.DocumentTypeID,
					Kind:           kind,
				}); err != nil {
					return nil, err
				}

				sequence, err := es.Where(key.where()...).Lock(keel.ForUpdate).One(ctx, db)
				if err != nil {
					return nil, sequenceErrors.Map(err, string(kind))
				}

				value := sequence.NextValue
				next := value + 1
				if next > sequence.MaxValue {
					next = sequence.MinValue
				}
				sequence.NextValue = next
				if err = es.Save(ctx, db, sequence, es.NextValue); err != nil {
					return nil, sequenceErrors.Map(err, sequence.ID.String())
				}
				allocated[kind] = value
			}

			return allocated, nil
		},
	)
}

func (r *repository) ResetControlNumber(
	ctx context.Context,
	req *ediports.ResetEDIControlNumberRequest,
) (*edi.EDIControlNumberSequence, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*edi.EDIControlNumberSequence, error) {
		es := edidb.EDIControlNumberSequences
		key := sequenceKey{
			partnerID:      req.PartnerID,
			documentTypeID: req.DocumentTypeID,
			kind:           req.Kind,
		}
		rows, err := es.Where(key.where()...).
			UpdateReturning(ctx, r.db.Keel(ctx), es.NextValue.To(req.NextValue))
		if err != nil {
			return nil, sequenceErrors.Map(err, string(req.Kind))
		}
		if len(rows) == 0 {
			return nil, sequenceErrors.Map(keel.ErrNotFound, string(req.Kind))
		}

		return rows[0], nil
	})
}

func (r *repository) StartControlNumber(
	ctx context.Context,
	req *ediports.ResetEDIControlNumberRequest,
) (*edi.EDIControlNumberSequence, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*edi.EDIControlNumberSequence, error) {
		db := r.db.Keel(ctx)
		es := edidb.EDIControlNumberSequences
		key := sequenceKey{
			partnerID:      req.PartnerID,
			documentTypeID: req.DocumentTypeID,
			kind:           req.Kind,
		}
		if err := createIfMissing(ctx, db, &edi.EDIControlNumberSequence{
			BusinessUnitID: req.TenantInfo.BuID,
			OrganizationID: req.TenantInfo.OrgID,
			EDIPartnerID:   req.PartnerID,
			DocumentTypeID: req.DocumentTypeID,
			Kind:           req.Kind,
			NextValue:      req.NextValue,
		}); err != nil {
			return nil, err
		}

		if _, err := es.Where(key.where()...).
			Where(es.NextValue.Lt(req.NextValue)).
			Update(ctx, db, es.NextValue.To(req.NextValue)); err != nil {
			return nil, sequenceErrors.Map(err, string(req.Kind))
		}

		current, err := es.Where(key.where()...).One(ctx, db)
		if err != nil {
			return nil, sequenceErrors.Map(err, string(req.Kind))
		}

		return current, nil
	})
}

func createIfMissing(
	ctx context.Context,
	db *keel.DB,
	sequence *edi.EDIControlNumberSequence,
) error {
	es := edidb.EDIControlNumberSequences
	_, err := es.CreateOrSkip(
		ctx,
		db,
		[]*edi.EDIControlNumberSequence{sequence},
		es.EDIPartnerID,
		es.BusinessUnitID,
		es.OrganizationID,
		es.DocumentTypeID,
		es.Kind,
	)

	return sequenceErrors.Map(err, string(sequence.Kind))
}
