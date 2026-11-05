package customerpaymentrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/billing/domain/customerpayment"
	"github.com/emoss08/trenova/internal/modules/billing/domain/customerpayment/customerpaymentdb"
	"github.com/emoss08/trenova/internal/modules/billing/ports/customerpaymentports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
)

func (r *repository) ListApplicationsByInvoiceIDs(
	ctx context.Context,
	req *customerpaymentports.ListApplicationsByInvoiceIDsRequest,
) (map[pulid.ID][]*customerpayment.Application, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (map[pulid.ID][]*customerpayment.Application, error) {
			result := make(map[pulid.ID][]*customerpayment.Application, len(req.InvoiceIDs))
			if len(req.InvoiceIDs) == 0 {
				return result, nil
			}

			cpa := customerpaymentdb.Applications
			rows, err := cpa.Where(cpa.InvoiceID.In(req.InvoiceIDs...)).
				Load(cpa.Payment()).
				Order(cpa.CreatedAt.Desc(), cpa.LineNumber.Asc()).
				All(ctx, r.db.Keel(ctx))
			if err != nil {
				return nil, fmt.Errorf("list customer payment applications by invoice: %w", err)
			}

			for _, row := range rows {
				result[row.InvoiceID] = append(result[row.InvoiceID], row)
			}

			return result, nil
		},
	)
}

func (r *repository) CreateCreditMemoApplications(
	ctx context.Context,
	applications []*customerpayment.CreditMemoApplication,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		if len(applications) == 0 {
			return nil
		}
		if err := customerpaymentdb.CreditMemoApplications.CreateMany(
			ctx,
			r.db.Keel(ctx),
			applications,
		); err != nil {
			return fmt.Errorf("create credit memo applications: %w", err)
		}

		return nil
	})
}

func (r *repository) UpdateCreditMemoApplication(
	ctx context.Context,
	application *customerpayment.CreditMemoApplication,
) (*customerpayment.CreditMemoApplication, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*customerpayment.CreditMemoApplication, error) {
			cma := customerpaymentdb.CreditMemoApplications
			changed, err := cma.Where(
				cma.ID.Eq(application.ID),
				cma.Status.Eq(customerpayment.CreditApplicationStatusApplied),
			).Update(
				ctx,
				r.db.Keel(ctx),
				cma.Status.To(application.Status),
				cma.UnappliedAt.To(*application.UnappliedAt),
				cma.UnappliedByID.To(application.UnappliedByID),
				cma.UnappliedReason.To(application.UnappliedReason),
			)
			if err != nil {
				return nil, creditMemoApplicationErrors.Map(err, application.ID.String())
			}
			if changed == 0 {
				return nil, errortypes.NewConflictError(
					"This credit memo application has already been unapplied",
				)
			}

			return application, nil
		},
	)
}

func (r *repository) GetCreditMemoApplicationByID(
	ctx context.Context,
	req customerpaymentports.GetCreditMemoApplicationRequest,
) (*customerpayment.CreditMemoApplication, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*customerpayment.CreditMemoApplication, error) {
			entity, err := customerpaymentdb.CreditMemoApplications.Get(ctx, r.db.Keel(ctx), req.ID)
			if err != nil {
				return nil, creditMemoApplicationErrors.Map(err, req.ID.String())
			}

			return entity, nil
		},
	)
}

func (r *repository) ListCreditMemoApplicationsByInvoiceIDs(
	ctx context.Context,
	req *customerpaymentports.ListApplicationsByInvoiceIDsRequest,
) (map[pulid.ID][]*customerpayment.CreditMemoApplication, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (map[pulid.ID][]*customerpayment.CreditMemoApplication, error) {
			result := make(
				map[pulid.ID][]*customerpayment.CreditMemoApplication,
				len(req.InvoiceIDs),
			)
			if len(req.InvoiceIDs) == 0 {
				return result, nil
			}

			cma := customerpaymentdb.CreditMemoApplications
			rows, err := cma.Where(keel.Or(
				cma.InvoiceID.In(req.InvoiceIDs...),
				cma.CreditMemoInvoiceID.In(req.InvoiceIDs...),
			)).
				Load(cma.CreditMemo(), cma.Invoice()).
				Order(cma.CreatedAt.Desc(), cma.LineNumber.Asc()).
				All(ctx, r.db.Keel(ctx))
			if err != nil {
				return nil, fmt.Errorf("list credit memo applications by invoice: %w", err)
			}

			for _, row := range rows {
				result[row.InvoiceID] = append(result[row.InvoiceID], row)
				if row.CreditMemoInvoiceID != row.InvoiceID {
					result[row.CreditMemoInvoiceID] = append(result[row.CreditMemoInvoiceID], row)
				}
			}

			return result, nil
		},
	)
}
