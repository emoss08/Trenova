package rbactest

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/shared/pulid"
)

type Repository struct {
	AuthorizedRoles       []*permission.Role
	AuthorizedRolesErr    error
	StaticViolations      []repositories.ConstraintViolation
	StaticErr             error
	DynamicViolations     []repositories.ConstraintViolation
	DynamicErr            error
	RoleHierarchyEdges    []*permission.RoleHierarchyEdge
	RoleHierarchyEdgesErr error
	RoleClosure           []pulid.ID
	RoleClosureErr        error
	RoleConstraints       []*permission.RoleConstraint
	RoleConstraintsErr    error
	RoleConstraint        *permission.RoleConstraint
	RoleConstraintErr     error
	SaveConstraintErr     error
	DeleteConstraintErr   error
	UpsertHierarchyErr    error
	DeleteHierarchyErr    error
	PreflightReport       *repositories.RBACPreflightReport
	PreflightErr          error
}

func (r *Repository) ListRoleHierarchyEdges(
	context.Context,
	pulid.ID,
) ([]*permission.RoleHierarchyEdge, error) {
	if r.RoleHierarchyEdgesErr != nil {
		return nil, r.RoleHierarchyEdgesErr
	}
	if r.RoleHierarchyEdges == nil {
		return []*permission.RoleHierarchyEdge{}, nil
	}
	return r.RoleHierarchyEdges, nil
}

func (r *Repository) UpsertRoleHierarchyEdge(
	context.Context,
	*repositories.UpsertRoleHierarchyEdgeRequest,
) error {
	return r.UpsertHierarchyErr
}

func (r *Repository) DeleteRoleHierarchyEdge(
	context.Context,
	repositories.DeleteRoleHierarchyEdgeRequest,
) error {
	return r.DeleteHierarchyErr
}

func (r *Repository) GetRoleClosure(context.Context, []pulid.ID) ([]pulid.ID, error) {
	if r.RoleClosureErr != nil {
		return nil, r.RoleClosureErr
	}
	if r.RoleClosure == nil {
		return []pulid.ID{}, nil
	}
	return r.RoleClosure, nil
}

func (r *Repository) GetAuthorizedRoles(
	context.Context,
	pulid.ID,
	pulid.ID,
) ([]*permission.Role, error) {
	if r.AuthorizedRolesErr != nil {
		return nil, r.AuthorizedRolesErr
	}
	if r.AuthorizedRoles == nil {
		return []*permission.Role{}, nil
	}
	return r.AuthorizedRoles, nil
}

func (r *Repository) ListRoleConstraints(
	context.Context,
	repositories.ListRoleConstraintsRequest,
) ([]*permission.RoleConstraint, error) {
	if r.RoleConstraintsErr != nil {
		return nil, r.RoleConstraintsErr
	}
	if r.RoleConstraints == nil {
		return []*permission.RoleConstraint{}, nil
	}
	return r.RoleConstraints, nil
}

func (r *Repository) GetRoleConstraint(
	context.Context,
	pulid.ID,
	pulid.ID,
) (*permission.RoleConstraint, error) {
	if r.RoleConstraintErr != nil {
		return nil, r.RoleConstraintErr
	}
	if r.RoleConstraint == nil {
		return &permission.RoleConstraint{}, nil
	}
	return r.RoleConstraint, nil
}

func (r *Repository) SaveRoleConstraint(
	context.Context,
	*repositories.SaveRoleConstraintRequest,
) error {
	return r.SaveConstraintErr
}

func (r *Repository) DeleteRoleConstraint(context.Context, pulid.ID, pulid.ID) error {
	return r.DeleteConstraintErr
}

func (r *Repository) ValidateStaticSeparationOfDuty(
	context.Context,
	pulid.ID,
	pulid.ID,
	[]pulid.ID,
) ([]repositories.ConstraintViolation, error) {
	if r.StaticErr != nil {
		return nil, r.StaticErr
	}
	if r.StaticViolations == nil {
		return []repositories.ConstraintViolation{}, nil
	}
	return r.StaticViolations, nil
}

func (r *Repository) ValidateDynamicSeparationOfDuty(
	context.Context,
	pulid.ID,
	[]pulid.ID,
) ([]repositories.ConstraintViolation, error) {
	if r.DynamicErr != nil {
		return nil, r.DynamicErr
	}
	if r.DynamicViolations == nil {
		return []repositories.ConstraintViolation{}, nil
	}
	return r.DynamicViolations, nil
}

func (r *Repository) RunPreflight(
	context.Context,
	pulid.ID,
) (*repositories.RBACPreflightReport, error) {
	if r.PreflightErr != nil {
		return nil, r.PreflightErr
	}
	if r.PreflightReport == nil {
		return &repositories.RBACPreflightReport{}, nil
	}
	return r.PreflightReport, nil
}
