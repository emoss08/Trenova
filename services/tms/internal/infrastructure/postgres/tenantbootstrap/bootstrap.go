package tenantbootstrap

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/emoss08/trenova/shared/tokenutils"
	"github.com/uptrace/bun"
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
	Record       Recorder
}

type BootstrapResult struct {
	BusinessUnit *tenant.BusinessUnit
	Organization *tenant.Organization
	Owner        *tenant.User
	Membership   *tenant.OrganizationMembership
	AdminRole    *permission.Role
}

func Bootstrap(ctx context.Context, db bun.IDB, params BootstrapParams) (*BootstrapResult, error) {
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

	scope := Scope{
		OrganizationID: org.ID,
		BusinessUnitID: bu.ID,
		Now:            now,
		Record:         params.Record,
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
		Record:         params.Record,
	}); err != nil {
		return nil, err
	}

	if err = createReferenceData(ctx, db, scope); err != nil {
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
	db bun.IDB,
	params BootstrapParams,
	now int64,
) (*tenant.BusinessUnit, *tenant.Organization, error) {
	bu := params.BusinessUnit
	if bu.CreatedAt == 0 {
		bu.CreatedAt = now
	}
	bu.UpdatedAt = now
	if _, err := db.NewInsert().Model(bu).Exec(ctx); err != nil {
		return nil, nil, fmt.Errorf("create business unit: %w", err)
	}
	if params.Record != nil {
		if err := params.Record(ctx, "business_units", bu.ID); err != nil {
			return nil, nil, err
		}
	}

	org := params.Organization
	org.BusinessUnitID = bu.ID
	org.UpdatedAt = now
	if _, err := db.NewInsert().Model(org).Exec(ctx); err != nil {
		return nil, nil, fmt.Errorf("create organization: %w", err)
	}
	if params.Record != nil {
		if err := params.Record(ctx, "organizations", org.ID); err != nil {
			return nil, nil, err
		}
	}

	return bu, org, nil
}

func createOwner(
	ctx context.Context,
	db bun.IDB,
	params BootstrapParams,
	scope Scope,
) (*tenant.User, *tenant.OrganizationMembership, error) {
	owner := params.Owner
	owner.CurrentOrganizationID = scope.OrganizationID
	owner.BusinessUnitID = scope.BusinessUnitID
	owner.UpdatedAt = scope.Now
	if _, err := db.NewInsert().Model(owner).Exec(ctx); err != nil {
		return nil, nil, fmt.Errorf("create owner user: %w", err)
	}
	if err := scope.record(ctx, "users", owner.ID); err != nil {
		return nil, nil, err
	}

	membership := &tenant.OrganizationMembership{
		BusinessUnitID: scope.BusinessUnitID,
		UserID:         owner.ID,
		OrganizationID: scope.OrganizationID,
		JoinedAt:       scope.Now,
		GrantedByID:    owner.ID,
		IsDefault:      true,
	}
	if _, err := db.NewInsert().Model(membership).Exec(ctx); err != nil {
		return nil, nil, fmt.Errorf("create owner membership: %w", err)
	}
	if err := scope.record(ctx, "organization_memberships", membership.ID); err != nil {
		return nil, nil, err
	}

	return owner, membership, nil
}

func createReferenceData(ctx context.Context, db bun.IDB, scope Scope) error {
	if _, err := CreateChartOfAccounts(ctx, db, scope); err != nil {
		return err
	}
	if _, err := CreateDocumentTypes(ctx, db, scope); err != nil {
		return fmt.Errorf("create system document types: %w", err)
	}
	if _, err := CreateServiceFailureReasonCodes(ctx, db, scope); err != nil {
		return fmt.Errorf("create service failure reason codes: %w", err)
	}
	if _, err := CreateDocumentTemplateStarters(ctx, db, scope); err != nil {
		return fmt.Errorf("create document template starters: %w", err)
	}
	if _, err := CreateSystemAgentDefinitions(ctx, db, scope); err != nil {
		return fmt.Errorf("create system agent definitions: %w", err)
	}

	return nil
}

func AvailableLoginSlug(ctx context.Context, db bun.IDB, base string) (string, error) {
	base = strings.Trim(base, "-")
	if base == "" {
		base = "workspace"
	}

	return firstAvailable(ctx, base, maxLoginSlugLength, func(candidate string) (bool, error) {
		cols := buncolgen.OrganizationColumns
		return db.NewSelect().
			Model((*tenant.Organization)(nil)).
			Where(cols.LoginSlug.Eq(), candidate).
			Exists(ctx)
	})
}

func AvailableUsername(ctx context.Context, db bun.IDB, base string) (string, error) {
	base = strings.Trim(base, "-._")
	if base == "" {
		base = "owner"
	}

	return firstAvailable(ctx, base, maxUsernameLength, func(candidate string) (bool, error) {
		cols := buncolgen.UserColumns
		return db.NewSelect().
			Model((*tenant.User)(nil)).
			Where("lower("+cols.Username.Qualified()+") = ?", strings.ToLower(candidate)).
			Exists(ctx)
	})
}

func AvailableBusinessUnitCode(ctx context.Context, db bun.IDB) (string, error) {
	for range maxCandidateAttempts {
		suffix, err := tokenutils.RandomHex(businessUnitCodeBytes)
		if err != nil {
			return "", err
		}
		code := businessUnitCodeHead + strings.ToUpper(suffix)

		cols := buncolgen.BusinessUnitColumns
		taken, err := db.NewSelect().
			Model((*tenant.BusinessUnit)(nil)).
			Where("lower("+cols.Code.Qualified()+") = ?", strings.ToLower(code)).
			Exists(ctx)
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
