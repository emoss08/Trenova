package agentextensionrepository

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/agent/domain/extensionwebwatch"
	"github.com/emoss08/trenova/internal/modules/agent/domain/extensionwebwatch/extensionwebwatchdb"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentextensionports"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

const claimWebWatchChecksReason = "claim the web watches in every tenant whose next check is due"

// ScheduleCheck makes the watch due for a check by at, never later than it
// already was: a vendor's report asks for one now, a failed check for one
// soon, and neither puts off one already sooner.
func (r *webWatchRepository) ScheduleCheck(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	id pulid.ID,
	at int64,
) error {
	ctx = dbscope.WithTenant(ctx, tenantInfo.DBTenant())
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		w := extensionwebwatchdb.WebWatches
		_, err := w.Where(w.ID.Eq(id)).Update(
			ctx,
			r.db.Keel(ctx),
			w.NextCheckAt.ToExpr(w.NextCheckAt.Least(keel.Int64(at))),
			w.Version.ToExpr(w.Version),
			w.UpdatedAt.ToExpr(w.UpdatedAt),
		)

		return err
	})
}

// ClearCheck takes a watch a check found ended off the sweep.
func (r *webWatchRepository) ClearCheck(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	id pulid.ID,
) error {
	ctx = dbscope.WithTenant(ctx, tenantInfo.DBTenant())
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		w := extensionwebwatchdb.WebWatches
		_, err := w.Where(w.ID.Eq(id)).Update(
			ctx,
			r.db.Keel(ctx),
			w.NextCheckAt.ToNull(),
			w.Version.ToExpr(w.Version),
			w.UpdatedAt.ToExpr(w.UpdatedAt),
		)

		return err
	})
}

// ClaimDueChecks takes, across every tenant, the watches whose check is due
// and moves each one's next check to req.Next in the same statement, so a
// watch is checked once per claim and a check that never reports back is
// simply tried at the next one.
func (r *webWatchRepository) ClaimDueChecks(
	ctx context.Context,
	req *agentextensionports.ClaimWebWatchChecksRequest,
) ([]agentextensionports.WebWatchRef, error) {
	ctx = dbscope.WithSystem(ctx, claimWebWatchChecksReason)
	return dbtx.Write(ctx, r.db, func(ctx context.Context) ([]agentextensionports.WebWatchRef, error) {
		w := extensionwebwatchdb.WebWatches
		due := w.Pick(w.ID).
			Where(w.NextCheckAt.Lte(req.Now)).
			Order(w.NextCheckAt.Asc(), w.ID.Asc()).
			Limit(req.Limit).
			AllTenants()

		rows, err := w.Pick(w.ID, w.OrganizationID, w.BusinessUnitID).
			Where(w.ID.InQuery(due), w.NextCheckAt.Lte(req.Now)).
			AllTenants().
			UpdateReturning(
				ctx,
				r.db.Keel(ctx),
				w.NextCheckAt.To(req.Next),
				w.Version.ToExpr(w.Version),
				w.UpdatedAt.ToExpr(w.UpdatedAt),
			)
		if err != nil {
			return nil, err
		}

		refs := make([]agentextensionports.WebWatchRef, 0, len(rows))
		for _, row := range rows {
			refs = append(refs, refOf(row))
		}

		return refs, nil
	})
}

func refOf(watch *extensionwebwatch.WebWatch) agentextensionports.WebWatchRef {
	return agentextensionports.WebWatchRef{
		ID:     watch.ID,
		Tenant: pagination.TenantInfo{OrgID: watch.OrganizationID, BuID: watch.BusinessUnitID},
	}
}
