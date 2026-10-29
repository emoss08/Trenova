package userservice

import (
	"context"
	"errors"
	"strings"

	"github.com/emoss08/trenova/internal/modules/identity/app/passwordresetservice"
	"github.com/emoss08/trenova/internal/modules/identity/app/roleservice"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/rbac"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/workspace/app/auditservice"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/audit"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/realtimeinvalidation"
	"github.com/emoss08/trenova/shared/i18n"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/mailtext"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/sliceutils"
	"github.com/emoss08/trenova/shared/tokenutils"
	"go.uber.org/zap"
)

const roleIDsField = "roleIds"

type CreateUserRequest struct {
	TenantInfo pagination.TenantInfo
	User       *tenant.User
	RoleIDs    []pulid.ID
}

func (s *Service) Create(ctx context.Context, req *CreateUserRequest) (*tenant.User, error) {
	log := s.l.With(
		zap.String("operation", "Create"),
		zap.String("actorID", req.TenantInfo.UserID.String()),
	)

	entity := req.User
	if entity == nil {
		return nil, errortypes.NewBusinessError("There is no user to create.")
	}

	if entity.ID.IsNotNil() {
		return nil, errortypes.NewValidationError(
			"id",
			errortypes.ErrInvalid,
			"ID must not be set on create",
		)
	}

	prepareNewUser(entity, req.TenantInfo)

	if multiErr := s.validator.ValidateCreate(ctx, entity); multiErr != nil {
		return nil, multiErr
	}

	password, err := unusablePassword(entity)
	if err != nil {
		log.Error("failed to seal the new user's password", zap.Error(err))
		return nil, err
	}
	entity.Password = password

	roleIDs := sliceutils.Dedupe(req.RoleIDs)

	var created *tenant.User
	if err = s.db.Transact(
		ctx,
		kernelports.TxOptions{},
		func(txCtx context.Context) error {
			created, err = s.repo.Create(txCtx, tenantports.CreateUserRequest{
				User: entity,
				Membership: &tenant.OrganizationMembership{
					BusinessUnitID: req.TenantInfo.BuID,
					OrganizationID: req.TenantInfo.OrgID,
					GrantedByID:    req.TenantInfo.UserID,
					IsDefault:      true,
				},
			})
			if err != nil {
				return err
			}

			return s.assignRoles(txCtx, req.TenantInfo, created.ID, roleIDs)
		},
	); err != nil {
		log.Error("failed to create user", zap.Error(err))
		return nil, err
	}

	inviteSent := s.sendInvite(ctx, req.TenantInfo, created)

	s.recordCreation(ctx, &creationRecord{
		tenantInfo: req.TenantInfo,
		user:       created,
		roleIDs:    roleIDs,
		inviteSent: inviteSent,
	})

	if err = realtimeinvalidation.Publish(ctx, s.realtime, &realtimeinvalidation.PublishParams{
		OrganizationID: req.TenantInfo.OrgID,
		BusinessUnitID: req.TenantInfo.BuID,
		ActorUserID:    req.TenantInfo.UserID,
		Resource:       permission.ResourceUser.String(),
		Action:         "created",
		RecordID:       created.ID,
		Entity:         created,
	}); err != nil {
		log.Warn("failed to publish user invalidation", zap.Error(err))
	}

	return created, nil
}

func prepareNewUser(entity *tenant.User, tenantInfo pagination.TenantInfo) {
	entity.CurrentOrganizationID = tenantInfo.OrgID
	entity.BusinessUnitID = tenantInfo.BuID
	entity.Name = strings.TrimSpace(entity.Name)
	entity.Username = strings.TrimSpace(entity.Username)
	entity.EmailAddress = mailtext.NormalizeEmailAddress(entity.EmailAddress)
	entity.Timezone = strings.TrimSpace(entity.Timezone)
	entity.LastLoginAt = nil
	entity.PasswordChangedAt = nil
	entity.IsLocked = false
	entity.Version = 0

	if entity.Status == "" {
		entity.Status = domaintypes.StatusActive
	}
	if entity.TimeFormat == "" {
		entity.TimeFormat = domaintypes.TimeFormat12Hour
	}
	if entity.Locale == "" {
		entity.Locale = string(i18n.Default)
	}
	if entity.AvatarStyle == "" {
		entity.AvatarStyle = tenant.AvatarStyleAuto
	}
	if entity.AvailabilityStatus == "" {
		entity.AvailabilityStatus = tenant.AvailabilityAvailable
	}
}

func unusablePassword(entity *tenant.User) (string, error) {
	secret, _, err := tokenutils.New()
	if err != nil {
		return "", err
	}

	return entity.GeneratePassword(secret)
}

func (s *Service) assignRoles(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	userID pulid.ID,
	roleIDs []pulid.ID,
) error {
	for _, roleID := range roleIDs {
		if err := s.roles.AssignRole(ctx, roleservice.AssignRoleRequest{
			ActorID:        tenantInfo.UserID,
			OrganizationID: tenantInfo.OrgID,
			Assignment: &rbac.UserRoleAssignment{
				UserID: userID,
				RoleID: roleID,
			},
		}); err != nil {
			return roleAssignmentError(err)
		}
	}

	return nil
}

func roleAssignmentError(err error) error {
	multiErr := errortypes.NewMultiError()

	if source, ok := errors.AsType[*errortypes.MultiError](err); ok {
		for _, item := range source.Errors {
			multiErr.Add(roleIDsField, item.Code, item.Message, item.Args...)
		}
		return multiErr
	}

	switch {
	case errortypes.IsNotFoundError(err):
		multiErr.Add(
			roleIDsField,
			errortypes.ErrInvalidReference,
			"One of the selected roles was not found",
		)
	case errors.Is(err, roleservice.ErrRoleAlreadyAssigned):
		multiErr.Add(roleIDsField, errortypes.ErrDuplicate, "A role was selected more than once")
	default:
		return err
	}

	return multiErr
}

func (s *Service) sendInvite(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	user *tenant.User,
) bool {
	if s.invites == nil || !user.IsActive() {
		return false
	}

	if err := s.invites.SendInvite(ctx, passwordresetservice.InviteRequest{
		User:  user,
		Actor: tenantInfo,
	}); err != nil {
		s.l.Error("failed to send the new user's invitation",
			zap.String("userID", user.ID.String()),
			zap.Error(err))
		return false
	}

	return true
}

type creationRecord struct {
	tenantInfo pagination.TenantInfo
	user       *tenant.User
	roleIDs    []pulid.ID
	inviteSent bool
}

func (s *Service) recordCreation(ctx context.Context, record *creationRecord) {
	roleIDs := sliceutils.Strings(record.roleIDs)

	if s.auditor != nil {
		s.auditor.RecordChange(ctx, &permissionports.SecurityChange{
			Resource:       permission.ResourceUser,
			ResourceID:     record.user.ID.String(),
			Operation:      permission.OpCreate,
			Actor:          permissionports.UserActor(record.tenantInfo).AuditActor(),
			OrganizationID: record.tenantInfo.OrgID,
			BusinessUnitID: record.tenantInfo.BuID,
			After: map[string]any{
				"user":             record.user,
				"roleIds":          roleIDs,
				organizationIDsKey: []string{record.tenantInfo.OrgID.String()},
			},
			Comment: "User created",
			Metadata: map[string]any{
				"inviteSent":   record.inviteSent,
				"emailAddress": record.user.EmailAddress,
			},
		})
	}

	if err := s.auditService.LogAction(&auditports.LogActionParams{
		Resource:       permission.ResourceUser,
		ResourceID:     record.user.ID.String(),
		Operation:      permission.OpCreate,
		UserID:         record.tenantInfo.UserID,
		CurrentState:   jsonutils.MustToJSON(record.user),
		OrganizationID: record.tenantInfo.OrgID,
		BusinessUnitID: record.tenantInfo.BuID,
	},
		auditservice.WithComment("User created"),
		auditservice.WithCategory(audit.CategoryUser),
		auditservice.WithRequest(ctx),
	); err != nil {
		s.l.Error("failed to log user creation", zap.Error(err))
	}
}
