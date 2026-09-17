package carrierintelrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/carrier"
	"github.com/emoss08/trenova/internal/core/domain/carrierintel"
	"github.com/emoss08/trenova/internal/core/domain/customer"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/domain/tender"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/intutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

const (
	defaultCarrierSubjectLimit = 1000
	maxCarrierSubjectLimit     = 5000
	brokerCustomerCapacity     = 16
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

func NewSubjectRepository(p Params) repositories.CarrierIntelSubjectRepository {
	return &subjectRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.carrier-intel-subject-repository"),
	}
}

func assignmentActivityQuery(db bun.IDB) *bun.SelectQuery {
	carr := buncolgen.CarrierColumns
	casn := buncolgen.CarrierAssignmentColumns
	return db.NewSelect().
		Model((*shipment.CarrierAssignment)(nil)).
		Where(casn.CarrierID.EqColumn(carr.ID)).
		Where(casn.OrganizationID.EqColumn(carr.OrganizationID)).
		Where(casn.BusinessUnitID.EqColumn(carr.BusinessUnitID))
}

func tenderOfferActivityQuery(db bun.IDB) *bun.SelectQuery {
	carr := buncolgen.CarrierColumns
	tof := buncolgen.TenderOfferColumns
	return db.NewSelect().
		Model((*tender.TenderOffer)(nil)).
		Where(tof.CarrierID.EqColumn(carr.ID)).
		Where(tof.OrganizationID.EqColumn(carr.OrganizationID)).
		Where(tof.BusinessUnitID.EqColumn(carr.BusinessUnitID))
}

func buildCarrierSubjectsQuery(
	db bun.IDB,
	req *repositories.ListCarrierIntelSubjectsRequest,
	limit int,
) *bun.SelectQuery {
	cols := buncolgen.CarrierColumns
	casn := buncolgen.CarrierAssignmentColumns
	tof := buncolgen.TenderOfferColumns

	lastAssignment := assignmentActivityQuery(db).ColumnExpr(casn.CreatedAt.Expr("MAX({})"))
	lastOffer := tenderOfferActivityQuery(db).ColumnExpr(tof.CreatedAt.Expr("MAX({})"))

	q := db.NewSelect().
		Model((*carrier.Carrier)(nil)).
		Column(
			cols.ID.Bare(),
			cols.Name.Bare(),
			cols.DOTNumber.Bare(),
			cols.MCNumber.Bare(),
			cols.CarrierType.Bare(),
		).
		ColumnExpr(
			"GREATEST((?), (?)) AS ?",
			lastAssignment,
			lastOffer,
			bun.Ident(buncolgen.CarrierMonitoringEnrollmentColumns.LastUsedAt.Bare()),
		).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.CarrierScopeTenant(sq, req.TenantInfo).
				Where(cols.DOTNumber.IsNotNull())
		}).
		Order(cols.ID.OrderAsc()).
		Limit(limit)

	if req.ActiveOnly {
		q = q.Where(cols.Status.Eq(), carrier.StatusActive)
	}
	if len(req.CarrierIDs) > 0 {
		q = q.Where(cols.ID.In(), bun.List(req.CarrierIDs))
	}
	if !req.AfterID.IsNil() {
		q = q.Where(cols.ID.Gt(), req.AfterID)
	}
	if req.UsedSince != nil {
		since := *req.UsedSince
		recentAssignment := assignmentActivityQuery(db).
			ColumnExpr("1").
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return sq.
					Where(casn.CreatedAt.Gte(), since).
					WhereOr(casn.Status.In(), bun.List(openCarrierAssignmentStatuses))
			})
		q = q.WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			sq = sq.Where("EXISTS (?)", recentAssignment)
			if req.IncludeOpenTenders {
				openOffer := tenderOfferActivityQuery(db).
					ColumnExpr("1").
					Where(tof.Status.In(), bun.List(openTenderOfferStatuses))
				sq = sq.WhereOr("EXISTS (?)", openOffer)
			}
			return sq
		})
	}

	return q
}

func (r *subjectRepository) ListCarrierSubjects(
	ctx context.Context,
	req *repositories.ListCarrierIntelSubjectsRequest,
) ([]repositories.CarrierIntelSubject, error) {
	limit := intutils.Clamp(
		intutils.WithDefault(max(req.Limit, 0), defaultCarrierSubjectLimit),
		1,
		maxCarrierSubjectLimit,
	)

	rows := make([]carrierSubjectRow, 0, limit)
	if err := buildCarrierSubjectsQuery(r.db.DBForContext(ctx), req, limit).
		Scan(ctx, &rows); err != nil {
		r.l.Error("failed to list carrier intel carrier subjects", zap.Error(err))
		return nil, fmt.Errorf("list carrier intel carrier subjects: %w", err)
	}

	subjects := make([]repositories.CarrierIntelSubject, 0, len(rows))
	for i := range rows {
		row := &rows[i]
		subjects = append(subjects, repositories.CarrierIntelSubject{
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
}

func customerSubject(row *customerSubjectRow) repositories.CarrierIntelSubject {
	return repositories.CarrierIntelSubject{
		SubjectType:  carrierintel.SubjectTypeCustomer,
		SubjectID:    row.ID.String(),
		Name:         row.Name,
		DOTNumber:    row.DOTNumber,
		DocketNumber: row.MCNumber,
		Broker:       true,
	}
}

func customerSubjectColumns(q *bun.SelectQuery) *bun.SelectQuery {
	cols := buncolgen.CustomerColumns
	return q.Column(
		cols.ID.Bare(),
		cols.Name.Bare(),
		cols.DOTNumber.Bare(),
		cols.MCNumber.Bare(),
	)
}

func (r *subjectRepository) ListBrokerCustomerSubjects(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) ([]repositories.CarrierIntelSubject, error) {
	cols := buncolgen.CustomerColumns
	rows := make([]customerSubjectRow, 0, brokerCustomerCapacity)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model((*customer.Customer)(nil)).
		Apply(customerSubjectColumns).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.CustomerScopeTenant(sq, tenantInfo).
				Where(cols.BrokerVettingEnabled.IsTrue()).
				Where(cols.DOTNumber.IsNotNull())
		}).
		Order(cols.ID.OrderAsc()).
		Scan(ctx, &rows)
	if err != nil {
		r.l.Error("failed to list carrier intel broker customer subjects", zap.Error(err))
		return nil, fmt.Errorf("list carrier intel broker customer subjects: %w", err)
	}

	subjects := make([]repositories.CarrierIntelSubject, 0, len(rows))
	for i := range rows {
		subjects = append(subjects, customerSubject(&rows[i]))
	}

	return subjects, nil
}

func (r *subjectRepository) GetOrganizationSubject(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*repositories.CarrierIntelSubject, error) {
	cols := buncolgen.OrganizationColumns
	var (
		name      string
		dotNumber string
	)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model((*tenant.Organization)(nil)).
		Column(cols.Name.Bare(), cols.DOTNumber.Bare()).
		Where(cols.ID.Eq(), tenantInfo.OrgID).
		Where(cols.BusinessUnitID.Eq(), tenantInfo.BuID).
		Scan(ctx, &name, &dotNumber)
	if err != nil {
		if dberror.IsNotFoundError(err) {
			return nil, dberror.HandleNotFoundError(err, organizationEntityName)
		}
		r.l.Error("failed to get carrier intel organization subject", zap.Error(err))
		return nil, fmt.Errorf("get carrier intel organization subject: %w", err)
	}
	if dotNumber == "" {
		return nil, nil //nolint:nilnil // an organization without a DOT number cannot be monitored
	}

	return &repositories.CarrierIntelSubject{
		SubjectType: carrierintel.SubjectTypeOrganization,
		SubjectID:   tenantInfo.OrgID.String(),
		Name:        name,
		DOTNumber:   dotNumber,
	}, nil
}

func (r *subjectRepository) GetCustomerSubject(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	customerID pulid.ID,
) (*repositories.CarrierIntelSubject, error) {
	row := new(customerSubjectRow)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model((*customer.Customer)(nil)).
		Apply(customerSubjectColumns).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.CustomerScopeTenant(sq, tenantInfo).
				Where(buncolgen.CustomerColumns.ID.Eq(), customerID)
		}).
		Scan(ctx, row)
	if err != nil {
		if dberror.IsNotFoundError(err) {
			return nil, dberror.HandleNotFoundError(err, customerEntityName)
		}
		r.l.Error("failed to get carrier intel customer subject", zap.Error(err))
		return nil, fmt.Errorf("get carrier intel customer subject: %w", err)
	}

	subject := customerSubject(row)
	return &subject, nil
}

func (r *subjectRepository) ListExistingCarrierDOTs(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	dotNumbers []string,
) (map[string]pulid.ID, error) {
	if len(dotNumbers) == 0 {
		return map[string]pulid.ID{}, nil
	}

	cols := buncolgen.CarrierColumns
	rows := make([]carrierDOTRow, 0, len(dotNumbers))
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model((*carrier.Carrier)(nil)).
		Column(cols.ID.Bare(), cols.DOTNumber.Bare()).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.CarrierScopeTenant(sq, tenantInfo).
				Where(cols.DOTNumber.In(), bun.List(dotNumbers))
		}).
		Order(cols.CreatedAt.OrderAsc(), cols.ID.OrderAsc()).
		Scan(ctx, &rows)
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
}
