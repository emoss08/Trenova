package instancebootstrapservice

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/instancebootstrap"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/emailutils"
	"github.com/emoss08/trenova/shared/passwordutils"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	bootstrapScopeReason = "create the first organization and administrator of an instance before any tenant exists"
	lockTimeout          = 10 * time.Second
	defaultLocale        = "en"
	loginSlugBaseLength  = 60
	maxUsernameLength    = 20
	auditSource          = "instance_bootstrap"
	fieldPassword        = "password"
	keyUserID            = "userId"
	keyRoleID            = "roleId"
	keyOrganizationID    = "organizationId"
)

var (
	ErrAlreadyInitialized = errors.New("instance already has users")
	ErrInputsDiffer       = errors.New("instance was bootstrapped with different inputs")
)

type transactor interface {
	WithTx(ctx context.Context, opts ports.TxOptions, fn func(context.Context, bun.Tx) error) error
}

type Params struct {
	fx.In

	DB      ports.DBConnection
	Tenants repositories.TenantBootstrapRepository
	Repo    repositories.InstanceBootstrapRepository
	Auditor services.SecurityAuditor
	Config  *config.Config
	Logger  *zap.Logger
}

type Service struct {
	db                 transactor
	tenants            repositories.TenantBootstrapRepository
	repo               repositories.InstanceBootstrapRepository
	auditor            services.SecurityAuditor
	systemUserPassword string
	now                func() int64
	l                  *zap.Logger
}

//nolint:gocritic // this is dependency injection
func New(p Params) *Service {
	var systemUserPassword string
	if p.Config != nil {
		systemUserPassword = p.Config.System.SystemUserPassword
	}

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

var _ services.InstanceBootstrapService = (*Service)(nil)

func (s *Service) Plan(
	ctx context.Context,
	inputs *instancebootstrap.Inputs,
) (*services.InstanceBootstrapPlan, error) {
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

	return &services.InstanceBootstrapPlan{
		Status: status,
		Inputs: normalized,
		Record: state.Record,
	}, nil
}

func (s *Service) Bootstrap(
	ctx context.Context,
	req *services.InstanceBootstrapRequest,
) (*services.InstanceBootstrapResult, error) {
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
	err = s.db.WithTx(sysCtx, ports.TxOptions{LockTimeout: lockTimeout}, func(
		txCtx context.Context,
		_ bun.Tx,
	) error {
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
		if status == services.InstanceBootstrapCompleted {
			out = &created{record: state.Record}
			return nil
		}

		var createErr error
		out, createErr = s.create(txCtx, &inputs, req.Password)
		return createErr
	})
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

	return &services.InstanceBootstrapResult{
		Status:            services.InstanceBootstrapCreated,
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
	tenant    *repositories.BootstrapTenantResult
	finalized *repositories.FinalizeInstanceBootstrapResult
	record    *instancebootstrap.InstanceBootstrap
}

func (s *Service) create(
	ctx context.Context,
	inputs *instancebootstrap.Inputs,
	password string,
) (*created, error) {
	owner := &tenant.User{
		Name:               inputs.AdminName,
		EmailAddress:       inputs.AdminEmail,
		Status:             domaintypes.StatusActive,
		Timezone:           inputs.Timezone,
		Locale:             defaultLocale,
		TimeFormat:         domaintypes.TimeFormat12Hour,
		MustChangePassword: false,
	}
	hashed, err := owner.GeneratePassword(password)
	if err != nil {
		return nil, fmt.Errorf("hash administrator password: %w", err)
	}
	owner.Password = hashed

	now := s.now()
	result, err := s.tenants.Bootstrap(ctx, &repositories.BootstrapTenantRequest{
		BusinessUnitName:  inputs.OrganizationName,
		Organization:      organizationFor(inputs),
		LoginSlugBase:     stringutils.SlugifyASCII(inputs.OrganizationName, loginSlugBaseLength),
		StateAbbreviation: inputs.State,
		Owner:             owner,
		UsernameBase:      usernameBase(inputs.AdminEmail),
		Now:               now,
	})
	if err != nil {
		return nil, err
	}

	finalized, err := s.repo.Finalize(ctx, &repositories.FinalizeInstanceBootstrapRequest{
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
	if s.auditor == nil {
		return
	}

	org := out.tenant.Organization
	owner := out.tenant.Owner
	actor := services.SystemAuditActor()
	metadata := map[string]any{
		"source":              auditSource,
		"instanceBootstrapId": out.record.ID.String(),
	}

	s.auditor.RecordChange(ctx, &services.SecurityChange{
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

	s.auditor.RecordChange(ctx, &services.SecurityChange{
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

	s.auditor.RecordChange(ctx, &services.SecurityChange{
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

	s.auditor.RecordChange(ctx, &services.SecurityChange{
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
) *services.InstanceBootstrapResult {
	return &services.InstanceBootstrapResult{
		Status:         services.InstanceBootstrapCompleted,
		OrganizationID: record.OrganizationID,
		BusinessUnitID: record.BusinessUnitID,
		AdminUserID:    record.AdminUserID,
		AdminEmail:     inputs.AdminEmail,
		CompletedAt:    record.CompletedAt,
	}
}

func decide(
	state *repositories.InstanceBootstrapState,
	inputs *instancebootstrap.Inputs,
) (services.InstanceBootstrapStatus, error) {
	if state.Record != nil {
		differences := state.Record.Inputs.IdentityDifferences(inputs)
		if len(differences) == 0 {
			return services.InstanceBootstrapCompleted, nil
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

	return services.InstanceBootstrapReady, nil
}

func validateInputs(inputs *instancebootstrap.Inputs) (instancebootstrap.Inputs, error) {
	normalized := inputs.Normalize()

	multiErr := errortypes.NewMultiError()
	normalized.Validate(multiErr)
	if multiErr.HasErrors() {
		return normalized, multiErr
	}

	return normalized, nil
}

func validateRequest(req *services.InstanceBootstrapRequest) (instancebootstrap.Inputs, error) {
	normalized := req.Inputs.Normalize()

	multiErr := errortypes.NewMultiError()
	normalized.Validate(multiErr)

	address, _ := emailutils.Parse(normalized.AdminEmail)
	if err := passwordutils.CheckPolicy(req.Password, &address); err != nil {
		multiErr.Add(fieldPassword, errortypes.ErrInvalid, passwordMessage(err))
	}

	if multiErr.HasErrors() {
		return normalized, multiErr
	}

	return normalized, nil
}

func passwordMessage(err error) string {
	switch {
	case errors.Is(err, passwordutils.ErrTooShort):
		return "Password must be at least 12 characters"
	case errors.Is(err, passwordutils.ErrTooLong):
		return "Password must be 72 bytes or fewer"
	case errors.Is(err, passwordutils.ErrBlank):
		return "Password cannot be only spaces"
	case errors.Is(err, passwordutils.ErrContainsEmail):
		return "Password must not contain the administrator's email address"
	case errors.Is(err, passwordutils.ErrCommon):
		return "This password is too common. Choose something harder to guess"
	default:
		return "Password does not meet the password policy"
	}
}

func usernameBase(email string) string {
	local := emailutils.LocalPart(email)

	var builder strings.Builder
	builder.Grow(min(len(local), maxUsernameLength))
	for _, r := range local {
		if builder.Len() >= maxUsernameLength {
			break
		}
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '.', r == '_', r == '-':
			builder.WriteRune(r)
		}
	}

	return strings.Trim(builder.String(), "._-")
}
