package tenantbootstrap

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/accountingseed"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/referenceseed"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/tenantscope"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant/tenantdb"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/emoss08/trenova/shared/tokenutils"
)

const (
	maxLoginSlugLength    = 100
	maxBucketNameLength   = 63
	maxUsernameLength     = 20
	businessUnitCodeBytes = 4
	businessUnitCodeHead  = "CL"
	maxCandidateAttempts  = 50
)

var (
	ErrBusinessUnitRequired = errors.New("tenant bootstrap: business unit is required")
	ErrOrganizationRequired = errors.New("tenant bootstrap: organization is required")
	ErrOwnerRequired        = errors.New("tenant bootstrap: owner user is required")
	ErrNoCandidate          = errors.New("tenant bootstrap: no free identifier was found")
)

type BootstrapParams struct {
	BusinessUnit *tenant.BusinessUnit
	Organization *tenant.Organization
	Owner        *tenant.User
	Now          int64
}

type BootstrapResult struct {
	BusinessUnit *tenant.BusinessUnit
	Organization *tenant.Organization
	Owner        *tenant.User
	Membership   *tenant.OrganizationMembership
	AdminRole    *permission.Role
}

func Bootstrap(ctx context.Context, db *keel.DB, params BootstrapParams) (*BootstrapResult, error) {
	if err := params.validate(); err != nil {
		return nil, err
	}

	now := params.Now
	if now <= 0 {
		now = timeutils.NowUnix()
	}

	bu, org, err := createTenant(ctx, db, params, now)
	if err != nil {
		return nil, err
	}

	scope := tenantscope.Scope{
		OrganizationID: org.ID,
		BusinessUnitID: bu.ID,
		Now:            now,
	}

	if err = CreateControls(ctx, db, scope); err != nil {
		return nil, err
	}
	if err = CreateSequences(ctx, db, scope); err != nil {
		return nil, err
	}

	owner, membership, err := createOwner(ctx, db, params, scope)
	if err != nil {
		return nil, err
	}

	role, err := CreateAdminRole(ctx, db, AdminRoleParams{Scope: scope, CreatedBy: owner.ID})
	if err != nil {
		return nil, err
	}
	if err = AssignRole(ctx, db, RoleAssignmentParams{
		UserID:         owner.ID,
		OrganizationID: org.ID,
		RoleID:         role.ID,
		AssignedBy:     owner.ID,
		AssignedAt:     now,
	}); err != nil {
		return nil, err
	}

	if err = CreateReferenceData(ctx, db, scope); err != nil {
		return nil, err
	}

	return &BootstrapResult{
		BusinessUnit: bu,
		Organization: org,
		Owner:        owner,
		Membership:   membership,
		AdminRole:    role,
	}, nil
}

func (p BootstrapParams) validate() error {
	switch {
	case p.BusinessUnit == nil:
		return ErrBusinessUnitRequired
	case p.Organization == nil:
		return ErrOrganizationRequired
	case p.Owner == nil:
		return ErrOwnerRequired
	default:
		return nil
	}
}

func createTenant(
	ctx context.Context,
	db *keel.DB,
	params BootstrapParams,
	now int64,
) (*tenant.BusinessUnit, *tenant.Organization, error) {
	bu := params.BusinessUnit
	if bu.CreatedAt == 0 {
		bu.CreatedAt = now
	}
	bu.UpdatedAt = now
	if err := tenantdb.BusinessUnits.Create(ctx, db, bu); err != nil {
		return nil, nil, fmt.Errorf("create business unit: %w", err)
	}

	org := params.Organization
	org.BusinessUnitID = bu.ID
	org.UpdatedAt = now
	if err := tenantdb.Organizations.Create(ctx, db, org); err != nil {
		return nil, nil, fmt.Errorf("create organization: %w", err)
	}

	return bu, org, nil
}

func createOwner(
	ctx context.Context,
	db *keel.DB,
	params BootstrapParams,
	scope tenantscope.Scope,
) (*tenant.User, *tenant.OrganizationMembership, error) {
	owner := params.Owner
	owner.CurrentOrganizationID = scope.OrganizationID
	owner.BusinessUnitID = scope.BusinessUnitID
	owner.UpdatedAt = scope.Now
	if err := tenantdb.Users.Create(ctx, db, owner); err != nil {
		return nil, nil, fmt.Errorf("create owner user: %w", err)
	}

	membership := &tenant.OrganizationMembership{
		BusinessUnitID: scope.BusinessUnitID,
		UserID:         owner.ID,
		OrganizationID: scope.OrganizationID,
		JoinedAt:       scope.Now,
		GrantedByID:    owner.ID,
		IsDefault:      true,
	}
	if err := tenantdb.OrganizationMemberships.Create(ctx, db, membership); err != nil {
		return nil, nil, fmt.Errorf("create owner membership: %w", err)
	}

	return owner, membership, nil
}

func CreateReferenceData(ctx context.Context, db *keel.DB, scope tenantscope.Scope) error {
	if _, err := accountingseed.EnsureChartOfAccounts(ctx, db, scope); err != nil {
		return err
	}
	if _, err := referenceseed.CreateDocumentTypes(ctx, db, scope); err != nil {
		return fmt.Errorf("create system document types: %w", err)
	}
	if _, err := referenceseed.CreateServiceFailureReasonCodes(ctx, db, scope); err != nil {
		return fmt.Errorf("create service failure reason codes: %w", err)
	}
	if _, err := referenceseed.CreateDocumentTemplateStarters(ctx, db, scope); err != nil {
		return fmt.Errorf("create document template starters: %w", err)
	}
	if _, err := referenceseed.CreateSystemAgentDefinitions(ctx, db, scope); err != nil {
		return fmt.Errorf("create system agent definitions: %w", err)
	}
	if _, err := referenceseed.CreateTCAAllowlist(ctx, db, scope); err != nil {
		return fmt.Errorf("create table change alert allowlist: %w", err)
	}

	return nil
}

func AvailableLoginSlug(ctx context.Context, db *keel.DB, base string) (string, error) {
	base = strings.Trim(base, "-")
	if base == "" {
		base = "workspace"
	}

	return firstAvailable(ctx, base, maxLoginSlugLength, func(candidate string) (bool, error) {
		o := tenantdb.Organizations
		return tenantscope.Scoped(ctx, o.Where(o.LoginSlug.Eq(candidate))).Exists(ctx, db)
	})
}

func AvailableUsername(ctx context.Context, db *keel.DB, base string) (string, error) {
	base = strings.Trim(base, "-._")
	if base == "" {
		base = "owner"
	}

	return firstAvailable(ctx, base, maxUsernameLength, func(candidate string) (bool, error) {
		u := tenantdb.Users
		return tenantscope.Scoped(
			ctx,
			u.Where(u.Username.Lower().Eq(strings.ToLower(candidate))),
		).Exists(ctx, db)
	})
}

func AvailableBusinessUnitCode(ctx context.Context, db *keel.DB) (string, error) {
	for range maxCandidateAttempts {
		suffix, err := tokenutils.RandomHex(businessUnitCodeBytes)
		if err != nil {
			return "", err
		}
		code := businessUnitCodeHead + strings.ToUpper(suffix)

		b := tenantdb.BusinessUnits
		taken, err := b.Where(b.Code.Lower().Eq(strings.ToLower(code))).Exists(ctx, db)
		if err != nil {
			return "", fmt.Errorf("check business unit code: %w", err)
		}
		if !taken {
			return code, nil
		}
	}

	return "", ErrNoCandidate
}

func BucketNameFor(loginSlug string) string {
	name := strings.Trim(loginSlug, "-")
	if len(name) > maxBucketNameLength {
		name = strings.TrimRight(name[:maxBucketNameLength], "-")
	}
	if len(name) < 3 {
		name = "tenant-" + name
	}

	return name
}

func firstAvailable(
	ctx context.Context,
	base string,
	maxLength int,
	taken func(candidate string) (bool, error),
) (string, error) {
	for attempt := range maxCandidateAttempts {
		if err := ctx.Err(); err != nil {
			return "", err
		}

		candidate, err := candidateFor(base, maxLength, attempt)
		if err != nil {
			return "", err
		}

		exists, err := taken(candidate)
		if err != nil {
			return "", fmt.Errorf("check identifier %q: %w", candidate, err)
		}
		if !exists {
			return candidate, nil
		}
	}

	return "", ErrNoCandidate
}

func candidateFor(base string, maxLength, attempt int) (string, error) {
	var suffix string
	switch {
	case attempt == 0:
		suffix = ""
	case attempt < 10:
		suffix = "-" + strconv.Itoa(attempt+1)
	default:
		random, err := tokenutils.RandomHex(3)
		if err != nil {
			return "", err
		}
		suffix = "-" + random
	}

	head := base
	if len(head)+len(suffix) > maxLength {
		head = strings.TrimRight(head[:maxLength-len(suffix)], "-._")
	}

	return head + suffix, nil
}
