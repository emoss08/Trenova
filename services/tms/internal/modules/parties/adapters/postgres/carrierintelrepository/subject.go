package carrierintelrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant/tenantdb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/tender"
	"github.com/emoss08/trenova/internal/modules/operations/domain/tender/tenderdb"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrier"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrier/carrierdb"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrierintel"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer/customerdb"
	"github.com/emoss08/trenova/internal/modules/parties/ports/carrierintelports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/intutils"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

const (
	defaultCarrierSubjectLimit = 1000
	maxCarrierSubjectLimit     = 5000
	customerEntityName         = "Customer"
	organizationEntityName     = "Organization"
)

var (
	openCarrierAssignmentStatuses = []shipment.CarrierAssignmentStatus{
		shipment.CarrierAssignmentStatusPending,
		shipment.CarrierAssignmentStatusConfirmed,
	}
	openTenderOfferStatuses = []tender.OfferStatus{
		tender.OfferStatusPending,
		tender.OfferStatusSent,
	}
)

type carrierSubjectRow struct {
	ID          pulid.ID
	Name        string
	DOTNumber   string
	MCNumber    string
	CarrierType carrier.Type
	LastUsedAt  *int64
}

type customerSubjectRow struct {
	ID        pulid.ID
	Name      string
	DOTNumber string
	MCNumber  string
}

type carrierDOTRow struct {
	ID        pulid.ID
	DOTNumber string
}

type subjectRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewSubjectRepository(p Params) carrierintelports.CarrierIntelSubjectRepository {
	return &subjectRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.carrier-intel-subject-repository"),
	}
}

func buildCarrierSubjectsQuery(
	req *carrierintelports.ListCarrierIntelSubjectsRequest,
	limit int,
) *keel.Query[carrier.Carrier] {
	c := carrierdb.Carriers
	ca := shipmentdb.CarrierAssignments
	to := tenderdb.TenderOffers

	lastAssignment := keel.Scalar(ca.Pick(ca.CreatedAt.Max()).Where(ca.CarrierID.EqCol(c.ID)))
	lastOffer := keel.Scalar(to.Pick(to.CreatedAt.Max()).Where(to.CarrierID.EqCol(c.ID)))

	q := c.Pick(
		c.ID,
		keel.Raw("coalesce(?, '')", c.Name),
		c.DOTNumber,
		keel.Raw("coalesce(?, '')", c.MCNumber),
		c.CarrierType,
		keel.Raw("GREATEST(?, ?)", lastAssignment, lastOffer),
	).
		Where(
			c.DOTNumber.IsNotNull(),
			keel.When(req.ActiveOnly, c.Status.Eq(carrier.StatusActive)),
			keel.When(len(req.CarrierIDs) > 0, c.ID.In(req.CarrierIDs...)),
			keel.When(!req.AfterID.IsNil(), c.ID.Gt(req.AfterID)),
		).
		Order(c.ID.Asc()).
		Limit(limit)

	if req.UsedSince != nil {
		recentAssignment := keel.Exists(ca.Pick(ca.ID).Where(
			ca.CarrierID.EqCol(c.ID),
			keel.Or(
				ca.CreatedAt.Gte(*req.UsedSince),
				ca.Status.In(openCarrierAssignmentStatuses...),
			),
		))
		openOffer := keel.Exists(to.Pick(to.ID).Where(
			to.CarrierID.EqCol(c.ID),
			to.Status.In(openTenderOfferStatuses...),
		))
		q = q.Where(keel.Or(recentAssignment, keel.When(req.IncludeOpenTenders, openOffer)))
	}

	return q
}

func (r *subjectRepository) ListCarrierSubjects(
	ctx context.Context,
	req *carrierintelports.ListCarrierIntelSubjectsRequest,
) ([]carrierintelports.CarrierIntelSubject, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]carrierintelports.CarrierIntelSubject, error) {
			limit := intutils.Clamp(
				intutils.WithDefault(max(req.Limit, 0), defaultCarrierSubjectLimit),
				1,
				maxCarrierSubjectLimit,
			)

			rows, err := keel.Into[carrierSubjectRow](
				ctx,
				r.db.Keel(ctx),
				buildCarrierSubjectsQuery(req, limit),
			)
			if err != nil {
				r.l.Error("failed to list carrier intel carrier subjects", zap.Error(err))
				return nil, fmt.Errorf("list carrier intel carrier subjects: %w", err)
			}

			subjects := make([]carrierintelports.CarrierIntelSubject, 0, len(rows))
			for i := range rows {
				row := &rows[i]
				subjects = append(subjects, carrierintelports.CarrierIntelSubject{
					SubjectType:  carrierintel.SubjectTypeCarrier,
					SubjectID:    row.ID.String(),
					CarrierID:    row.ID,
					Name:         row.Name,
					DOTNumber:    row.DOTNumber,
					DocketNumber: row.MCNumber,
					LastUsedAt:   row.LastUsedAt,
					Broker:       row.CarrierType == carrier.TypeBroker,
					Exempt:       row.CarrierType == carrier.TypeExempt,
				})
			}

			return subjects, nil
		},
	)
}

func customerSubject(row *customerSubjectRow) carrierintelports.CarrierIntelSubject {
	return carrierintelports.CarrierIntelSubject{
		SubjectType:  carrierintel.SubjectTypeCustomer,
		SubjectID:    row.ID.String(),
		Name:         row.Name,
		DOTNumber:    row.DOTNumber,
		DocketNumber: row.MCNumber,
		Broker:       true,
	}
}

func customerSubjectQuery(preds ...keel.Pred) *keel.Query[customer.Customer] {
	c := customerdb.Customers
	return c.Pick(
		c.ID,
		keel.Raw("coalesce(?, '')", c.Name),
		keel.Raw("coalesce(?, '')", c.DOTNumber),
		keel.Raw("coalesce(?, '')", c.MCNumber),
	).Where(preds...)
}

func (r *subjectRepository) ListBrokerCustomerSubjects(
	ctx context.Context,
	_ pagination.TenantInfo,
) ([]carrierintelports.CarrierIntelSubject, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]carrierintelports.CarrierIntelSubject, error) {
			c := customerdb.Customers
			rows, err := keel.Into[customerSubjectRow](
				ctx,
				r.db.Keel(ctx),
				customerSubjectQuery(c.BrokerVettingEnabled.IsTrue(), c.DOTNumber.IsNotNull()).
					Order(c.ID.Asc()),
			)
			if err != nil {
				r.l.Error("failed to list carrier intel broker customer subjects", zap.Error(err))
				return nil, fmt.Errorf("list carrier intel broker customer subjects: %w", err)
			}

			subjects := make([]carrierintelports.CarrierIntelSubject, 0, len(rows))
			for i := range rows {
				subjects = append(subjects, customerSubject(&rows[i]))
			}

			return subjects, nil
		},
	)
}

func (r *subjectRepository) GetOrganizationSubject(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*carrierintelports.CarrierIntelSubject, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*carrierintelports.CarrierIntelSubject, error) {
			o := tenantdb.Organizations
			org, err := o.Pick(o.ID, o.Name, o.DOTNumber).
				Where(o.ID.Eq(tenantInfo.OrgID)).
				One(ctx, r.db.Keel(ctx))
			if dberror.IsNotFoundError(err) {
				return nil, dberror.HandleNotFoundError(err, organizationEntityName)
			}
			if err != nil {
				r.l.Error("failed to get carrier intel organization subject", zap.Error(err))
				return nil, fmt.Errorf("get carrier intel organization subject: %w", err)
			}
			if org.DOTNumber == "" {
				return nil, nil //nolint:nilnil // an organization without a DOT number cannot be monitored
			}

			return &carrierintelports.CarrierIntelSubject{
				SubjectType: carrierintel.SubjectTypeOrganization,
				SubjectID:   tenantInfo.OrgID.String(),
				Name:        org.Name,
				DOTNumber:   org.DOTNumber,
			}, nil
		},
	)
}

func (r *subjectRepository) GetCustomerSubject(
	ctx context.Context,
	_ pagination.TenantInfo,
	customerID pulid.ID,
) (*carrierintelports.CarrierIntelSubject, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*carrierintelports.CarrierIntelSubject, error) {
			c := customerdb.Customers
			row, err := keel.IntoOne[customerSubjectRow](
				ctx,
				r.db.Keel(ctx),
				customerSubjectQuery(c.ID.Eq(customerID)),
			)
			if dberror.IsNotFoundError(err) {
				return nil, dberror.HandleNotFoundError(err, customerEntityName)
			}
			if err != nil {
				r.l.Error("failed to get carrier intel customer subject", zap.Error(err))
				return nil, fmt.Errorf("get carrier intel customer subject: %w", err)
			}

			subject := customerSubject(&row)
			return &subject, nil
		},
	)
}

func (r *subjectRepository) ListExistingCarrierDOTs(
	ctx context.Context,
	_ pagination.TenantInfo,
	dotNumbers []string,
) (map[string]pulid.ID, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (map[string]pulid.ID, error) {
		if len(dotNumbers) == 0 {
			return map[string]pulid.ID{}, nil
		}

		c := carrierdb.Carriers
		rows, err := keel.Into[carrierDOTRow](
			ctx,
			r.db.Keel(ctx),
			c.Pick(c.ID, c.DOTNumber).
				Where(c.DOTNumber.In(dotNumbers...)).
				Order(c.CreatedAt.Asc(), c.ID.Asc()),
		)
		if err != nil {
			r.l.Error("failed to list existing carrier DOT numbers", zap.Error(err))
			return nil, fmt.Errorf("list existing carrier DOT numbers: %w", err)
		}

		existing := make(map[string]pulid.ID, len(rows))
		for _, row := range rows {
			if _, ok := existing[row.DOTNumber]; !ok {
				existing[row.DOTNumber] = row.ID
			}
		}

		return existing, nil
	})
}
