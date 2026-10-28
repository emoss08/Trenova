package workerports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/portalinvitation"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type GetPortalInvitationByIDRequest struct {
	ID         pulid.ID              `json:"id"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
}

type GetPendingPortalInvitationRequest struct {
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
	WorkerID   pulid.ID              `json:"workerId"`
}

type ListPortalInvitationsRequest struct {
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
	WorkerID   pulid.ID              `json:"workerId"`
}

type ActivatePortalAccessRequest struct {
	Invitation      *portalinvitation.PortalInvitation `json:"invitation"`
	User            *tenant.User                       `json:"user"`
	RoleName        string                             `json:"roleName"`
	RoleDescription string                             `json:"roleDescription"`
}

type ListWorkerLoadsRequest struct {
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
	WorkerID   pulid.ID              `json:"workerId"`
	Statuses   []shipment.MoveStatus `json:"statuses"`
	Limit      int                   `json:"limit"`
}

type PortalAccessRepository interface {
	CreateInvitation(
		ctx context.Context,
		entity *portalinvitation.PortalInvitation,
	) (*portalinvitation.PortalInvitation, error)
	UpdateInvitation(
		ctx context.Context,
		entity *portalinvitation.PortalInvitation,
	) (*portalinvitation.PortalInvitation, error)
	GetInvitationByID(
		ctx context.Context,
		req GetPortalInvitationByIDRequest,
	) (*portalinvitation.PortalInvitation, error)
	GetInvitationByTokenHash(
		ctx context.Context,
		tokenHash string,
	) (*portalinvitation.PortalInvitation, error)
	GetPendingInvitationForWorker(
		ctx context.Context,
		req GetPendingPortalInvitationRequest,
	) (*portalinvitation.PortalInvitation, error)
	ListInvitations(
		ctx context.Context,
		req *ListPortalInvitationsRequest,
	) ([]*portalinvitation.PortalInvitation, error)
	GetWorkerByUserID(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
	) (*worker.Worker, error)
	ExistsWorkerForUser(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
	) (bool, error)
	GetWorkerForPortalManagement(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		workerID pulid.ID,
	) (*worker.Worker, error)
	ActivatePortalAccess(
		ctx context.Context,
		req *ActivatePortalAccessRequest,
	) (*tenant.User, error)
	RevokePortalAccess(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		workerID pulid.ID,
	) error
	ListWorkerLoads(
		ctx context.Context,
		req *ListWorkerLoadsRequest,
	) ([]*shipment.Assignment, error)
	ListWorkerPTO(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		workerID pulid.ID,
	) ([]*worker.WorkerPTO, error)
	UpdateAssignmentAck(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		workerID pulid.ID,
		assignmentID pulid.ID,
		ack shipment.AssignmentAck,
		reason string,
	) (*shipment.Assignment, error)
	WorkerAssignedToShipment(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		workerID pulid.ID,
		shipmentID pulid.ID,
	) (bool, error)
	WorkerAssignedToMove(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		workerID pulid.ID,
		moveID pulid.ID,
	) (bool, error)
	ListDriverShipmentComments(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		shipmentID pulid.ID,
	) ([]*shipment.ShipmentComment, error)
}
