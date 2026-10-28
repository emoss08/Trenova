package ptopolicyrepository

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/pto"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker/workerdb"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workerports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

const defaultOpenAssignmentPageSize = 100

var assignmentErrors = dberror.Mapping{Entity: "WorkerPTOPolicyAssignment"}

type tenantRow struct {
	OrganizationID pulid.ID
	BusinessUnitID pulid.ID
}

func policyWithRules() keel.Loader {
	rl := workerdb.WorkerPTOPolicyAssignments.Policy().Rules()
	return rl.Order(rl.SortOrder.Asc(), rl.PTOType.Asc())
}

func policyLoads(include bool) []keel.Loader {
	return listquery.When(include, policyWithRules())
}

func (r *repository) ListAssignments(
	ctx context.Context,
	req *workerports.ListPTOAssignmentsRequest,
) ([]*pto.WorkerPTOPolicyAssignment, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*pto.WorkerPTOPolicyAssignment, error) {
			a := workerdb.WorkerPTOPolicyAssignments
			entities, err := a.Where(a.WorkerID.Eq(req.WorkerID)).
				Load(policyLoads(req.IncludePolicy)...).
				Order(a.EffectiveFrom.Desc()).
				All(ctx, r.db.Keel(ctx))
			if err != nil {
				r.l.Error("failed to list PTO assignments", zap.Error(err))
				return nil, err
			}

			return entities, nil
		},
	)
}

func (r *repository) GetAssignmentByID(
	ctx context.Context,
	req *workerports.GetPTOAssignmentByIDRequest,
) (*pto.WorkerPTOPolicyAssignment, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pto.WorkerPTOPolicyAssignment, error) {
			a := workerdb.WorkerPTOPolicyAssignments
			entity, err := a.Where(a.ID.Eq(req.ID)).
				Load(policyWithRules()).
				One(ctx, r.db.Keel(ctx))
			if err != nil {
				return nil, assignmentErrors.Map(err, req.ID.String())
			}

			return entity, nil
		},
	)
}

func activeAssignmentQuery(
	req *workerports.GetPTOAssignmentRequest,
) *keel.Query[*pto.WorkerPTOPolicyAssignment] {
	a := workerdb.WorkerPTOPolicyAssignments
	active := a.EffectiveTo.IsNull()
	if req.AsOf > 0 {
		active = keel.And(
			a.EffectiveFrom.Lte(req.AsOf),
			keel.Or(a.EffectiveTo.IsNull(), a.EffectiveTo.Gt(req.AsOf)),
		)
	}

	return a.Where(a.WorkerID.Eq(req.WorkerID), active).
		Load(policyLoads(req.IncludePolicy)...).
		Order(a.EffectiveFrom.Desc())
}

func (r *repository) GetActiveAssignment(
	ctx context.Context,
	req *workerports.GetPTOAssignmentRequest,
) (*pto.WorkerPTOPolicyAssignment, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pto.WorkerPTOPolicyAssignment, error) {
			entity, err := activeAssignmentQuery(req).First(ctx, r.db.Keel(ctx))
			if err != nil {
				return nil, assignmentErrors.Map(err, "")
			}

			return entity, nil
		},
	)
}

func (r *repository) CreateAssignment(
	ctx context.Context,
	entity *pto.WorkerPTOPolicyAssignment,
) (*pto.WorkerPTOPolicyAssignment, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*pto.WorkerPTOPolicyAssignment, error) {
			err := workerdb.WorkerPTOPolicyAssignments.Create(ctx, r.db.Keel(ctx), entity)
			if err != nil {
				r.l.Error("failed to create PTO assignment", zap.Error(err))
				return nil, assignmentErrors.Map(err, entity.ID.String())
			}

			return entity, nil
		},
	)
}

func (r *repository) UpdateAssignment(
	ctx context.Context,
	entity *pto.WorkerPTOPolicyAssignment,
) (*pto.WorkerPTOPolicyAssignment, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*pto.WorkerPTOPolicyAssignment, error) {
			err := workerdb.WorkerPTOPolicyAssignments.Save(ctx, r.db.Keel(ctx), entity)
			if err != nil {
				r.l.Error("failed to update PTO assignment", zap.Error(err))
				return nil, assignmentErrors.Map(err, entity.ID.String())
			}

			return entity, nil
		},
	)
}

func (r *repository) ListOpenAssignments(
	ctx context.Context,
	req *workerports.ListOpenPTOAssignmentsRequest,
) ([]*pto.WorkerPTOPolicyAssignment, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*pto.WorkerPTOPolicyAssignment, error) {
			limit := req.Limit
			if limit <= 0 {
				limit = defaultOpenAssignmentPageSize
			}

			a := workerdb.WorkerPTOPolicyAssignments
			entities, err := a.Where(
				a.EffectiveTo.IsNull(),
				keel.When(req.PolicyID.IsNotNil(), a.PTOPolicyID.Eq(req.PolicyID)),
				keel.When(req.WorkerID.IsNotNil(), a.WorkerID.Eq(req.WorkerID)),
				keel.When(req.AfterID.IsNotNil(), a.ID.Gt(req.AfterID)),
			).
				Load(policyLoads(req.IncludePolicy)...).
				Order(a.ID.Asc()).
				Limit(limit).
				List(ctx, r.db.Keel(ctx))
			if err != nil {
				r.l.Error("failed to list open PTO assignments", zap.Error(err))
				return nil, err
			}

			return entities, nil
		},
	)
}

func (r *repository) ListTenantsWithOpenAssignments(
	ctx context.Context,
) ([]pagination.TenantInfo, error) {
	ctx = dbscope.WithSystem(ctx, "list organizations with open PTO assignments to accrue")
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]pagination.TenantInfo, error) {
		a := workerdb.WorkerPTOPolicyAssignments
		rows, err := keel.Into[tenantRow](ctx, r.db.Keel(ctx), a.
			AllTenants().
			Group(a.OrganizationID, a.BusinessUnitID).
			Pick(a.OrganizationID, a.BusinessUnitID).
			Where(a.EffectiveTo.IsNull()))
		if err != nil {
			r.l.Error("failed to list tenants with open PTO assignments", zap.Error(err))
			return nil, err
		}

		tenants := make([]pagination.TenantInfo, 0, len(rows))
		for _, row := range rows {
			tenants = append(tenants, pagination.TenantInfo{
				OrgID: row.OrganizationID,
				BuID:  row.BusinessUnitID,
			})
		}

		return tenants, nil
	})
}
