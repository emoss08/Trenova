package supportaccessports

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/modules/identity/domain/supportaccess"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type CreateGrantRequest struct {
	Grant     *supportaccess.Grant
	Principal *tenant.User
	Now       int64
}

type ListGrantsRequest struct {
	Filter *pagination.QueryOptions
	Cursor pagination.CursorInfo
}

type FindLiveGrantRequest struct {
	OrganizationID pulid.ID
	StaffEmail     string
	Now            int64
}

const MaxLiveGrantsPerStaff = 50

type ListLiveForStaffRequest struct {
	StaffEmail string
	Now        int64
	Limit      int
}

const MaxReportedGrants = 100

type ListLiveRequest struct {
	Now             int64
	Limit           int
	OrganizationIDs []pulid.ID
}

type LiveGrant struct {
	Grant            *supportaccess.Grant
	OrganizationName string
}

type MarkEnteredRequest struct {
	Grant     *supportaccess.Grant
	StaffName string
	At        int64
}

// GrantRepository stores support access grants. Every method but FindLiveForEntry and the
// expiry sweep's two runs under the caller's tenant scope.
type GrantRepository interface {
	// Create stores the grant and its support principal in one transaction, serialized per
	// organization: it refuses a grant past supportaccess.MaxLiveGrants live grants or for an
	// email that already holds a live grant there.
	Create(ctx context.Context, req *CreateGrantRequest) (*supportaccess.Grant, error)
	Get(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		id pulid.ID,
	) (*supportaccess.Grant, error)
	List(
		ctx context.Context,
		req *ListGrantsRequest,
	) (*pagination.CursorListResult[*supportaccess.Grant], error)
	// Revoke writes the revocation only while the grant is still live and returns
	// supportaccess.ErrGrantNotFound when it is not, so two revocations never both succeed.
	Revoke(ctx context.Context, grant *supportaccess.Grant, now int64) (*supportaccess.Grant, error)
	Extend(ctx context.Context, grant *supportaccess.Grant, now int64) (*supportaccess.Grant, error)
	// ByPrincipal reads the grant whose support principal is userID in the scope's tenant.
	ByPrincipal(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		userID pulid.ID,
	) (*supportaccess.Grant, error)
	// FindLiveForEntry is the one lookup before a tenant is known: the live grant for an email
	// in an organization, read under the system scope.
	FindLiveForEntry(ctx context.Context, req *FindLiveGrantRequest) (*supportaccess.Grant, error)
	ListLiveForStaff(
		ctx context.Context,
		req *ListLiveForStaffRequest,
	) ([]*supportaccess.Grant, error)
	ListLive(ctx context.Context, req *ListLiveRequest) ([]*LiveGrant, error)
	// OrganizationBusinessUnit reads, under the system scope, the business unit of the
	// organization a verified assertion names, so a refused entry is recorded in that
	// organization's audit log.
	OrganizationBusinessUnit(ctx context.Context, orgID pulid.ID) (pulid.ID, error)
	MarkEntered(ctx context.Context, req *MarkEnteredRequest) error
	ListUnrecordedExpiries(
		ctx context.Context,
		now int64,
		limit int,
	) ([]*supportaccess.Grant, error)
	MarkExpiryRecorded(ctx context.Context, grant *supportaccess.Grant, at int64) (bool, error)
}

// KeySetRepository holds the one row of keys the control plane delivered.
type KeySetRepository interface {
	Get(ctx context.Context) (*supportaccess.KeySet, error)
	Save(ctx context.Context, keys *supportaccess.KeySet) error
	Delete(ctx context.Context) error
}

var ErrTenantNotHeld = errors.New("no tenant of this shared server has that instance id")

type SupportTenant struct {
	OrganizationID pulid.ID
	BusinessUnitID pulid.ID
	Active         bool
}

type Tenants interface {
	Tenant(ctx context.Context, instanceID string) (*SupportTenant, error)
}

// AssertionLedger records accepted assertion ids. Use reports false when the id was already
// used, decided by the database so two uses of one assertion at once cannot both succeed.
type AssertionLedger interface {
	Use(ctx context.Context, use *supportaccess.AssertionUse) (bool, error)
}

type NoticeKind string

const (
	NoticeGranted    NoticeKind = "granted"
	NoticeExtended   NoticeKind = "extended"
	NoticeFirstEntry NoticeKind = "first_entry"
	NoticeRevoked    NoticeKind = "revoked"
	NoticeExpired    NoticeKind = "expired"
)

type Notice struct {
	Kind       NoticeKind
	TenantInfo pagination.TenantInfo
	ActorID    pulid.ID
	Grant      *supportaccess.Grant
	StaffName  string
}

type Notifier interface {
	SupportAccessChanged(ctx context.Context, notice *Notice) error
}

// Principals answers for the support principal: the permission engine asks it for the
// permissions of a support principal, and sign-in asks it whether a support session's grant is
// still live.
type Principals interface {
	Permissions(
		ctx context.Context,
		userID, orgID pulid.ID,
	) (*permissionports.CachedPermissions, error)
	RequireLive(ctx context.Context, req *RequireLiveRequest) error
}

type ModelCalls interface {
	AdmitModelCall(ctx context.Context, tenant pagination.TenantInfo) error
}

type RequireLiveRequest struct {
	OrganizationID  pulid.ID
	BusinessUnitID  pulid.ID
	GrantID         pulid.ID
	PrincipalUserID pulid.ID
	Now             int64
}
