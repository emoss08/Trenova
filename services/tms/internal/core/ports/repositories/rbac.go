package repositories

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/shared/pulid"
)

var ErrCircularRoleHierarchy = errors.New("role hierarchy would create a cycle")

type ConstraintViolation struct {
	ConstraintID   pulid.ID                      `json:"constraintId"`
	ConstraintName string                        `json:"constraintName"`
	Type           permission.RoleConstraintType `json:"type"`
	MaxRoles       int                           `json:"maxRoles"`
	RoleIDs        []pulid.ID                    `json:"roleIds"`
	MatchedRoleIDs []pulid.ID                    `json:"matchedRoleIds"`
}

type RBACPreflightReport struct {
	HierarchyCycles []RoleHierarchyCycle  `json:"hierarchyCycles"`
	SSDViolations   []ConstraintViolation `json:"ssdViolations"`
	DSDViolations   []ConstraintViolation `json:"dsdViolations"`
}

type RoleHierarchyCycle struct {
	RoleIDs []pulid.ID `json:"roleIds"`
}

type UpsertRoleHierarchyEdgeRequest struct {
	ActorID        pulid.ID
	OrganizationID pulid.ID
	BusinessUnitID pulid.ID
	SeniorRoleID   pulid.ID
	JuniorRoleID   pulid.ID
}

type DeleteRoleHierarchyEdgeRequest struct {
	OrganizationID pulid.ID
	EdgeID         pulid.ID
}

type ListRoleConstraintsRequest struct {
	OrganizationID pulid.ID
	Type           permission.RoleConstraintType
}

type SaveRoleConstraintRequest struct {
	Constraint *permission.RoleConstraint
	RoleIDs    []pulid.ID
}

type RBACRepository interface {
	ListRoleHierarchyEdges(
		ctx context.Context,
		orgID pulid.ID,
	) ([]*permission.RoleHierarchyEdge, error)
	UpsertRoleHierarchyEdge(ctx context.Context, req *UpsertRoleHierarchyEdgeRequest) error
	DeleteRoleHierarchyEdge(ctx context.Context, req DeleteRoleHierarchyEdgeRequest) error
	GetRoleClosure(ctx context.Context, roleIDs []pulid.ID) ([]pulid.ID, error)
	GetAuthorizedRoles(ctx context.Context, userID, orgID pulid.ID) ([]*permission.Role, error)
	ListRoleConstraints(
		ctx context.Context,
		req ListRoleConstraintsRequest,
	) ([]*permission.RoleConstraint, error)
	GetRoleConstraint(
		ctx context.Context,
		orgID, constraintID pulid.ID,
	) (*permission.RoleConstraint, error)
	SaveRoleConstraint(ctx context.Context, req *SaveRoleConstraintRequest) error
	DeleteRoleConstraint(ctx context.Context, orgID, constraintID pulid.ID) error
	ValidateStaticSeparationOfDuty(
		ctx context.Context,
		userID, orgID pulid.ID,
		roleIDs []pulid.ID,
	) ([]ConstraintViolation, error)
	ValidateDynamicSeparationOfDuty(
		ctx context.Context,
		orgID pulid.ID,
		roleIDs []pulid.ID,
	) ([]ConstraintViolation, error)
	RunPreflight(ctx context.Context, orgID pulid.ID) (*RBACPreflightReport, error)
}
