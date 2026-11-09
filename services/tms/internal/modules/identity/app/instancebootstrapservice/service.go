package instancebootstrapservice

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/config/systemconfig"
	"github.com/emoss08/trenova/internal/modules/identity/domain/instancebootstrap"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/ports/instancebootstrapports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	bootstrapScopeReason = "create the first organization and administrator of an instance before any tenant exists"
	lockTimeout          = 10 * time.Second
	defaultLocale        = "en"
	loginSlugBaseLength  = 60
	auditSource          = "instance_bootstrap"
	fieldPassword        = "password"
	fieldAdminSetup      = "adminSetup"
	keyUserID            = "userId"
	keyRoleID            = "roleId"
	keyOrganizationID    = "organizationId"
)

var (
	ErrAlreadyInitialized = errors.New("instance already has users")
	ErrInputsDiffer       = errors.New("instance was bootstrapped with different inputs")
)

type Params struct {
	fx.In

	DB           kernelports.Transactor
	Tenants      tenantports.TenantBootstrapRepository
	Repo         instancebootstrapports.InstanceBootstrapRepository
	Auditor      permissionports.SecurityAuditor
	SystemConfig *systemconfig.Config
	Logger       *zap.Logger
}

type Service struct {
	db                 kernelports.Transactor
	tenants            tenantports.TenantBootstrapRepository
	repo               instancebootstrapports.InstanceBootstrapRepository
	auditor            permissionports.SecurityAuditor
	systemUserPassword string
	now                func() int64
	l                  *zap.Logger
}

//nolint:gocritic // this is dependency injection
func New(p Params) *Service {
	var systemUserPassword string
	systemUserPassword = p.SystemConfig.SystemUserPassword

	return &Service{
		db:                 p.DB,
		tenants:            p.Tenants,
		repo:               p.Repo,
		auditor:            p.Auditor,
		systemUserPassword: systemUserPassword,
		now:                timeutils.NowUnix,
		l:                  p.Logger.Named("service.instance-bootstrap"),
	}
}

var _ instancebootstrapports.InstanceBootstrapService = (*Service)(nil)

func (s *Service) Plan(
	ctx context.Context,
	inputs *instancebootstrap.Inputs,
) (*instancebootstrapports.InstanceBootstrapPlan, error) {
	normalized, err := validateInputs(inputs)
	if err != nil {
		return nil, err
	}

	state, err := s.repo.GetState(ctx)
	if err != nil {
		return nil, err
	}

	status, err := decide(state, &normalized)
	if err != nil {
		return nil, err
	}

	return &instancebootstrapports.InstanceBootstrapPlan{
		Status: status,
		Inputs: normalized,
		Record: state.Record,
	}, nil
}

func (s *Service) Bootstrap(
	ctx context.Context,
	req *instancebootstrapports.InstanceBootstrapRequest,
) (*instancebootstrapports.InstanceBootstrapResult, error) {
	if req == nil {
		return nil, errortypes.NewValidationError(
			instancebootstrap.FieldOrganizationName,
			errortypes.ErrRequired,
			"Bootstrap inputs are required",
		)
	}

	inputs, err := validateRequest(req)
	if err != nil {
		return nil, err
	}

	var out *created
	sysCtx := dbscope.WithSystem(ctx, bootstrapScopeReason)
	err = s.db.Transact(
		sysCtx,
		kernelports.TxOptions{LockTimeout: lockTimeout},
		func(txCtx context.Context) error {
			if lockErr := s.tenants.LockProvisioning(txCtx); lockErr != nil {
				return lockErr
			}

			state, stateErr := s.repo.GetState(txCtx)
			if stateErr != nil {
				return stateErr
			}

			status, decideErr := decide(state, &inputs)
			if decideErr != nil {
				return decideErr
			}
			if status == instancebootstrapports.InstanceBootstrapCompleted {
				out = &created{record: state.Record}
				return nil
			}

			var createErr error
			out, createErr = s.create(txCtx, &inputs, req)
			return createErr
		},
	)
	if err != nil {
		return nil, err
	}

	if out.tenant == nil {
		return completedResult(&inputs, out.record), nil
	}

	s.audit(ctx, out)
	s.l.Info("instance bootstrapped",
		zap.String("organizationId", out.tenant.Organization.ID.String()),
		zap.String("businessUnitId", out.tenant.BusinessUnit.ID.String()),
		zap.String("userId", out.tenant.Owner.ID.String()),
		zap.Bool("systemUserCreated", out.finalized.SystemUserCreated),
	)

	return &instancebootstrapports.InstanceBootstrapResult{
		Status:            instancebootstrapports.InstanceBootstrapCreated,
		OrganizationID:    out.tenant.Organization.ID,
		BusinessUnitID:    out.tenant.BusinessUnit.ID,
		AdminUserID:       out.tenant.Owner.ID,
		AdminUsername:     out.tenant.Owner.Username,
		AdminEmail:        out.tenant.Owner.EmailAddress,
		LoginSlug:         out.tenant.Organization.LoginSlug,
		SystemUserCreated: out.finalized.SystemUserCreated,
		CompletedAt:       out.record.CompletedAt,
	}, nil
}

type created struct {
	tenant    *tenantports.BootstrapTenantResult
	finalized *instancebootstrapports.FinalizeInstanceBootstrapResult
	record    *instancebootstrap.InstanceBootstrap
}

func (s *Service) create(
	ctx context.Context,
	inputs *instancebootstrap.Inputs,
	req *instancebootstrapports.InstanceBootstrapRequest,
) (*created, error) {
	now := s.now()
	owner := &tenant.User{
		Name:               inputs.AdminName,
		EmailAddress:       inputs.AdminEmail,
		Status:             domaintypes.StatusActive,
		Timezone:           inputs.Timezone,
		Locale:             defaultLocale,
		TimeFormat:         domaintypes.TimeFormat12Hour,
		MustChangePassword: false,
	}
	if err := setAdminPassword(owner, req, now); err != nil {
		return nil, err
	}

	result, err := s.tenants.Bootstrap(ctx, &tenantports.BootstrapTenantRequest{
		BusinessUnitName:  inputs.OrganizationName,
		Organization:      organizationFor(inputs),
		LoginSlugBase:     stringutils.SlugifyASCII(inputs.OrganizationName, loginSlugBaseLength),
		StateAbbreviation: inputs.State,
		Owner:             owner,
		UsernameBase:      tenant.UsernameBase(inputs.AdminEmail),
		Now:               now,
	})
	if err != nil {
		return nil, err
	}

	finalized, err := s.repo.Finalize(ctx, &instancebootstrapports.FinalizeInstanceBootstrapRequest{
		OrganizationID:     result.Organization.ID,
		BusinessUnitID:     result.BusinessUnit.ID,
		AdminUserID:        result.Owner.ID,
		Inputs:             *inputs,
		SystemUserPassword: s.systemUserPassword,
		Now:                now,
	})
	if err != nil {
		return nil, err
	}

	return &created{tenant: result, finalized: finalized, record: finalized.Record}, nil
}

func setAdminPassword(
	owner *tenant.User,
	req *instancebootstrapports.InstanceBootstrapRequest,
	now int64,
) error {
	if req.AdminSetup == instancebootstrap.AdminSetupLink {
		hashed, err := owner.GenerateUnusablePassword()
		if err != nil {
			return fmt.Errorf("hash an unusable administrator password: %w", err)
		}
		owner.Password = hashed
		return nil
	}

	hashed, err := owner.GeneratePassword(req.Password)
	if err != nil {
		return fmt.Errorf("hash administrator password: %w", err)
	}
	owner.Password = hashed
	owner.PasswordChangedAt = &now
	return nil
}

func organizationFor(inputs *instancebootstrap.Inputs) *tenant.Organization {
	return &tenant.Organization{
		Name:                   inputs.OrganizationName,
		ScacCode:               inputs.SCAC,
		DOTNumber:              inputs.DOTNumber,
		AddressLine1:           inputs.AddressLine1,
		City:                   inputs.City,
		PostalCode:             inputs.PostalCode,
		Timezone:               inputs.Timezone,
		Locale:                 defaultLocale,
		BrokerageEnabled:       true,
		AssetOperationsEnabled: true,
	}
}

func (s *Service) audit(ctx context.Context, out *created) {
	org := out.tenant.Organization
	owner := out.tenant.Owner
	actor := permissionports.SystemAuditActor()
	metadata := map[string]any{
		"source":              auditSource,
		"instanceBootstrapId": out.record.ID.String(),
	}

	s.auditor.RecordChange(ctx, &permissionports.SecurityChange{
		Resource:       permission.ResourceUser,
		ResourceID:     owner.ID.String(),
		Operation:      permission.OpCreate,
		Actor:          actor,
		OrganizationID: org.ID,
		BusinessUnitID: org.BusinessUnitID,
		After: map[string]any{
			keyUserID:         owner.ID.String(),
			"username":        owner.Username,
			"emailAddress":    owner.EmailAddress,
			"name":            owner.Name,
			"status":          string(owner.Status),
			keyOrganizationID: org.ID.String(),
			"membership":      "default",
		},
		Comment:  "Instance bootstrap created the first administrator",
		Metadata: metadata,
	})

	s.auditor.RecordChange(ctx, &permissionports.SecurityChange{
		Resource:       permission.ResourceRole,
		ResourceID:     out.tenant.AdminRoleID.String(),
		Operation:      permission.OpCreate,
		Actor:          actor,
		OrganizationID: org.ID,
		BusinessUnitID: org.BusinessUnitID,
		After: map[string]any{
			keyRoleID:         out.tenant.AdminRoleID.String(),
			"name":            "Organization Administrator",
			keyOrganizationID: org.ID.String(),
		},
		Comment:  "Instance bootstrap created the Organization Administrator role",
		Metadata: metadata,
	})

	s.auditor.RecordChange(ctx, &permissionports.SecurityChange{
		Resource:       permission.ResourceRole,
		ResourceID:     out.tenant.AdminRoleID.String(),
		Operation:      permission.OpAssign,
		Actor:          actor,
		OrganizationID: org.ID,
		BusinessUnitID: org.BusinessUnitID,
		After: map[string]any{
			keyUserID:         owner.ID.String(),
			keyRoleID:         out.tenant.AdminRoleID.String(),
			keyOrganizationID: org.ID.String(),
		},
		Comment:  "Instance bootstrap granted the first administrator the Organization Administrator role",
		Metadata: metadata,
	})

	if !out.finalized.SystemUserCreated {
		return
	}

	s.auditor.RecordChange(ctx, &permissionports.SecurityChange{
		Resource:       permission.ResourceUser,
		ResourceID:     out.finalized.SystemUserID.String(),
		Operation:      permission.OpCreate,
		Actor:          actor,
		OrganizationID: org.ID,
		BusinessUnitID: org.BusinessUnitID,
		After: map[string]any{
			keyUserID:         out.finalized.SystemUserID.String(),
			"username":        tenant.SystemUsername,
			keyOrganizationID: org.ID.String(),
		},
		Comment:  "Instance bootstrap created the system account background work runs as",
		Metadata: metadata,
	})
}

func completedResult(
	inputs *instancebootstrap.Inputs,
	record *instancebootstrap.InstanceBootstrap,
) *instancebootstrapports.InstanceBootstrapResult {
	return &instancebootstrapports.InstanceBootstrapResult{
		Status:         instancebootstrapports.InstanceBootstrapCompleted,
		OrganizationID: record.OrganizationID,
		BusinessUnitID: record.BusinessUnitID,
		AdminUserID:    record.AdminUserID,
		AdminEmail:     inputs.AdminEmail,
		CompletedAt:    record.CompletedAt,
	}
}

func decide(
	state *instancebootstrapports.InstanceBootstrapState,
	inputs *instancebootstrap.Inputs,
) (instancebootstrapports.InstanceBootstrapStatus, error) {
	if state.Record != nil {
		differences := state.Record.Inputs.IdentityDifferences(inputs)
		if len(differences) == 0 {
			return instancebootstrapports.InstanceBootstrapCompleted, nil
		}

		return "", fmt.Errorf(
			"%w: %s differ from the completed bootstrap; restore the original values "+
				"or stop running the bootstrap, and change the organization and its "+
				"administrators in the application instead",
			ErrInputsDiffer,
			strings.Join(differences, ", "),
		)
	}

	if state.UserCount > 0 {
		return "", fmt.Errorf(
			"%w: %d user(s) already exist, so this database was set up another way; "+
				"sign in with an existing administrator and add users in the application",
			ErrAlreadyInitialized,
			state.UserCount,
		)
	}

	return instancebootstrapports.InstanceBootstrapReady, nil
}

func validateInputs(inputs *instancebootstrap.Inputs) (instancebootstrap.Inputs, error) {
	normalized := inputs.Normalize()

	multiErr := errortypes.NewMultiError()
	normalized.Validate(multiErr)
	if multiErr.HasErrors() {
		return normalized, multiErr.Err()
	}

	return normalized, nil
}

func validateRequest(
	req *instancebootstrapports.InstanceBootstrapRequest,
) (instancebootstrap.Inputs, error) {
	normalized := req.Inputs.Normalize()

	multiErr := errortypes.NewMultiError()
	normalized.Validate(multiErr)

	switch req.AdminSetup {
	case instancebootstrap.AdminSetupLink:
		if req.Password != "" {
			multiErr.Add(
				fieldPassword,
				errortypes.ErrInvalid,
				"No password is set when the administrator sets up their account from a link",
			)
		}
	case instancebootstrap.AdminSetupPassword, "":
		tenant.ValidateNewPassword(multiErr, fieldPassword, req.Password, normalized.AdminEmail)
	default:
		multiErr.Add(
			fieldAdminSetup,
			errortypes.ErrInvalid,
			"Administrator setup must be password or link",
		)
	}

	if multiErr.HasErrors() {
		return normalized, multiErr.Err()
	}

	return normalized, nil
}
